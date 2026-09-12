//go:build !dev

// Package web holds the built frontend, embedded into the archdoc binary so that users need
// nothing installed: `archdoc serve` works from the one file.
//
// Building from source needs Node; using archdoc does not (docs/stack-decision.md §3). The
// `all:` prefix embeds the tracked dist/.gitkeep, so the binary compiles on a fresh clone before
// the frontend has ever been built — serve then says what to run instead of showing a blank page.
package web

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var dist embed.FS

// Assets returns the built frontend, and whether it has actually been built.
func Assets() (fs.FS, bool) {
	sub, err := fs.Sub(dist, "dist")
	if err != nil {
		return nil, false
	}
	_, err = fs.Stat(sub, "index.html")
	return sub, err == nil
}

// DevServer is the frontend development server to proxy to; empty outside the dev build.
func DevServer() string { return "" }
