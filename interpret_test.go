package archdoc_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/cruzambrociogl/archdoc"
)

// An interpretation sends nothing until its Consent agrees, says both requests and their cost when it
// asks, and is never sent above its cap.
func TestInterpretAsksFirstAndKeepsToItsCap(t *testing.T) {
	m, err := archdoc.ReadModel(shop(t))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := archdoc.Interpret(context.Background(), m, archdoc.InterpretOptions{}); err == nil {
		t.Error("an interpretation with no Consent went ahead")
	}
	var asked archdoc.Request
	_, err = archdoc.Interpret(context.Background(), m, archdoc.InterpretOptions{DryRun: true,
		Consent: func(r archdoc.Request) error { asked = r; return nil }})
	if !errors.Is(err, archdoc.ErrStopped) {
		t.Fatalf("a dry run: %v", err)
	}
	if asked.What != "interpret" || len(asked.Plans) != 2 || !asked.EstimateKnown || asked.Estimate <= 0 ||
		asked.Model != archdoc.Haiku {
		t.Errorf("the Consent was asked %+v", asked)
	}
	var withSonnet archdoc.Request
	archdoc.Interpret(context.Background(), m, archdoc.InterpretOptions{DryRun: true, WriteModel: archdoc.Sonnet,
		Consent: func(r archdoc.Request) error { withSonnet = r; return nil }})
	if withSonnet.Model != archdoc.Haiku+", then "+archdoc.Sonnet || withSonnet.Estimate <= asked.Estimate {
		t.Errorf("writing on Sonnet: asked %q, about $%.4f against Haiku's $%.4f", withSonnet.Model, withSonnet.Estimate, asked.Estimate)
	}
	if strings.Contains(asked.Plans[0].Prompt+asked.Plans[1].Prompt, "return this.") {
		t.Error("a request carries code")
	}
	called := false
	_, err = archdoc.Interpret(context.Background(), m, archdoc.InterpretOptions{Cap: asked.Estimate / 2,
		Consent: func(archdoc.Request) error { called = true; return nil }})
	if !errors.Is(err, archdoc.ErrOverCap) || called {
		t.Errorf("over its cap: %v, asked %v", err, called)
	}
}
