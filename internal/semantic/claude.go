package semantic

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/anthropics/anthropic-sdk-go/shared/constant"
)

// Claude returns a Completer backed by the Anthropic API.
//
// Credentials come from the environment the way the SDK always resolves them —
// ANTHROPIC_API_KEY, or a profile from `ant auth login`. archdoc never asks for, stores, or logs
// a key.
//
// Two request choices worth knowing:
//
//   - Structured outputs with a strict schema. The answer is guaranteed to be the operation list
//     the schema describes, so parsing cannot fail on a well-formed response.
//   - Server-side fallbacks in "default" mode. If Claude Opus 5's safety classifiers decline a
//     request, the API re-runs it on Anthropic's recommended fallback instead of returning an
//     empty refusal. A refusal that still gets through is reported, not guessed around.
func Claude(model string) Completer {
	// A key created at organisation level rather than inside a workspace must name the
	// workspace on every request. Keys created inside a workspace need nothing extra, so
	// the header is sent only when the variable is set.
	var opts []option.RequestOption
	ws := strings.TrimSpace(os.Getenv("ANTHROPIC_WORKSPACE_ID"))
	if ws != "" && !strings.HasPrefix(ws, "sk-ant-") {
		opts = append(opts, option.WithHeader("anthropic-workspace-id", ws))
	}
	client := anthropic.NewClient(opts...)

	return func(ctx context.Context, system string, turns []Turn) (Reply, error) {
		// A key pasted into the wrong variable would otherwise travel as a header value. Refuse
		// before sending anything, rather than pass a secret where an identifier belongs.
		if strings.HasPrefix(ws, "sk-ant-") {
			return Reply{}, errors.New("ANTHROPIC_WORKSPACE_ID contains an API key, not a workspace id " +
				"(workspace ids start wrkspc_). Put the key in ANTHROPIC_API_KEY; a key created " +
				"inside a workspace needs no ANTHROPIC_WORKSPACE_ID at all")
		}

		msgs := make([]anthropic.BetaMessageParam, 0, len(turns))
		for _, t := range turns {
			block := anthropic.NewBetaTextBlock(t.Text)
			if t.Role == "assistant" {
				msgs = append(msgs, anthropic.BetaMessageParam{
					Role:    anthropic.BetaMessageParamRoleAssistant,
					Content: []anthropic.BetaContentBlockParamUnion{block},
				})
				continue
			}
			msgs = append(msgs, anthropic.NewBetaUserMessage(block))
		}

		resp, err := client.Beta.Messages.New(ctx, anthropic.BetaMessageNewParams{
			Model:     model,
			MaxTokens: 16000,
			System:    []anthropic.BetaTextBlockParam{{Text: system}},
			Messages:  msgs,
			OutputConfig: anthropic.BetaOutputConfigParam{
				Format: anthropic.BetaJSONOutputFormatParam{Schema: Schema()},
			},
			Fallbacks: anthropic.BetaFallbacksParamUnion{OfDefault: constant.ValueOf[constant.Default]()},
			Betas:     []anthropic.AnthropicBeta{anthropic.AnthropicBetaServerSideFallback2026_07_01},
		})
		if err != nil {
			return Reply{}, explain(err)
		}

		if resp.StopReason == anthropic.BetaStopReasonRefusal {
			return Reply{Refused: true}, nil
		}

		var text string
		for _, block := range resp.Content {
			if t, ok := block.AsAny().(anthropic.BetaTextBlock); ok {
				text += t.Text
			}
		}
		return Reply{Text: text}, nil
	}
}

// explain turns an API failure into something a person running a documentation tool can act on.
func explain(err error) error {
	var apierr *anthropic.Error
	if !errors.As(err, &apierr) {
		return fmt.Errorf("could not reach the Anthropic API: %w", err)
	}

	switch apierr.StatusCode {
	case 400:
		if strings.Contains(err.Error(), "anthropic-workspace-id") {
			return errors.New("this API key is not tied to a workspace — set ANTHROPIC_WORKSPACE_ID " +
				"to the workspace to bill, or create a key inside a workspace in the console")
		}
		return fmt.Errorf("Anthropic API error 400: %w", err)
	case 401:
		return errors.New("no valid Anthropic credentials — set ANTHROPIC_API_KEY, or run without --label")
	case 429:
		return errors.New("rate limited by the Anthropic API — try again shortly, or run without --label")
	default:
		return fmt.Errorf("Anthropic API error %d: %w", apierr.StatusCode, err)
	}
}
