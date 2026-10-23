package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/cruzambrociogl/archdoc"
)

// mcp answers a coding agent's questions about the architecture over the Model Context Protocol
// (platform-surface §5.7, "Connect your agent"): JSON-RPC 2.0, one message per line on stdin and
// stdout. The agent starts it in the repository it works in, and asks what show, status and the
// app's Impact answer — every fact cited, read from the committed model. It writes nothing, asks no
// model and opens no connection: the agent's own model does the thinking.
//
// Written by hand, not with an SDK: an agent's stdio server needs four methods, and the SDKs carry
// HTTP transports that would put an HTTP client outside internal/semantic (decisions, 23 Oct).
func mcpCommand(e env, args []string) error {
	fs := flags("mcp")
	positional, err := parse(e, fs, args, nil)
	if err != nil {
		return err
	}
	root, err := readPath(fs, positional)
	if err != nil {
		return err
	}
	// Stdout is the protocol's: anything printed there by accident would break the agent's reading
	// of it. What the commands print goes to a buffer; whatever else writes to os.Stdout lands on
	// stderr, which an agent shows as the server's log.
	protocol := os.Stdout
	os.Stdout = os.Stderr
	defer func() { os.Stdout = protocol }()
	return serveMCP(os.Stdin, protocol, root)
}

// The protocol versions this server speaks. Tools are the same in each; an agent asking for one
// of these gets it, any other the newest.
var mcpVersions = []string{"2025-11-25", "2025-06-18", "2025-03-26", "2024-11-05"}

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"` // absent on a notification, which is never answered
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

// JSON-RPC's own error codes.
const (
	rpcParse          = -32700
	rpcInvalidRequest = -32600
	rpcNoMethod       = -32601
	rpcInvalidParams  = -32602
)

// serveMCP answers each request in turn until the agent closes stdin.
func serveMCP(in io.Reader, out io.Writer, root string) error {
	enc := json.NewEncoder(out) // one line per message: Encode ends each with a newline, and escapes any inside
	enc.SetEscapeHTML(false)
	null := json.RawMessage("null")
	sc := bufio.NewScanner(in)
	sc.Buffer(make([]byte, 0, 64<<10), 16<<20)
	for sc.Scan() {
		line := bytes.TrimSpace(sc.Bytes())
		if len(line) == 0 {
			continue
		}
		var req rpcRequest
		if err := json.Unmarshal(line, &req); err != nil {
			if err := enc.Encode(rpcResponse{JSONRPC: "2.0", ID: null, Error: &rpcError{rpcParse, "not a JSON-RPC message: " + err.Error()}}); err != nil {
				return err
			}
			continue
		}
		var result any
		var rerr *rpcError
		if req.JSONRPC != "2.0" || req.Method == "" {
			rerr = &rpcError{rpcInvalidRequest, `a request needs "jsonrpc": "2.0" and a method`}
		} else {
			result, rerr = handleMCP(root, req)
		}
		if req.ID == nil {
			continue
		}
		if err := enc.Encode(rpcResponse{JSONRPC: "2.0", ID: req.ID, Result: result, Error: rerr}); err != nil {
			return err
		}
	}
	return sc.Err()
}

func handleMCP(root string, req rpcRequest) (any, *rpcError) {
	switch req.Method {
	case "initialize":
		var p struct {
			ProtocolVersion string `json:"protocolVersion"`
		}
		json.Unmarshal(req.Params, &p)
		version := mcpVersions[0]
		for _, v := range mcpVersions {
			if v == p.ProtocolVersion {
				version = v
			}
		}
		b := archdoc.Build()
		return map[string]any{
			"protocolVersion": version,
			"capabilities":    map[string]any{"tools": map[string]any{}},
			"serverInfo":      map[string]any{"name": "archdoc", "version": b.Version},
			"instructions":    mcpInstructions(root),
		}, nil
	case "ping":
		return struct{}{}, nil
	case "tools/list":
		return map[string]any{"tools": mcpTools}, nil
	case "tools/call":
		var p struct {
			Name      string         `json:"name"`
			Arguments map[string]any `json:"arguments"`
		}
		if err := json.Unmarshal(req.Params, &p); err != nil {
			return nil, &rpcError{rpcInvalidParams, "tools/call needs a name and arguments: " + err.Error()}
		}
		for _, t := range mcpTools {
			if t.Name == p.Name {
				return callTool(t, root, p.Arguments)
			}
		}
		return nil, &rpcError{rpcInvalidParams, fmt.Sprintf("no tool %q — tools/list names them", p.Name)}
	}
	if strings.HasPrefix(req.Method, "notifications/") {
		return nil, nil
	}
	return nil, &rpcError{rpcNoMethod, fmt.Sprintf("archdoc does not answer %q", req.Method)}
}

// callTool runs a tool and says what it printed. A failure the agent can act on — nothing
// documented yet, a name that matches nothing — is the tool's answer, marked as an error, not the
// protocol's: the agent reads it and asks again.
func callTool(t mcpTool, root string, raw map[string]any) (any, *rpcError) {
	args := map[string]string{}
	for k, v := range raw {
		s, ok := v.(string)
		if !ok {
			return nil, &rpcError{rpcInvalidParams, fmt.Sprintf("%s: %q must be a string", t.Name, k)}
		}
		args[k] = strings.TrimSpace(s)
	}
	for _, k := range t.required() {
		if args[k] == "" {
			return nil, &rpcError{rpcInvalidParams, fmt.Sprintf("%s needs %q", t.Name, k)}
		}
	}
	var buf bytes.Buffer
	err := t.run(env{out: &buf}, root, args)
	text := strings.TrimRight(buf.String(), "\n")
	if err != nil {
		if text != "" {
			text += "\n\n"
		}
		text += err.Error()
	}
	return map[string]any{
		"content": []map[string]any{{"type": "text", "text": text}},
		"isError": err != nil,
	}, nil
}

func mcpInstructions(root string) string {
	return "archdoc answers from the architecture documentation of the repository at " + absolute(root) + ", read from its code " +
		"and configuration — not written by a model. Every fact cites the file and line that proves it (path:line); a sentence " +
		"marked ◇ was written by a model from facts it cites. Start with overview; show follows one element, route or file; " +
		"impact says what a change to something touches. The documentation is as of its last 'archdoc generate': status says " +
		"whether the code has moved since, and running 'archdoc generate <path>' in a terminal brings it up to date."
}
