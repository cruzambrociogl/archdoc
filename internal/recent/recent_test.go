package recent

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"
)

func keep(t *testing.T) {
	t.Helper()
	file, was := filepath.Join(t.TempDir(), "recent.json"), File
	File = func() (string, error) { return file, nil }
	t.Cleanup(func() { File = was })
}

// The most recently touched place comes first; opening one remembers what it was at; the list
// keeps the most recent twenty.
func TestTheListKeepsTheMostRecentPlacesFirst(t *testing.T) {
	keep(t)
	at := time.Date(2026, 10, 22, 9, 0, 0, 0, time.UTC)
	if err := Touch("/w/immich", at); err != nil {
		t.Fatal(err)
	}
	if err := Touch("/w/ws", at.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if err := Open("/w/immich", at.Add(2*time.Minute), map[string]int64{"/w/immich": 3}); err != nil {
		t.Fatal(err)
	}
	places, err := List()
	if err != nil {
		t.Fatal(err)
	}
	if len(places) != 2 || places[0].Path != "/w/immich" || places[0].Seen["/w/immich"] != 3 || places[0].Opened.IsZero() || !places[1].Opened.IsZero() {
		t.Fatalf("places: %+v", places)
	}

	// Touching keeps what an earlier opening saw.
	Touch("/w/immich", at.Add(3*time.Minute))
	if places, _ := List(); places[0].Seen["/w/immich"] != 3 {
		t.Errorf("touching forgot what was seen: %+v", places[0])
	}

	for i := range Most + 5 {
		Touch(fmt.Sprintf("/w/r%d", i), at.Add(time.Hour+time.Duration(i)*time.Minute))
	}
	if places, _ := List(); len(places) != Most || places[0].Path != fmt.Sprintf("/w/r%d", Most+4) {
		t.Errorf("after many: %d places, first %s", len(places), places[0].Path)
	}
}

func TestNoFileIsAnEmptyList(t *testing.T) {
	keep(t)
	if places, err := List(); err != nil || len(places) != 0 {
		t.Errorf("%v %v", places, err)
	}
}

func TestAnIDIsStableAndSaysNothingOfThePath(t *testing.T) {
	if ID("/w/immich") != ID("/w/immich") || ID("/w/immich") == ID("/w/ws") || len(ID("/w/immich")) != 10 {
		t.Error("ids")
	}
}
