package store

import (
	"path/filepath"
	"testing"
	"time"

	"database/sql"

	_ "modernc.org/sqlite"
)

func sampleRun(status string) Run {
	return Run{
		StartedAt: time.Date(2026, 9, 12, 10, 0, 0, 0, time.UTC), FinishedAt: time.Date(2026, 9, 12, 10, 2, 0, 0, time.UTC),
		Status: status, Commit: "abc123", EgressMode: "structure-only", Model: "claude-opus-5",
		Requests: 1, BytesSent: 18, TokensIn: 1200, TokensOut: 300, CostUSD: 0.0135, CostKnown: true,
		Exchanges: []Exchange{{Method: "POST", URL: "https://api.anthropic.com/v1/messages", Status: 200, Body: `{"model":"x","a":1}`}},
	}
}

// SUR-14 depends on this: the body comes back exactly as stored, byte for byte.
func TestRunRoundTripsExactly(t *testing.T) {
	s := history(t)

	id, err := s.SaveRun(sampleRun("ok"))
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	got, err := s.Run(id)
	if err != nil || got == nil {
		t.Fatalf("run: %v", err)
	}
	if len(got.Exchanges) != 1 || got.Exchanges[0].Body != `{"model":"x","a":1}` {
		t.Errorf("the payload did not survive exactly: %+v", got.Exchanges)
	}
	if got.TokensIn != 1200 || got.TokensOut != 300 || !got.CostKnown || got.CostUSD != 0.0135 {
		t.Errorf("accounting lost: %+v", got)
	}
}

// A failed run is logged like any other — a request that left the machine is in the log.
func TestFailedRunsAreLoggedToo(t *testing.T) {
	s := history(t)
	if _, err := s.SaveRun(sampleRun("the model's operations failed validation 3 times")); err != nil {
		t.Fatal(err)
	}
	list, err := s.Runs(0)
	if err != nil || len(list) != 1 {
		t.Fatalf("got %d runs (err %v)", len(list), err)
	}
	if list[0].Status == "ok" {
		t.Error("a failed run was recorded as ok")
	}
	// A listing does not carry payloads; showing a run does.
	if len(list[0].Exchanges) != 0 {
		t.Error("the listing loaded payloads it does not need")
	}
}

func TestRunsAreNewestFirst(t *testing.T) {
	s := history(t)
	first, _ := s.SaveRun(sampleRun("ok"))
	second, _ := s.SaveRun(sampleRun("ok"))

	list, err := s.Runs(0)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 || list[0].ID != second || list[1].ID != first {
		t.Errorf("not newest first: %+v", list)
	}
}

// A history file written before the run log existed gains it on open, keeping every version.
func TestOlderHistoryGainsTheRunLog(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, File)

	old, err := Open(root) // creates the file
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := old.Save(model("example"), nil, "abc123", "test"); err != nil {
		t.Fatal(err)
	}
	old.Close()

	// Simulate an older archdoc: drop the table this version adds.
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`DROP TABLE runs`); err != nil {
		t.Fatal(err)
	}
	db.Close()

	s, err := Open(root)
	if err != nil {
		t.Fatalf("an older history file failed to open: %v", err)
	}
	defer s.Close()
	if _, err := s.SaveRun(sampleRun("ok")); err != nil {
		t.Errorf("the run log was not added: %v", err)
	}
	if v, _ := s.Latest(); v == nil || v.Model.Name != "example" {
		t.Error("an existing version was lost")
	}
}
