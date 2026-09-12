package store

import (
	"database/sql"
	"encoding/json"
	"time"
)

// A Run is one use of the network: the requests that left the machine, and what they cost.
//
// It is recorded whether or not the run succeeded. A labelling run that failed validation three
// times still transmitted three requests, and a run log that only kept successes would miss
// exactly the runs someone is most likely to go looking for.
type Run struct {
	ID         int64
	StartedAt  time.Time
	FinishedAt time.Time
	Status     string // "ok", or the error that stopped it
	Commit     string
	EgressMode string // "structure-only" — the only mode archdoc has
	Model      string

	Requests  int
	BytesSent int
	TokensIn  int64
	TokensOut int64
	CostUSD   float64
	CostKnown bool

	// Exchanges is exactly what was sent, request by request. Empty in listings; filled by Run.
	Exchanges []Exchange
}

// Exchange is one request as it went onto the wire.
type Exchange struct {
	Method string `json:"method"`
	URL    string `json:"url"`
	Status int    `json:"status"`
	Body   string `json:"body"`
}

const runsSchema = `
CREATE TABLE IF NOT EXISTS runs (
	id          INTEGER PRIMARY KEY AUTOINCREMENT,
	started_at  TEXT    NOT NULL,
	finished_at TEXT    NOT NULL,
	status      TEXT    NOT NULL,
	commit_sha  TEXT    NOT NULL DEFAULT '',
	egress_mode TEXT    NOT NULL,
	model       TEXT    NOT NULL DEFAULT '',
	requests    INTEGER NOT NULL,
	bytes_sent  INTEGER NOT NULL,
	tokens_in   INTEGER NOT NULL,
	tokens_out  INTEGER NOT NULL,
	cost_usd    REAL    NOT NULL DEFAULT 0,
	cost_known  INTEGER NOT NULL DEFAULT 0,
	payload     TEXT    NOT NULL
);
`

// SaveRun records a run and returns its id.
func (s *Store) SaveRun(r Run) (int64, error) {
	payload, err := json.Marshal(r.Exchanges)
	if err != nil {
		return 0, err
	}
	res, err := s.db.Exec(`INSERT INTO runs
		(started_at, finished_at, status, commit_sha, egress_mode, model,
		 requests, bytes_sent, tokens_in, tokens_out, cost_usd, cost_known, payload)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		r.StartedAt.UTC().Format(time.RFC3339), r.FinishedAt.UTC().Format(time.RFC3339),
		r.Status, r.Commit, r.EgressMode, r.Model,
		r.Requests, r.BytesSent, r.TokensIn, r.TokensOut, r.CostUSD, boolInt(r.CostKnown), string(payload))
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// Runs lists runs, newest first, without their payloads.
func (s *Store) Runs(limit int) ([]Run, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.Query(`SELECT id, started_at, finished_at, status, commit_sha, egress_mode, model,
		requests, bytes_sent, tokens_in, tokens_out, cost_usd, cost_known
		FROM runs ORDER BY id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Run
	for rows.Next() {
		r, err := scanRun(rows, false)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// Run returns one run with everything it sent.
func (s *Store) Run(id int64) (*Run, error) {
	row := s.db.QueryRow(`SELECT id, started_at, finished_at, status, commit_sha, egress_mode, model,
		requests, bytes_sent, tokens_in, tokens_out, cost_usd, cost_known, payload
		FROM runs WHERE id = ?`, id)
	r, err := scanRun(row, true)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &r, nil
}

type scanner interface{ Scan(...any) error }

func scanRun(sc scanner, withPayload bool) (Run, error) {
	var (
		r              Run
		started, ended string
		known          int
		payload        string
	)
	dest := []any{&r.ID, &started, &ended, &r.Status, &r.Commit, &r.EgressMode, &r.Model,
		&r.Requests, &r.BytesSent, &r.TokensIn, &r.TokensOut, &r.CostUSD, &known}
	if withPayload {
		dest = append(dest, &payload)
	}
	if err := sc.Scan(dest...); err != nil {
		return Run{}, err
	}
	r.StartedAt, _ = time.Parse(time.RFC3339, started)
	r.FinishedAt, _ = time.Parse(time.RFC3339, ended)
	r.CostKnown = known == 1
	if withPayload && payload != "" {
		if err := json.Unmarshal([]byte(payload), &r.Exchanges); err != nil {
			return Run{}, err
		}
	}
	return r, nil
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
