package semantic

import (
	"context"
	"strings"
	"testing"

	"github.com/anthropics/anthropic-sdk-go/option"

	"github.com/cruzambrociogl/archdoc/internal/core"
)

// A question picks the facts that share its words, best first, and the calls around them; one
// that shares none picks nothing, and is answered without asking.
func TestAQuestionPicksTheFactsItIsAbout(t *testing.T) {
	m := explainModel()
	facts := QuestionFacts(m, "Where are the albums listed?")
	if len(facts) == 0 || !strings.Contains(facts[0].Text, "GET /api/albums") {
		t.Fatalf("facts: %+v", facts)
	}
	calls := false
	for _, f := range facts {
		calls = calls || strings.Contains(f.Text, `"controllers" uses "services"`)
	}
	if !calls {
		t.Errorf("the call from the component that handles it was not picked: %+v", facts)
	}
	// More of the question met ranks first; a word repeated counts once.
	m2 := m
	m2.Nodes = append(append([]core.Node(nil), m.Nodes...),
		core.Node{ID: "cmp:photos", Name: "photos photos photos", Kind: core.Component, Parent: "svc:server", Prov: core.Provenance{File: "p.ts", Line: 1}},
		core.Node{ID: "cmp:faces", Name: "face detection", Kind: core.Component, Parent: "svc:server", Prov: core.Provenance{File: "f.ts", Line: 1}})
	if got := QuestionFacts(m2, "Where are faces detected in a photo?"); len(got) == 0 || !strings.Contains(got[0].Text, "face detection") {
		t.Errorf("the fact meeting most of the question is not first: %+v", got)
	}
	if got := QuestionFacts(m, "How is the weather in Lisbon?"); got != nil {
		t.Errorf("a question about nothing here picked %+v", got)
	}
	if got := QuestionFacts(m, "Which part can list everything?"); got != nil {
		t.Errorf("a question that names nothing, only a summary's word, picked %+v", got)
	}
	if got := QuestionFacts(m, "What does it do?"); got != nil {
		t.Errorf("a question of stop words alone picked %+v", got)
	}
	// camelCase and snake_case apart, plurals singular, the asking words gone.
	if got := strings.Join(words("getAllAlbums album_comments detected detection class"), " "); got != "album album comment detect detect class" {
		t.Errorf("words: %q", got)
	}
}

// An answer keeps only sentences whose citations resolve; one citing a fact it was not given is
// sent back, and the corrected answer is kept with what the facts did not settle.
func TestAnAnswerIsCitedOrRefused(t *testing.T) {
	facts := QuestionFacts(explainModel(), "Where are the albums listed?")
	complete, sent := scripted(t,
		Reply{Text: `{"sentences":[{"text":"Albums are listed by GET /api/albums.","cites":["F99"]}],"not_found":""}`},
		Reply{Text: `{"sentences":[{"text":"Albums are listed by GET /api/albums.","cites":["F1"]}],"not_found":"How they are sorted."}`},
	)
	a, rep, err := AskQuestion(context.Background(), complete, ExplainModel, "Where are the albums listed?", facts)
	if err != nil {
		t.Fatal(err)
	}
	if len(*sent) != 2 || rep.Attempts != 2 {
		t.Fatalf("%d requests, %d attempts", len(*sent), rep.Attempts)
	}
	if len(a.Claims) != 1 || a.Claims[0].Cites[0].Line != 7 || a.NotFound != "How they are sorted." || a.Facts != len(facts) {
		t.Errorf("answer: %+v", a)
	}
	if AnswerKey("Where are the albums  listed?", facts) != AnswerKey("where are the albums listed?", facts) {
		t.Error("the same question, spaced or cased otherwise, is remembered apart")
	}
}

// On the wire: the question, and the facts as --explain words them — no line, no provenance, no
// description, no column type, no key.
func TestAskSendsTheQuestionAndStructureAndNothingElse(t *testing.T) {
	srv, received := fakeAnthropic(t, `{"sentences":[{"text":"GET /api/albums lists them.","cites":["F1"]}],"not_found":""}`)
	rec := &Recorder{}
	complete := ClaudeWith(ExplainModel, AskSchema(), AskMaxTokens, AskEffort, rec, option.WithBaseURL(srv.URL), option.WithAPIKey("test-key"))

	m := explainModel()
	for i := range m.Nodes {
		m.Nodes[i].Description = "an album description nobody asked to send"
		m.Nodes[i].Prov.Line = 7777
	}
	m.Nodes = append(m.Nodes, core.Node{ID: "tbl:server/album", Name: "album", Kind: core.Table, Parent: "svc:server",
		Prov: core.Provenance{File: "x.ts", Line: 7777}, Columns: []core.Column{{Name: "ownerId", Type: "uuid-type-not-sent"}}})
	q := "Where are the albums listed?"
	if _, _, err := AskQuestion(context.Background(), complete, ExplainModel, q, QuestionFacts(m, q)); err != nil {
		t.Fatal(err)
	}
	sent := ""
	for _, b := range *received {
		sent += string(b)
	}
	for _, want := range []string{"Where are the albums listed?", "GET /api/albums", "List all albums", "ownerId", `"effort":"low"`} {
		if !strings.Contains(sent, want) {
			t.Errorf("%q was not sent", want)
		}
	}
	for _, leak := range []string{"7777", "uuid-type-not-sent", "nobody asked to send", "provenance", "test-key"} {
		if strings.Contains(sent, leak) {
			t.Errorf("%q left the machine", leak)
		}
	}
	if rec.Bytes() != len(sent) {
		t.Errorf("the recorder holds %d bytes, the server received %d", rec.Bytes(), len(sent))
	}
}
