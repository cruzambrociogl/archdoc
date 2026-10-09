package archdoc

import (
	"bytes"
	"encoding/json"
	"strings"
)

// inlineWidth is how long a value may be and still be written on one line.
const inlineWidth = 140

// JSON encodes a model the way .archdoc/model.json holds it: indented, so a change to it reads
// as a diff, with every value that fits on one line written on one — a citation is a line, not
// five. The same model always encodes to the same bytes (AC-7).
func (m Model) JSON() ([]byte, error) {
	compact, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	var b bytes.Buffer
	if err := layOut(&b, compact, 0, 0); err != nil {
		return nil, err
	}
	b.WriteByte('\n')
	return b.Bytes(), nil
}

// layOut writes one JSON value at a depth, lead characters into its line: on one line when it
// fits, otherwise a member per line.
func layOut(b *bytes.Buffer, raw json.RawMessage, depth, lead int) error {
	if len(raw) == 0 || (raw[0] != '{' && raw[0] != '[') {
		b.Write(raw)
		return nil
	}
	keys, values, err := members(raw)
	if err != nil {
		return err
	}
	open, shut := raw[0], raw[len(raw)-1]
	if len(values) == 0 {
		b.WriteByte(open)
		b.WriteByte(shut)
		return nil
	}
	if line, ok := inline(raw); ok && lead+len(line) < inlineWidth {
		b.WriteString(line)
		return nil
	}
	pad := strings.Repeat("  ", depth+1)
	b.WriteByte(open)
	for i, v := range values {
		b.WriteByte('\n')
		b.WriteString(pad)
		lead := len(pad)
		if keys != nil {
			b.Write(keys[i])
			b.WriteString(": ")
			lead += len(keys[i]) + 2
		}
		if err := layOut(b, v, depth+1, lead); err != nil {
			return err
		}
		if i < len(values)-1 {
			b.WriteByte(',')
		}
	}
	b.WriteByte('\n')
	b.WriteString(pad[2:])
	b.WriteByte(shut)
	return nil
}

// inline is a value on one line, spaced as a person would write it. ok is false once it is
// already too long to fit anywhere.
func inline(raw json.RawMessage) (string, bool) {
	if len(raw) > 2*inlineWidth {
		return "", false
	}
	if raw[0] != '{' && raw[0] != '[' {
		return string(raw), true
	}
	keys, values, err := members(raw)
	if err != nil {
		return "", false
	}
	var s strings.Builder
	s.WriteByte(raw[0])
	for i, v := range values {
		if i > 0 {
			s.WriteString(", ")
		}
		if keys != nil {
			s.Write(keys[i])
			s.WriteString(": ")
		}
		part, ok := inline(v)
		if !ok {
			return "", false
		}
		s.WriteString(part)
	}
	s.WriteByte(raw[len(raw)-1])
	return s.String(), true
}

// members are an object's keys and values, or an array's values, in the order written — which
// is the order the encoder gave them, so nothing here iterates a map.
func members(raw json.RawMessage) (keys []json.RawMessage, values []json.RawMessage, err error) {
	if raw[0] == '[' {
		err = json.Unmarshal(raw, &values)
		return nil, values, err
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	if _, err := dec.Token(); err != nil {
		return nil, nil, err
	}
	keys = []json.RawMessage{}
	for dec.More() {
		var k, v json.RawMessage
		if err := dec.Decode(&k); err != nil {
			return nil, nil, err
		}
		if err := dec.Decode(&v); err != nil {
			return nil, nil, err
		}
		keys = append(keys, k)
		values = append(values, v)
	}
	return keys, values, nil
}
