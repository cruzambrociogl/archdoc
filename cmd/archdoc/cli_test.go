package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// cli runs archdoc as a script would — no one to ask — and returns what it printed and its exit code.
func cli(t *testing.T, args ...string) (string, int, error) {
	t.Helper()
	var out bytes.Buffer
	err := runIn(env{out: &out}, args)
	return out.String(), exitCode(err), err
}

// Help is there for every command, and asking for it is not an error.
func TestEveryCommandHasItsHelp(t *testing.T) {
	for _, c := range commands {
		if c.name == "help" || c.name == "version" {
			continue
		}
		for _, ask := range [][]string{{c.name, "--help"}, {"help", c.name}, {c.name, "-h"}} {
			out, code, err := cli(t, ask...)
			if code != 0 || !strings.HasPrefix(out, "usage: archdoc "+c.name) {
				t.Errorf("%v: exit %d, %v\n%s", ask, code, err, out)
			}
		}
	}
}

// Used wrongly is exit 2, with something to act on: the nearest command or flag, where a flag moved.
func TestMistakesAreUsageErrorsThatSayWhatToDo(t *testing.T) {
	cases := []struct {
		args []string
		want string
	}{
		{[]string{"genrate", "."}, `did you mean "generate"`},
		{[]string{"generate", "--verbsoe", "."}, "did you mean --verbose"},
		{[]string{"generate", "--label", "."}, "archdoc label <path>"},
		{[]string{"generate", "--explain", "."}, "archdoc explain <path>"},
		{[]string{"generate", "--explain-gaps", "."}, "renamed: --gaps"},
		{[]string{"scan", "--explain", "."}, "renamed: --considered"},
		{[]string{"history", "-n"}, "-n needs a value"},
		{[]string{"export", "."}, "--site"},
	}
	for _, c := range cases {
		_, code, err := cli(t, c.args...)
		if code != 2 || err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%v: exit %d, %v — want exit 2 saying %q", c.args, code, err, c.want)
		}
	}
}

// A command that writes is never given a path by default: run from the wrong folder, it would write
// there. Found on 9 Oct, when a bare 'archdoc generate' documented archdoc itself.
func TestWritingCommandsNeedAPath(t *testing.T) {
	dir := t.TempDir()
	wd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(wd)
	os.WriteFile("app.py", []byte("print('hi')\n"), 0o644)
	for _, c := range []string{"generate", "init", "label", "explain"} {
		_, code, err := cli(t, c)
		if code != 2 || !strings.Contains(err.Error(), "needs one") {
			t.Errorf("%s with no path: exit %d, %v", c, code, err)
		}
	}
	if entries, _ := os.ReadDir(dir); len(entries) != 1 {
		t.Errorf("something was written: %v", entries)
	}
}

// A command that reads leaves a folder exactly as it found it — history included: opening the store
// creates it, which a read must not do.
func TestReadingCommandsWriteNothing(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "app.py"), []byte("print('hi')\n"), 0o644)
	for _, args := range [][]string{{"scan", dir}, {"status", dir}, {"history", dir}, {"runs", dir}, {"show", dir, "app"}, {"scan", dir, "--json"}} {
		cli(t, args...)
	}
	if entries, _ := os.ReadDir(dir); len(entries) != 1 {
		t.Errorf("reading wrote into the repository: %v", entries)
	}
}

// init creates archdoc's directory and never overwrites what is there.
func TestInitCreatesAndKeeps(t *testing.T) {
	dir := t.TempDir()
	out, code, _ := cli(t, "init", dir)
	if code != 0 || !strings.Contains(out, "created .archdoc/rules.yaml") {
		t.Fatalf("exit %d\n%s", code, out)
	}
	rules := filepath.Join(dir, ".archdoc", "rules.yaml")
	os.WriteFile(rules, []byte("rules: []\n"), 0o644)
	out, _, _ = cli(t, "init", dir)
	if b, _ := os.ReadFile(rules); string(b) != "rules: []\n" || !strings.Contains(out, "kept") {
		t.Errorf("init overwrote rules.yaml:\n%s", out)
	}
	// The template is all comments: generate reads it as no rules.
	os.Remove(rules)
	cli(t, "init", dir)
	os.WriteFile(filepath.Join(dir, "app.py"), []byte("print('hi')\n"), 0o644)
	if _, code, err := cli(t, "generate", dir); code != 0 {
		t.Errorf("generate after init: %v", err)
	}
}

// A paid run is never sent with no one to ask and no --yes, and a dry run shows the request and
// writes nothing — whatever key is set. Nothing here reaches the network.
func TestPaidRunsAskFirstAndDryRunsWriteNothing(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "not-a-real-key")
	root := copyFixture(t, "gateway")

	_, code, err := cli(t, "label", root)
	if code != 2 || !strings.Contains(err.Error(), "--yes") {
		t.Errorf("label with no one to ask: exit %d, %v", code, err)
	}
	out, code, _ := cli(t, "label", root, "--dry-run")
	if code != 0 || !strings.Contains(out, "[system]") || !strings.Contains(out, "Label this architecture") || !strings.Contains(out, "Nothing sent, nothing written") {
		t.Errorf("label --dry-run: exit %d\n%s", code, out)
	}
	if _, err := os.Stat(filepath.Join(root, docsDir)); !os.IsNotExist(err) {
		t.Error("a refused or dry run wrote the documentation")
	}

	t.Setenv("ANTHROPIC_API_KEY", "")
	_, code, err = cli(t, "label", root, "--yes")
	if code != 1 || !strings.Contains(err.Error(), "ANTHROPIC_API_KEY") {
		t.Errorf("label with no key: exit %d, %v", code, err)
	}
}

// status says whether the documentation still matches the code — by structure, not by words.
func TestStatusSaysWhenTheCodeHasMoved(t *testing.T) {
	root := copyFixture(t, "gateway")
	gen(t, root)
	out, code, _ := cli(t, "status", root)
	if code != 0 || !strings.Contains(out, "the architecture is as documented") || !strings.Contains(out, "network        never used") {
		t.Fatalf("exit %d\n%s", code, out)
	}
	compose := filepath.Join(root, "docker-compose.yml")
	b, _ := os.ReadFile(compose)
	os.WriteFile(compose, []byte(strings.Replace(string(b), "\n  db:", "\n  cache:\n    image: redis:7\n  db:", 1)), 0o644)
	out, _, _ = cli(t, "status", root)
	if !strings.Contains(out, "change(s) to the structure since") {
		t.Errorf("a new service did not show:\n%s", out)
	}
}

// show finds a thing by id or name, lists what an ambiguous word matches, and fails on nothing.
func TestShowFindsWhatItIsAskedFor(t *testing.T) {
	root := copyFixture(t, "gateway")
	if _, code, err := cli(t, "show", root, "db"); code != 1 || !strings.Contains(err.Error(), "generate") {
		t.Errorf("before generate: exit %d, %v", code, err)
	}
	gen(t, root)
	out, code, err := cli(t, "show", root, "svc:db")
	if code != 0 || !strings.Contains(out, "db  [datastore") || !strings.Contains(out, "docker-compose.yml:") {
		t.Errorf("by id: exit %d, %v\n%s", code, err, out)
	}
	if out, code, _ := cli(t, "show", root, "DB"); code != 0 || !strings.Contains(out, "id svc:db") {
		t.Errorf("by name, any case: exit %d\n%s", code, out)
	}
	if _, code, _ := cli(t, "show", root, "no-such-thing"); code != 1 {
		t.Errorf("nothing found: exit %d", code)
	}
	if out, code, _ := cli(t, "show", root, "svc:db", "--json"); code != 0 || !strings.Contains(out, `"element"`) {
		t.Errorf("--json: exit %d\n%s", code, out)
	}
}
