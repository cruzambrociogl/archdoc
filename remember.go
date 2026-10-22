package archdoc

import (
	"time"

	"github.com/cruzambrociogl/archdoc/internal/recent"
	"github.com/cruzambrociogl/archdoc/internal/system"
)

// Remember adds a documented repository, or a folder of them, to this machine's list of recent
// places, which Home shows (platform-surface §5.3). The list is a file in the user's configuration
// folder; nothing is written in the repository. The command calls it after generate and serve; a
// program using the library decides for itself — the platform's worker never does.
func Remember(root string) error {
	return recent.Touch(root, time.Now().UTC())
}

// Opened remembers a place as opened now, with each of its repositories' latest version: what Home
// counts changes from, the next time.
func Opened(root string) error {
	seen, err := system.Versions(root)
	if err != nil {
		return err
	}
	return recent.Open(root, time.Now().UTC(), seen)
}
