package semantic

import (
	"bytes"
	"io"
	"net/http"
	"sync"

	"github.com/anthropics/anthropic-sdk-go/option"
)

// A Recorder keeps an exact copy of every request that leaves the machine (SUR-14, AC-8).
//
// It is attached to the SDK client as middleware, so what it records is the request body as it
// goes onto the wire — not archdoc's own idea of what it meant to send. That distinction is the
// whole point: a run log built from the prompt string would account for the prompt, and miss
// anything the client added around it. This one accounts for every byte, because it is the bytes.
//
// It lives in internal/semantic because this is the only package permitted to make outbound
// calls; the recorder is the other half of that rule — the calls are allowed, and they are
// written down.
type Recorder struct {
	mu        sync.Mutex
	exchanges []Exchange
}

// Exchange is one request that left the machine, and what came back.
type Exchange struct {
	Method string
	URL    string
	Body   []byte // exactly as sent
	Status int    // 0 when the request never got a response
}

// Middleware returns the SDK hook that records each request. Safe to share across retries: the
// SDK retries by calling the middleware again, and each attempt is a separate transmission that
// must be counted separately.
func (r *Recorder) Middleware() option.Middleware {
	return func(req *http.Request, next option.MiddlewareNext) (*http.Response, error) {
		var body []byte
		if req.Body != nil {
			b, err := io.ReadAll(req.Body)
			if err != nil {
				return nil, err
			}
			req.Body.Close()
			body = b
			req.Body = io.NopCloser(bytes.NewReader(b))
		}

		resp, err := next(req)

		ex := Exchange{Method: req.Method, URL: req.URL.String(), Body: body}
		if resp != nil {
			ex.Status = resp.StatusCode
		}
		r.mu.Lock()
		r.exchanges = append(r.exchanges, ex)
		r.mu.Unlock()

		return resp, err
	}
}

// Exchanges returns everything recorded, in the order it was sent.
func (r *Recorder) Exchanges() []Exchange {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]Exchange(nil), r.exchanges...)
}

// Bytes is the total request payload that left the machine.
func (r *Recorder) Bytes() int {
	n := 0
	for _, e := range r.Exchanges() {
		n += len(e.Body)
	}
	return n
}

// Price is dollars per million tokens, from Anthropic's published rates. Only the default model
// is listed: a model archdoc does not know the price of reports its tokens and no cost, rather
// than a guessed figure (NFR-6 asks for actual cost, and a wrong number is worse than none).
var prices = map[string]struct{ In, Out float64 }{
	"claude-opus-5": {In: 5, Out: 25},
}

// Cost returns the dollar cost of a run, and whether the model's price is known.
func Cost(model string, in, out int64) (float64, bool) {
	p, ok := prices[model]
	if !ok {
		return 0, false
	}
	return (float64(in)*p.In + float64(out)*p.Out) / 1e6, true
}
