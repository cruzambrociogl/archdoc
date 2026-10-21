// Package server serves the archdoc web app for one repository, on this machine only (SUR-07).
//
// It is separate from package archdoc so that the engine's API never imports an HTTP package: the
// network-boundary check stays about imports, not exceptions. Like internal/serve, which does the
// work, this package only listens; it sends nothing anywhere.
package server

import (
	"context"

	"github.com/cruzambrociogl/archdoc/internal/serve"
)

// ListenAndServe serves the app for the repository at root on a local port until ctx ends. ready is
// called with the app's address once it is listening.
func ListenAndServe(ctx context.Context, root string, port int, ready func(url string)) error {
	return serve.ListenAndServe(ctx, root, port, ready)
}
