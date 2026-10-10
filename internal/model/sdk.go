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

	// Hosted backends: the database, the sign-in and the storage of an application that runs none
	// of its own — which is most of what an AI builds a first version on.
	"@supabase/supabase-js":    {"Supabase", "stores data and signs users in with"},
	"@supabase/ssr":            {"Supabase", "stores data and signs users in with"},
	"supabase":                 {"Supabase", "stores data and signs users in with"},
	"firebase":                 {"Firebase", "stores data and signs users in with"},
	"firebase_admin":           {"Firebase", "calls"},
	"appwrite":                 {"Appwrite", "stores data and signs users in with"},
	"pocketbase":               {"PocketBase", "stores data and signs users in with"},
	"convex":                   {"Convex", "stores data in"},
	"aws-amplify":              {"AWS Amplify", "stores data and signs users in with"},
	"@clerk/clerk-react":       {"Clerk", "signs users in with"},
	"@clerk/nextjs":            {"Clerk", "signs users in with"},
	"next-auth":                {"Identity provider (OAuth 2)", "signs users in with"},
	"@planetscale/database":    {"PlanetScale", "stores data in"},
	"@neondatabase/serverless": {"Neon", "stores data in"},
	"@upstash/redis":           {"Upstash Redis", "caches data in"},
	"@vercel/postgres":         {"Vercel Postgres", "stores data in"},
	"@vercel/kv":               {"Vercel KV", "caches data in"},
	"@vercel/blob":             {"Vercel Blob", "stores files in"},
	"cloudinary":               {"Cloudinary", "stores media in"},
	"algoliasearch":            {"Algolia", "searches with"},

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
	{"firebase/", sdk{"Firebase", "stores data and signs users in with"}},
	{"@firebase/", sdk{"Firebase", "stores data and signs users in with"}},
	{"@supabase/", sdk{"Supabase", "stores data and signs users in with"}},
	{"@clerk/", sdk{"Clerk", "signs users in with"}},
}

// Java packages and .NET namespaces name their client libraries by the vendor's own name: a package
// is matched when it is one of these or inside one, the longest first.
var namespaceCatalog = map[string]sdk{
	// Java
	"com.stripe":                                 {"Stripe", "takes payments through"},
	"software.amazon.awssdk.services.s3":         {"Object storage (S3)", "stores objects in"},
	"com.amazonaws.services.s3":                  {"Object storage (S3)", "stores objects in"},
	"software.amazon.awssdk":                     {"Amazon Web Services", "calls"},
	"com.amazonaws":                              {"Amazon Web Services", "calls"},
	"com.google.cloud.storage":                   {"Google Cloud Storage", "stores objects in"},
	"com.azure.storage.blob":                     {"Azure Blob Storage", "stores objects in"},
	"com.twilio":                                 {"Twilio", "sends messages through"},
	"com.sendgrid":                               {"SendGrid", "sends email through"},
	"jakarta.mail":                               {"Email server (SMTP)", "sends email through"},
	"javax.mail":                                 {"Email server (SMTP)", "sends email through"},
	"org.springframework.mail":                   {"Email server (SMTP)", "sends email through"},
	"org.springframework.security.oauth2.client": {"Identity provider (OAuth 2)", "signs users in with"},
	"com.google.firebase":                        {"Firebase", "calls"},
	"com.slack.api":                              {"Slack", "posts to"},
	"io.sentry":                                  {"Sentry", "reports errors to"},
	"com.openai":                                 {"OpenAI API", "calls a model at"},
	"org.springframework.ai.openai":              {"OpenAI API", "calls a model at"},
	"dev.langchain4j.model.openai":               {"OpenAI API", "calls a model at"},
	"com.anthropic":                              {"Anthropic API", "calls a model at"},
	"org.springframework.ai.anthropic":           {"Anthropic API", "calls a model at"},
	"org.springframework.ai.azure.openai":        {"Azure OpenAI", "calls a model at"},
	"io.opentelemetry.exporter":                  {"Telemetry collector (OpenTelemetry)", "exports telemetry to"},
	// .NET
	"Stripe":                  {"Stripe", "takes payments through"},
	"Amazon.S3":               {"Object storage (S3)", "stores objects in"},
	"Amazon":                  {"Amazon Web Services", "calls"},
	"Azure.Storage.Blobs":     {"Azure Blob Storage", "stores objects in"},
	"Azure.AI.OpenAI":         {"Azure OpenAI", "calls a model at"},
	"Azure.Security.KeyVault": {"Azure Key Vault", "reads secrets from"},
	"Google.Cloud.Storage":    {"Google Cloud Storage", "stores objects in"},
	"Twilio":                  {"Twilio", "sends messages through"},
	"SendGrid":                {"SendGrid", "sends email through"},
	"MailKit":                 {"Email server (SMTP)", "sends email through"},
	"System.Net.Mail":         {"Email server (SMTP)", "sends email through"},
	"Microsoft.Identity.Web":  {"Microsoft Entra ID", "signs users in with"},
	"Auth0":                   {"Auth0", "signs users in with"},
	"Sentry":                  {"Sentry", "reports errors to"},
	"OpenAI":                  {"OpenAI API", "calls a model at"},
	"Anthropic":               {"Anthropic API", "calls a model at"},
	"OpenTelemetry.Exporter":  {"Telemetry collector (OpenTelemetry)", "exports telemetry to"},
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
	for ns := pkg; ns != ""; {
		if s, ok := namespaceCatalog[ns]; ok {
			return s, true
		}
		i := strings.LastIndex(ns, ".")
		if i < 0 {
			break
		}
		ns = ns[:i]
	}
	return sdk{}, false
}

// starterCatalog: a Spring Boot starter in a module's build names the system it connects to by
// itself — the starter configures the client, and the code may only ever use Spring's own interface
// to it (ChatClient, JavaMailSender). The manifest line is the evidence.
var starterCatalog = map[string]sdk{
	"spring-ai-starter-model-openai":          {"OpenAI API", "calls a model at"},
	"spring-ai-openai-spring-boot-starter":    {"OpenAI API", "calls a model at"},
	"spring-ai-starter-model-anthropic":       {"Anthropic API", "calls a model at"},
	"spring-ai-anthropic-spring-boot-starter": {"Anthropic API", "calls a model at"},
	"spring-ai-starter-model-azure-openai":    {"Azure OpenAI", "calls a model at"},
	"spring-ai-starter-model-ollama":          {"Ollama", "calls a model at"},
	"spring-boot-starter-mail":                {"Email server (SMTP)", "sends email through"},
	"spring-boot-starter-oauth2-client":       {"Identity provider (OAuth 2)", "signs users in with"},
	"spring-cloud-starter-aws":                {"Amazon Web Services", "calls"},
	"spring-cloud-azure-starter-storage-blob": {"Azure Blob Storage", "stores objects in"},
}

// starters adds the external systems a Java module's starters name.
func starters(m *archdoc.Model, f *archdoc.FactSet) {
	containerOf := map[string]string{}
	has := map[string]bool{}
	for _, n := range m.Nodes {
		has[n.ID] = true
		if n.Dir != "" && n.Kind == archdoc.Application {
			containerOf[n.Dir] = n.ID
		}
	}
	for _, a := range f.Apps {
		container, ok := containerOf[a.Dir]
		if !ok || a.Language != "Java" {
			continue
		}
		for _, r := range a.Requires {
			_, artifact, _ := strings.Cut(r.Name, ":")
			s, ok := starterCatalog[artifact]
			if !ok || r.Dev {
				continue
			}
			id := externalID(strings.ToLower(strings.NewReplacer(" ", "-", "(", "", ")", "").Replace(s.system)))
			lookup := archdoc.Provenance{Origin: archdoc.Catalog, Note: "starter: " + artifact}
			if !has[id] {
				has[id] = true
				m.Nodes = append(m.Nodes, archdoc.Node{ID: id, Name: s.system, NameProv: lookup, Kind: archdoc.External,
					Evidence: archdoc.Referenced, Prov: r.Prov})
			}
			m.Edges = append(m.Edges, archdoc.Edge{From: container, To: id, Label: s.label, LabelProv: lookup,
				Traffic: true, Prov: []archdoc.Provenance{r.Prov}})
		}
	}
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
