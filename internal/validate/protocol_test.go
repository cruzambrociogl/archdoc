package validate

import (
	"testing"

	"github.com/cruzambrociogl/archdoc/internal/core"
)

func relabel(label string, origin core.Origin, prov core.Provenance) (core.Model, Result) {
	return Apply(good(), []core.Op{{
		Kind: core.SetEdgeLabel, Target: "actor:user", To: "svc:api", Value: label,
		Origin: origin, Prov: prov,
	}})
}

// The live-run defect, as a test: "over HTTPS" on an edge whose recorded technology says nothing
// about HTTPS is the model asserting a fact. It must be rejected, so VAL-07 sends it back.
func TestModelLabelMayNotAssertAProtocol(t *testing.T) {
	for _, label := range []string{
		"calls the API over HTTPS",
		"streams updates via WebSocket",
		"talks gRPC to",
		"connects over TLS",
	} {
		if _, r := relabel(label, core.Semantic, runProv()); r.OK() {
			t.Errorf("accepted a model label asserting a protocol: %q", label)
		}
	}
}

// Describing what a relationship does is the model's job, and technology words that are not
// transport claims — PostgreSQL, SQL — are fine.
func TestModelLabelWithoutProtocolIsAccepted(t *testing.T) {
	for _, label := range []string{
		"sends API requests to",
		"runs SQL queries against",
		"opens pooled PostgreSQL connections to",
	} {
		if _, r := relabel(label, core.Semantic, runProv()); !r.OK() {
			t.Errorf("rejected a legitimate label %q:\n%s", label, r.Error())
		}
	}
}

// When extraction recorded the protocol, naming it is repetition, not invention.
func TestModelLabelMayRepeatARecordedProtocol(t *testing.T) {
	m := good()
	out, r := Apply(m, []core.Op{{
		Kind: core.SetEdgeLabel, Target: "svc:api", To: "svc:db", Value: "queries over postgres",
		Origin: core.Semantic, Prov: runProv(),
	}})
	if !r.OK() {
		t.Fatalf("a label naming the recorded protocol was rejected:\n%s", r.Error())
	}
	_ = out
}

// A person is accountable for rules.yaml, so a rule may state a protocol the file does not.
func TestRuleLabelMayStateAProtocol(t *testing.T) {
	if _, r := relabel("calls the API over HTTPS", core.Rules, rulesAt(9)); !r.OK() {
		t.Errorf("a rule stating a protocol was rejected:\n%s", r.Error())
	}
}
