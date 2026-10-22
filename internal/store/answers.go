package store

import (
	"database/sql"
	"errors"
	"time"
)

// Remembered answers: what a paid question or plan check came back with, by what it was asked —
// the question and its facts, the plan and its facts — so asking the same again costs nothing.
// Kept here, in the local cache, not in a committed file: what someone asked about their code is
// theirs, and losing it costs no more than asking again.

const answersSchema = `
CREATE TABLE IF NOT EXISTS answers (
	kind       TEXT NOT NULL,
	key        TEXT NOT NULL,
	created_at TEXT NOT NULL,
	body       TEXT NOT NULL,
	PRIMARY KEY (kind, key)
);
`

// Keep remembers an answer of a kind ("ask", "plan") by its key, replacing an older one.
func (s *Store) Keep(kind, key string, body []byte) error {
	_, err := s.db.Exec(`INSERT OR REPLACE INTO answers (kind, key, created_at, body) VALUES (?, ?, ?, ?)`,
		kind, key, time.Now().UTC().Format(time.RFC3339Nano), string(body))
	return err
}

// Recall is the answer remembered by a key, and when it was given; nil when there is none.
func (s *Store) Recall(kind, key string) ([]byte, time.Time, error) {
	var body, at string
	err := s.db.QueryRow(`SELECT body, created_at FROM answers WHERE kind = ? AND key = ?`, kind, key).Scan(&body, &at)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, time.Time{}, nil
	}
	if err != nil {
		return nil, time.Time{}, err
	}
	when, _ := time.Parse(time.RFC3339Nano, at)
	return []byte(body), when, nil
}

// LastAnswer is the most recent answer of a kind, whatever it was asked by — the last plan check,
// shown until the plan or the code changes and it is asked for again.
func (s *Store) LastAnswer(kind string) ([]byte, time.Time, error) {
	var body, at string
	err := s.db.QueryRow(`SELECT body, created_at FROM answers WHERE kind = ? ORDER BY created_at DESC LIMIT 1`, kind).Scan(&body, &at)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, time.Time{}, nil
	}
	if err != nil {
		return nil, time.Time{}, err
	}
	when, _ := time.Parse(time.RFC3339Nano, at)
	return []byte(body), when, nil
}
