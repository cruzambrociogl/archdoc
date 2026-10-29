// Package server serves the archdoc web app for one repository, on this machine only (SUR-07).
//
// It is separate from package archdoc so that the engine's API never imports an HTTP package: the
// network-boundary check stays about imports, not exceptions. Like internal/serve, which does the
// work, this package only listens; it sends nothing anywhere — the app's Update with AI goes
// through package archdoc's paid runs, which ask internal/semantic.
package server

import (
	"context"
	"errors"

	"github.com/cruzambrociogl/archdoc"
	"github.com/cruzambrociogl/archdoc/internal/serve"
)

// ListenAndServe serves the app for the repository at root — or a folder of documented repositories,
// as one system — on a local port until ctx ends. ready is called with the app's address once it is
// listening. The app can update each repository's documentation, as archdoc generate does.
func ListenAndServe(ctx context.Context, root string, port int, ready func(url string)) error {
	return serve.ListenAndServe(ctx, root, port, update, ready)
}

// ListenAndServeHome serves Home — every repository, and folder of them, this machine remembers —
// on a local port until ctx ends.
func ListenAndServeHome(ctx context.Context, port int, ready func(url string)) error {
	return serve.ListenAndServeHome(ctx, port, update, ready)
}

// update is generate for the app: recorded in the repository's history like the command's runs,
// and each paid run put to the app's Agree with what it would send and cost.
func update(ctx context.Context, root string, r serve.Run) (serve.Outcome, error) {
	agree := archdoc.Consent(func(q archdoc.Request) error {
		if r.Agree == nil {
			return errors.New("not sent: nothing agreed to sending")
		}
		err := r.Agree(serve.Ask{What: q.What, Model: q.Model, Sends: q.Sends, Requests: len(q.Plans), Bytes: q.Bytes, USD: q.Estimate, Known: q.EstimateKnown})
		if errors.Is(err, serve.ErrStopped) {
			return archdoc.ErrStopped
		}
		return err
	})
	o := archdoc.GenerateOptions{Log: r.Log, History: true, DryRun: r.DryRun, By: "the app"}
	if r.Label {
		o.Label = &archdoc.LabelOptions{Model: archdoc.LabelModel, Consent: agree}
	}
	if r.Explain {
		o.Explain = &archdoc.ExplainOptions{Consent: agree}
	}
	g, err := archdoc.Generate(ctx, root, o)
	if errors.Is(err, archdoc.ErrStopped) {
		return serve.Outcome{}, serve.ErrStopped
	}
	if err != nil {
		return serve.Outcome{}, err
	}
	return serve.Outcome{Version: g.Version, Changed: g.Changed}, nil
}
