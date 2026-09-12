package main

import (
	"flag"
	"io"
	"reflect"
	"testing"
)

func historyFlags() *flag.FlagSet {
	fs := flag.NewFlagSet("history", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.Int("n", 20, "")
	fs.Bool("json", false, "")
	return fs
}

// `archdoc history -n 3 ../repo` failed with "flag needs an argument: -n": every flag was treated
// as on/off, so the 3 became the path. A value flag must keep the word after it.
func TestValueFlagKeepsItsValue(t *testing.T) {
	cases := []struct {
		args       []string
		flags, pos []string
	}{
		{[]string{"-n", "3", "../repo"}, []string{"-n", "3"}, []string{"../repo"}},
		{[]string{"../repo", "-n", "3"}, []string{"-n", "3"}, []string{"../repo"}},
		{[]string{"-n=3", "../repo"}, []string{"-n=3"}, []string{"../repo"}},
		// A boolean flag never swallows the path that follows it.
		{[]string{"--json", "../repo"}, []string{"--json"}, []string{"../repo"}},
	}

	for _, c := range cases {
		flags, pos := partitionArgs(historyFlags(), c.args)
		if !reflect.DeepEqual(flags, c.flags) || !reflect.DeepEqual(pos, c.pos) {
			t.Errorf("%v: flags=%v pos=%v, want flags=%v pos=%v", c.args, flags, pos, c.flags, c.pos)
		}
	}
}

// The end-to-end form of the same bug: the flags must actually parse, and n must be 3.
func TestHistoryDashNParses(t *testing.T) {
	fs := historyFlags()
	flags, pos := partitionArgs(fs, []string{"-n", "3", "../repo"})
	if err := fs.Parse(flags); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if n := fs.Lookup("n").Value.String(); n != "3" {
		t.Errorf("n = %s, want 3", n)
	}
	if len(pos) != 1 || pos[0] != "../repo" {
		t.Errorf("path = %v, want ../repo", pos)
	}
}
