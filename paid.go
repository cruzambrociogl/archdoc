package archdoc

import (
	"errors"

	"github.com/cruzambrociogl/archdoc/internal/semantic"
)

// The two runs that send anything anywhere — label and explain. Each says what it will send and
// what that should cost before sending, through the caller's Consent, built by the same functions
// the real requests are, so what is shown is what would go.

// Consent decides whether a paid run goes ahead. It is given what would be sent and what that
// should cost; it returns nil to send, ErrStopped to stop with nothing sent or written, or any
// other error to fail. The command asks at a terminal, or takes --yes; a caller with no one to ask
// should refuse. A nil Consent refuses.
type Consent func(Request) error

// Request is what a paid run is about to send.
type Request struct {
	// What is the run: "label" or "explain".
	What string
	// Model is the model the requests go to.
	Model string
	// Sends says, in words, what the requests carry.
	Sends string
	// Plans are the requests, as they would be sent.
	Plans []Plan
	// Bytes is their size in all.
	Bytes int
	// Estimate is what they should cost in US dollars, from their size; EstimateKnown is false when
	// this model's price is unknown.
	Estimate      float64
	EstimateKnown bool
}

// errNoConsent is a paid run with no Consent to ask.
var errNoConsent = errors.New("not sent: nothing agreed to sending")

const (
	labelSends   = "names, kinds, technologies and relationships of the containers — no path, no code"
	explainSends = "each component's name, file paths, counts and the code's own route summaries — no code"
)

// What a request costs is its tokens; before it is sent, only its size is known. Measured on
// Immich's eleven runs: a request carries about 2.5 bytes per input token (run 11: 270,139 bytes,
// 107,916 tokens); an explanation's answer is about 250 tokens (22,358 over 91 requests); a label
// run answers about one token for every five bytes it was sent (run 10: 1,290 for 7,055).
const (
	bytesPerToken = 2.5
	explainAnswer = 250
)

func labelAnswer(p semantic.Planned) int64 { return int64(p.Bytes()) / 5 }

// LabelModel is the model a label run asks unless told otherwise.
const LabelModel = semantic.Model

// ask puts a run's requests to the Consent.
func (c Consent) ask(what, model, sends string, plans []Plan, answerTokens int64) error {
	bytes := 0
	for _, p := range plans {
		bytes += p.Bytes()
	}
	usd, known := semantic.Cost(model, int64(float64(bytes)/bytesPerToken), answerTokens)
	if c == nil {
		return errNoConsent
	}
	return c(Request{What: what, Model: model, Sends: sends, Plans: plans, Bytes: bytes, Estimate: usd, EstimateKnown: known})
}
