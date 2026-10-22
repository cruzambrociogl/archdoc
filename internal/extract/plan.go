package extract

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/cruzambrociogl/archdoc/internal/core"
)

// Plans (F-40): the files a person names as what they asked for — a spec, a task list, an agent's
// plan. archdoc reads these and no others (D-9): a plan file is read because someone pointed at it.
// What it asks for is its list items, each at its line; a plan with no list is read line by line.

// MostPlanItems is how many requirements a plan check reads, across its files: a plan longer than
// this is checked in its first part, and says so.
const MostPlanItems = 60

// planMax bounds a plan file: a plan is a page or a few, never a dump.
const planMax = 256 << 10

var listItem = regexp.MustCompile(`^\s*(?:[-*+]|\d+[.)])\s+(?:\[[ xX]\]\s+)?(.+?)\s*$`)

// PlanFile checks that a path names a plan archdoc may read: inside the repository, a Markdown or
// text file, not too large. It returns the path as the repository writes it.
func PlanFile(root, rel string) (string, error) {
	clean := filepath.ToSlash(filepath.Clean(strings.TrimSpace(rel)))
	if clean == "." || clean == "" || filepath.IsAbs(clean) || strings.HasPrefix(clean, "../") || clean == ".." {
		return "", fmt.Errorf("%q is not a path inside the repository", rel)
	}
	if ext := strings.ToLower(filepath.Ext(clean)); ext != ".md" && ext != ".markdown" && ext != ".txt" {
		return "", fmt.Errorf("%s: a plan is a Markdown or text file", clean)
	}
	info, err := os.Stat(filepath.Join(root, filepath.FromSlash(clean)))
	switch {
	case errors.Is(err, os.ErrNotExist):
		return "", fmt.Errorf("%s: no such file", clean)
	case err != nil:
		return "", err
	case !info.Mode().IsRegular():
		return "", fmt.Errorf("%s: not a file", clean)
	case info.Size() > planMax:
		return "", fmt.Errorf("%s: larger than a plan (%d KB at most)", clean, planMax>>10)
	}
	return clean, nil
}

// Plan reads what the named plan files ask for, in file order, numbered R1, R2, … — at most
// MostPlanItems; more says how many were left unread.
func Plan(root string, files []string) (items []core.PlanItem, more int, err error) {
	for _, rel := range files {
		clean, err := PlanFile(root, rel)
		if err != nil {
			return nil, 0, err
		}
		b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(clean)))
		if err != nil {
			return nil, 0, err
		}
		for _, it := range planItems(string(b)) {
			if len(items) == MostPlanItems {
				more++
				continue
			}
			items = append(items, core.PlanItem{ID: fmt.Sprintf("R%d", len(items)+1), Text: it.text, Prov: core.Provenance{File: clean, Line: it.line}})
		}
	}
	return items, more, nil
}

type planLine struct {
	text string
	line int
}

// planItems is a plan's list items — bullets, numbers, task boxes — or, with none, its lines of
// prose; never a heading, a code block, or a line of a table.
func planItems(text string) []planLine {
	var list, prose []planLine
	fenced := false
	for i, raw := range strings.Split(text, "\n") {
		line := strings.TrimSpace(raw)
		if strings.HasPrefix(line, "```") || strings.HasPrefix(line, "~~~") {
			fenced = !fenced
			continue
		}
		if fenced || line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "|") || strings.HasPrefix(line, ">") {
			continue
		}
		if m := listItem.FindStringSubmatch(raw); m != nil {
			list = append(list, planLine{m[1], i + 1})
		} else if !strings.HasPrefix(line, "---") && !strings.HasPrefix(line, "<!--") {
			prose = append(prose, planLine{line, i + 1})
		}
	}
	if len(list) > 0 {
		return list
	}
	return prose
}
