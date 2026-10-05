package render

import "testing"

func TestRemoteWebAddresses(t *testing.T) {
	cases := map[string]string{
		"git@github.com:immich-app/immich.git":            "https://github.com/immich-app/immich",
		"https://github.com/mastodon/mastodon.git":        "https://github.com/mastodon/mastodon",
		"https://token:x-oauth@github.com/owner/repo.git": "https://github.com/owner/repo",
		"ssh://git@gitlab.com/group/project.git":          "https://gitlab.com/group/project",
		"git@internal.example:team/repo.git":              "",
		"/some/local/path":                                "",
	}
	for in, want := range cases {
		if got := webURL(in); got != want {
			t.Errorf("webURL(%q) = %q, want %q", in, got, want)
		}
	}
}
