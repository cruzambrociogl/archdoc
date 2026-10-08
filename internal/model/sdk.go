package model

import (
	"strings"

	"github.com/cruzambrociogl/archdoc/internal/archdoc"
)

// What a well-known client library says about the world outside (F-08): code that imports
// nodemailer sends email through an SMTP server; code that imports openid-client signs users in
// with an identity provider. The import is a fact, at its line. What the library is for is a
// lookup — the same kind as the catalog that knows the postgres image is PostgreSQL — and is
// cited as one, so a reader can tell which half was read and which was looked up.
//
// Matching is on the exact package name, or a stated prefix; a library the table does not know
// draws nothing. Only systems outside the repository are listed: a Redis client says nothing the
// Compose file's redis service does not already say better.

type sdk struct {
	system string // the external system's name
	label  string // what the container does with it
}

var sdkCatalog = map[string]sdk{
	// Email
	"nodemailer":     {"Email server (SMTP)", "sends email through"},
	"smtplib":        {"Email server (SMTP)", "sends email through"},
	"emails":         {"Email server (SMTP)", "sends email through"},
	"@sendgrid/mail": {"SendGrid", "sends email through"},
	"resend":         {"Resend", "sends email through"},
	"postmark":       {"Postmark", "sends email through"},
	"mailgun.js":     {"Mailgun", "sends email through"},

	// Identity
	"openid-client":       {"Identity provider (OpenID Connect)", "signs users in with"},
	"passport-oauth2":     {"Identity provider (OAuth 2)", "signs users in with"},
	"authlib":             {"Identity provider (OAuth 2)", "signs users in with"},
	"requests_oauthlib":   {"Identity provider (OAuth 2)", "signs users in with"},
	"@auth0/auth0-spa-js": {"Auth0", "signs users in with"},
	"auth0":               {"Auth0", "signs users in with"},

	// Object storage
	"@aws-sdk/client-s3":    {"Object storage (S3)", "stores objects in"},
	"aws-sdk":               {"Amazon Web Services", "calls"},
	"boto3":                 {"Amazon Web Services", "calls"},
	"@google-cloud/storage": {"Google Cloud Storage", "stores objects in"},
	"@azure/storage-blob":   {"Azure Blob Storage", "stores objects in"},

	// Payments, messaging
	"stripe":         {"Stripe", "takes payments through"},
	"twilio":         {"Twilio", "sends messages through"},
	"@slack/web-api": {"Slack", "posts to"},
	"firebase-admin": {"Firebase", "calls"},

	// Models
	"openai":            {"OpenAI API", "calls a model at"},
	"@anthropic-ai/sdk": {"Anthropic API", "calls a model at"},
	"anthropic":         {"Anthropic API", "calls a model at"},
	"huggingface_hub":   {"Hugging Face Hub", "downloads models from"},

	// Observability
	"@sentry/node":      {"Sentry", "reports errors to"},
	"@sentry/browser":   {"Sentry", "reports errors to"},
	"@sentry/sveltekit": {"Sentry", "reports errors to"},
	"@sentry/react":     {"Sentry", "reports errors to"},
	"sentry_sdk":        {"Sentry", "reports errors to"},
}

// Families matched by prefix: every exporter of a kind talks to the same sort of system.
var sdkPrefixes = []struct {
	prefix string
	sdk
}{
	{"@opentelemetry/exporter-", sdk{"Telemetry collector (OpenTelemetry)", "exports telemetry to"}},
}

func lookupSDK(pkg string) (sdk, bool) {
	if s, ok := sdkCatalog[pkg]; ok {
		return s, true
	}
	for _, p := range sdkPrefixes {
		if strings.HasPrefix(pkg, p.prefix) {
			return p.sdk, true
		}
	}
	return sdk{}, false
}

// sdks adds the external systems a container's imports name, and its relationship to each.
func sdks(m *archdoc.Model, src archdoc.Source, container string, has map[string]bool) {
	cited := map[string][]archdoc.Provenance{} // system → the imports that name it
	labels := map[string]sdk{}
	from := map[string]string{} // system → the package that named it first
	for _, f := range src.Files {
		for _, imp := range f.Imports {
			if imp.How != archdoc.ByPackage {
				continue
			}
			s, ok := lookupSDK(imp.Package)
			if !ok {
				continue
			}
			if _, seen := labels[s.system]; !seen {
				labels[s.system], from[s.system] = s, imp.Package
			}
			if len(cited[s.system]) < maxCited {
				cited[s.system] = append(cited[s.system], imp.Prov)
			}
		}
	}
	for _, system := range sortedKeys(cited) {
		id := externalID(strings.ToLower(strings.NewReplacer(" ", "-", "(", "", ")", "").Replace(system)))
		lookup := archdoc.Provenance{Origin: archdoc.Catalog, Note: "library: " + from[system]}
		if !has[id] {
			has[id] = true
			m.Nodes = append(m.Nodes, archdoc.Node{ID: id, Name: system, NameProv: lookup, Kind: archdoc.External,
				Evidence: archdoc.Referenced, Prov: cited[system][0]})
		}
		m.Edges = append(m.Edges, archdoc.Edge{From: container, To: id, Label: labels[system].label, LabelProv: lookup,
			Traffic: true, Prov: cited[system]})
	}
}
