package serve

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cruzambrociogl/archdoc/internal/core"
	"github.com/cruzambrociogl/archdoc/internal/recent"
)

// One archdoc per machine (decisions, 18 Nov 2026). A bare `archdoc` serves the library — every
// place this machine remembers, Home's handler — and records where in the user's configuration
// folder; a second `archdoc`, in another project, finds it there and opens the project in it instead
// of failing on a taken port. `archdoc restart` stops it and starts it again from the shell it is run
// in, so a key set since is read.
//
// The requests below go to this machine's own archdoc and nowhere else: always 127.0.0.1, never a
// name, so no resolver is asked and nothing leaves the machine. This package is the one the network
// boundary lets serve; reaching outward stays internal/semantic's alone.

// MachinePort is where the library is served when the port is free.
const MachinePort = 7474

// ErrHandedOver ends a library that another archdoc asked to stop — `archdoc restart`.
var ErrHandedOver = errors.New("stopped by archdoc restart")

// Machine is the archdoc serving this machine's library, as it recorded itself.
type Machine struct {
	Port  int       `json:"port"`
	PID   int       `json:"pid"`
	Build string    `json:"build"`
	Since time.Time `json:"since"`
}

// URL is the library's address in a browser.
func (m Machine) URL() string { return fmt.Sprintf("http://localhost:%d", m.Port) }

// loopback is the address requests go to: the number, so nothing is looked up.
func (m Machine) loopback() string { return fmt.Sprintf("http://127.0.0.1:%d", m.Port) }

// MachineFile is where the running library records itself. A variable, so a test keeps it out of
// the user's own folder.
var MachineFile = func() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "archdoc", "running.json"), nil
}

// local is a client for this machine's archdoc: short waits, no proxy.
var local = &http.Client{Timeout: 2 * time.Second, Transport: &http.Transport{Proxy: nil}}

// Running is the archdoc serving this machine's library, if one answers where it recorded itself —
// as itself: its session says it is the library, with the process id it recorded.
func Running() (Machine, bool) {
	file, err := MachineFile()
	if err != nil {
		return Machine{}, false
	}
	b, err := os.ReadFile(file)
	if err != nil {
		return Machine{}, false
	}
	var m Machine
	if json.Unmarshal(b, &m) != nil || m.Port <= 0 {
		return Machine{}, false
	}
	s, err := m.session()
	if err != nil || s.Mode != "home" || s.PID != m.PID {
		return Machine{}, false
	}
	return m, true
}

type machineSession struct {
	Token string `json:"token"`
	Mode  string `json:"mode"`
	PID   int    `json:"pid"`
}

func (m Machine) session() (machineSession, error) {
	resp, err := local.Get(m.loopback() + "/api/session")
	if err != nil {
		return machineSession{}, err
	}
	defer resp.Body.Close()
	var s machineSession
	if resp.StatusCode != http.StatusOK {
		return s, fmt.Errorf("%s answered %s", m.URL(), resp.Status)
	}
	err = json.NewDecoder(resp.Body).Decode(&s)
	return s, err
}

// Stop asks the running library to stop, as its own page would ask for an action, and waits until
// its port is free for the one that replaces it.
func (m Machine) Stop(ctx context.Context) error {
	s, err := m.session()
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, m.loopback()+"/api/machine/stop", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Origin", m.loopback())
	req.Header.Set(TokenHeader, s.Token)
	resp, err := local.Do(req)
	if err != nil {
		return err
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("the archdoc at %s would not stop: %s", m.URL(), resp.Status)
	}
	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); {
		c, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", m.Port), 200*time.Millisecond)
		if err != nil {
			return nil
		}
		c.Close()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(100 * time.Millisecond):
		}
	}
	return fmt.Errorf("the archdoc at %s said it would stop, and is still there", m.URL())
}

// ServeMachine serves the library as this machine's one archdoc until ctx ends, or until another
// archdoc asks it to stop — then ErrHandedOver. It takes port, or any free one when another program
// has port, and records itself in MachineFile for the next archdoc to find; the record goes when it
// stops, unless a newer one has written its own.
func ServeMachine(ctx context.Context, port int, up Updater, ready func(url string)) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	var once sync.Once
	var handedOver atomic.Bool
	stop := func() {
		once.Do(func() {
			handedOver.Store(true)
			// After the answer is sent: the page, or the archdoc asking, hears that it is stopping.
			time.AfterFunc(50*time.Millisecond, cancel)
		})
	}
	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		ln, err = net.Listen("tcp", "127.0.0.1:0")
	}
	if err != nil {
		return err
	}
	m := Machine{Port: ln.Addr().(*net.TCPAddr).Port, PID: os.Getpid(), Build: core.Build().String(), Since: time.Now().UTC()}
	if err := record(m); err != nil {
		ln.Close()
		return err
	}
	defer forget(m)
	err = serveOn(ctx, ln, newHome(up, stop), func(string) {
		if ready != nil {
			ready(m.URL())
		}
	})
	if err == nil && handedOver.Load() {
		return ErrHandedOver
	}
	return err
}

func record(m Machine) error {
	file, err := MachineFile()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		return err
	}
	b, _ := json.MarshalIndent(m, "", "  ")
	tmp := file + "." + strconv.Itoa(m.PID)
	if err := os.WriteFile(tmp, append(b, '\n'), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, file)
}

// forget removes the record if it is still this one's.
func forget(m Machine) {
	file, err := MachineFile()
	if err != nil {
		return
	}
	b, err := os.ReadFile(file)
	if err != nil {
		return
	}
	var now Machine
	if json.Unmarshal(b, &now) == nil && now.PID == m.PID && now.Port == m.Port {
		os.Remove(file)
	}
}

// Address is a place's page in the library: the repository or folder at path, by the id Home gives
// it. path is absolute, as Home remembers it.
func Address(path string) string {
	return "/at/" + recent.ID(path) + "/"
}
