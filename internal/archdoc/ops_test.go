package archdoc

import "testing"

// Hard rule 4 in executable form. The semantic layer may describe what extraction found; it may
// never change what exists. Enforcing that on the operation type rather than inside the semantic
// package means the next producer of operations cannot forget it.
func TestSemanticLayerCannotChangeStructure(t *testing.T) {
	forbidden := []OpKind{Exclude, AddEdge, RemoveEdge}
	allowed := []OpKind{SetName, SetDescription, SetTechnology, SetKind, SetEdgeLabel, Group}

	for _, k := range forbidden {
		if k.AllowedFrom(Semantic) {
			t.Errorf("the model may produce %q — it can add or remove elements", k)
		}
	}
	for _, k := range allowed {
		if !k.AllowedFrom(Semantic) {
			t.Errorf("the model may not produce %q — labelling is its whole job", k)
		}
	}
}

// A person writing rules.yaml is accountable and the file records what they said, so rules carry
// no such restriction.
func TestRulesMayChangeStructure(t *testing.T) {
	for _, k := range []OpKind{Exclude, AddEdge, RemoveEdge, SetName, Group} {
		if !k.AllowedFrom(Rules) {
			t.Errorf("rules may not produce %q", k)
		}
	}
}

// AC-1 admits extraction *or* catalog provenance. A technology that came from a lookup table has
// no line, and demanding one would make the criterion unsatisfiable rather than strict.
func TestCatalogProvenanceIsTraceableWithoutALine(t *testing.T) {
	p := Provenance{Origin: Catalog, Note: "postgres"}

	if !p.Known() {
		t.Error("a catalog fact naming its entry is not traceable")
	}
	if got, want := p.String(), "catalog: postgres"; got != want {
		t.Errorf("rendered %q, want %q", got, want)
	}
	if (Provenance{Origin: Catalog}).Known() {
		t.Error("a catalog fact naming nothing was accepted")
	}
}

// Every parser writes Provenance{File, Line} without stating an origin, because a fact read from
// a file is the ordinary case. The zero value has to mean that.
func TestZeroOriginMeansExtraction(t *testing.T) {
	p := Provenance{File: "docker-compose.yml", Line: 13}

	if !p.Known() {
		t.Error("a parsed fact with a line is not traceable")
	}
	if p.origin() != Extraction {
		t.Errorf("zero origin is %q, want extraction", p.origin())
	}
	if p.Origin.Interpretation() {
		t.Error("a parsed fact was classed as interpretation")
	}
}

// PRV-05 renders the difference between what was read and what was judged. Only the model
// interprets.
func TestOnlyTheModelIsInterpretation(t *testing.T) {
	for _, o := range []Origin{Extraction, Catalog, Rules} {
		if o.Interpretation() {
			t.Errorf("%q was classed as interpretation", o)
		}
	}
	if !Semantic.Interpretation() {
		t.Error("the model's output was not classed as interpretation")
	}
}
