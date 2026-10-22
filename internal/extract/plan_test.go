package extract

import (
	"strconv"
	"strings"
	"testing"
)

// A plan's requirements are its list items, at their lines — task boxes and numbers too — and
// never a heading, a code block or a table; a plan with no list is read line by line.
func TestAPlanAsksForItsListItems(t *testing.T) {
	root := tree(t, map[string]string{
		"docs/plan.md": "# Album comments\n\nSome context nobody asked to check.\n\n- Members can comment on an album.\n  - [ ] Members can delete their own comments.\n1. Comments show on the album page.\n\n```\n- not a requirement\n```\n| - nor | this |\n",
		"notes.txt":    "Albums can be shared by link.\n\nShared links expire after a week.\n",
		"big.bin":      "x",
	})
	items, more, err := Plan(root, []string{"docs/plan.md", "notes.txt"})
	if err != nil {
		t.Fatal(err)
	}
	got := []string{}
	for _, it := range items {
		got = append(got, it.ID+" "+it.Text+" @"+it.Prov.File+":"+strconv.Itoa(it.Prov.Line))
	}
	want := []string{
		"R1 Members can comment on an album. @docs/plan.md:5",
		"R2 Members can delete their own comments. @docs/plan.md:6",
		"R3 Comments show on the album page. @docs/plan.md:7",
		"R4 Albums can be shared by link. @notes.txt:1",
		"R5 Shared links expire after a week. @notes.txt:3",
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") || more != 0 {
		t.Errorf("items:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	for _, bad := range []string{"../outside.md", "/etc/passwd.md", "big.bin", "missing.md", "docs"} {
		if _, err := PlanFile(root, bad); err == nil {
			t.Errorf("%q was taken for a plan", bad)
		}
	}
}
