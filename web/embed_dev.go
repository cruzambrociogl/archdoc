//go:build dev

// The dev build embeds nothing: `archdoc serve` proxies the page to Vite's development server,
// so a change to the frontend shows on reload without rebuilding the Go binary
// (docs/stack-decision.md §3 — "rebuilding the binary to see a CSS change is untenable").
package web

import "io/fs"

func Assets() (fs.FS, bool) { return nil, false }

func DevServer() string { return "http://127.0.0.1:5173" }
