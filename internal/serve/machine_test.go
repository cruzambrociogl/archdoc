package serve

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// This machine's one archdoc records where it is, is found there as itself, and stops when another
// asks — `archdoc restart` — ending with ErrHandedOver and taking its record with it.
func TestTheMachineIsFoundAndHandsOver(t *testing.T) {
	file, was := filepath.Join(t.TempDir(), "running.json"), MachineFile
	MachineFile = func() (string, error) { return file, nil }
	t.Cleanup(func() { MachineFile = was })

	if _, ok := Running(); ok {
		t.Fatal("found an archdoc before one started")
	}
	ready := make(chan string, 1)
	done := make(chan error, 1)
	go func() { done <- ServeMachine(context.Background(), 0, nil, func(url string) { ready <- url }) }()
	select {
	case <-ready:
	case err := <-done:
		t.Fatalf("the machine did not start: %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("the machine did not start")
	}

	m, ok := Running()
	if !ok {
		t.Fatal("the running machine was not found")
	}
	if m.PID != os.Getpid() || m.Port == 0 {
		t.Errorf("found %+v, want this process on a port", m)
	}
	if err := m.Stop(context.Background()); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	select {
	case err := <-done:
		if !errors.Is(err, ErrHandedOver) {
			t.Errorf("ServeMachine ended with %v, want ErrHandedOver", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the machine did not stop")
	}
	if _, err := os.Stat(file); !os.IsNotExist(err) {
		t.Errorf("the record is still there after stopping: %v", err)
	}
	if _, ok := Running(); ok {
		t.Error("a stopped machine is still found")
	}
}

// A record left by an archdoc that is gone — a crash, a reboot — finds nothing.
func TestAStaleRecordFindsNothing(t *testing.T) {
	file, was := filepath.Join(t.TempDir(), "running.json"), MachineFile
	MachineFile = func() (string, error) { return file, nil }
	t.Cleanup(func() { MachineFile = was })
	if err := record(Machine{Port: 1, PID: 99999, Build: "dev"}); err != nil {
		t.Fatal(err)
	}
	if _, ok := Running(); ok {
		t.Error("a record of nothing listening was taken for a running archdoc")
	}
}
