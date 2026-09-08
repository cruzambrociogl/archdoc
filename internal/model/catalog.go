package model

import (
	"strings"
	"unicode"

	"github.com/cruzambrociogl/archdoc/internal/archdoc"
)

// The catalog turns an image reference into the two things a container diagram needs: what the
// thing is, and what it runs. Both are lookups, not inferences — MDL-05 and MDL-07 are CAT
// capabilities, and an image not in the table gets the honest default rather than a guess.
//
// Matching is on the exact image name, never a substring. supabase/postgres-meta is an
// application that manages a database, and darthsim/imgproxy is an image transformer rather
// than a proxy; both would be misfiled by any rule loose enough to also catch them.

type entry struct {
	kind archdoc.Kind
	tech string // display name; the version is appended from the tag when there is one
}

var catalog = map[string]entry{
	// Relational
	"postgres":   {archdoc.Datastore, "PostgreSQL"},
	"postgresql": {archdoc.Datastore, "PostgreSQL"},
	"mysql":      {archdoc.Datastore, "MySQL"},
	"mariadb":    {archdoc.Datastore, "MariaDB"},
	"cockroach":  {archdoc.Datastore, "CockroachDB"},

	// Key-value and document
	"redis":     {archdoc.Datastore, "Redis"},
	"valkey":    {archdoc.Datastore, "Valkey"},
	"memcached": {archdoc.Datastore, "Memcached"},
	"mongo":     {archdoc.Datastore, "MongoDB"},
	"etcd":      {archdoc.Datastore, "etcd"},

	// Search and analytics
	"elasticsearch": {archdoc.Datastore, "Elasticsearch"},
	"opensearch":    {archdoc.Datastore, "OpenSearch"},
	"meilisearch":   {archdoc.Datastore, "Meilisearch"},
	"typesense":     {archdoc.Datastore, "Typesense"},
	"clickhouse":    {archdoc.Datastore, "ClickHouse"},

	// Object storage
	"minio": {archdoc.Datastore, "MinIO"},

	// Messaging
	"rabbitmq": {archdoc.Queue, "RabbitMQ"},
	"kafka":    {archdoc.Queue, "Kafka"},
	"nats":     {archdoc.Queue, "NATS"},

	// Edge — infrastructure, which the derivation rule keeps out of the container view
	"nginx":   {archdoc.Proxy, "nginx"},
	"traefik": {archdoc.Proxy, "Traefik"},
	"haproxy": {archdoc.Proxy, "HAProxy"},
	"caddy":   {archdoc.Proxy, "Caddy"},
	"envoy":   {archdoc.Proxy, "Envoy"},
	"kong":    {archdoc.Proxy, "Kong"},
}

// classify returns what an image is, what it runs, and the evidence for the second answer.
//
// An unknown image is an application with no technology. That is the honest default: almost
// every custom-built service is an application, and leaving the technology empty says "not
// known from configuration" rather than inventing a stack. MDL-08 hands exactly this case to
// the semantic layer later.
func classify(image string) (kind archdoc.Kind, tech string, prov archdoc.Provenance) {
	name, tag := splitImage(image)

	e, ok := catalog[name]
	if !ok {
		return archdoc.Application, "", archdoc.Provenance{}
	}

	// The technology is not something the repository stated — a lookup table supplied it.
	// Recording that is PRV-02, and it is what makes AC-1 measurable: the criterion admits
	// catalog provenance, but only if the catalog actually leaves a trace.
	prov = archdoc.Provenance{Origin: archdoc.Catalog, Note: name}

	if v := version(tag); v != "" {
		return e.kind, e.tech + " " + v, prov
	}
	return e.kind, e.tech, prov
}

// splitImage reduces a reference to the image name and its tag. Registry, namespace and digest
// are all discarded: ghcr.io/immich-app/postgres:14-vectorchord0.4.3@sha256:bcf... is postgres,
// tagged 14-vectorchord0.4.3.
func splitImage(image string) (name, tag string) {
	if image == "" {
		return "", ""
	}

	// The digest comes last and never contains a slash.
	if i := strings.Index(image, "@"); i >= 0 {
		image = image[:i]
	}

	ref := image
	if i := strings.LastIndex(ref, "/"); i >= 0 {
		ref = ref[i+1:]
	}

	if i := strings.Index(ref, ":"); i >= 0 {
		return strings.ToLower(ref[:i]), ref[i+1:]
	}
	return strings.ToLower(ref), ""
}

// version pulls the version out of a tag: the leading run of digits and dots, and nothing else.
// "14-alpine" is 14, "1.27" is 1.27, "16-vectorchord0.4.3" is 16.
//
// The whole prefix is kept rather than the major alone — "nginx 1" is wrong where "nginx 1.27"
// is right, and a long version like 17.6.1.136 is what the file actually says. A tag such as
// "release" or "latest" has no version and yields nothing: better a bare "PostgreSQL" than
// "PostgreSQL latest".
func version(tag string) string {
	end := 0
	for end < len(tag) && (unicode.IsDigit(rune(tag[end])) || tag[end] == '.') {
		end++
	}
	return strings.TrimRight(tag[:end], ".")
}
