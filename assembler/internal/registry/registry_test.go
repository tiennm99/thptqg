package registry

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeRegistry(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "datasets.json"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestLoadsTheRealRegistry(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	got, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) == 0 {
		t.Fatal("no datasets")
	}
	for _, d := range got {
		if d.ID == "" || d.ExpectedRows <= 0 || d.DbSizeMb <= 0 {
			t.Errorf("incomplete entry: %+v", d)
		}
		// Every declared dataset must have somewhere to read from and a parse
		// config, or the build fails much later with a worse message.
		for _, p := range []string{
			filepath.Join(root, "data", d.ID),
			filepath.Join(root, "parser", "configs", d.ID+".yml"),
		} {
			if _, err := os.Stat(p); err != nil {
				t.Errorf("%s: missing %s", d.ID, p)
			}
		}
	}
}

// TestIncompleteEntriesAreRejected: a dataset missing either guard figure would
// otherwise publish unverified. Both are load-bearing, so neither may default.
func TestIncompleteEntriesAreRejected(t *testing.T) {
	for name, body := range map[string]string{
		"no expectedRows": `{"datasets":[{"id":"x","dbSizeMb":1}]}`,
		"no dbSizeMb":     `{"datasets":[{"id":"x","expectedRows":1}]}`,
		"no id":           `{"datasets":[{"expectedRows":1,"dbSizeMb":1}]}`,
		"zero rows":       `{"datasets":[{"id":"x","expectedRows":0,"dbSizeMb":1}]}`,
		"empty":           `{"datasets":[]}`,
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := Load(writeRegistry(t, body)); err == nil {
				t.Error("expected an error")
			}
		})
	}
}

func TestMissingAndMalformedRegistry(t *testing.T) {
	if _, err := Load(t.TempDir()); err == nil {
		t.Error("a missing registry must fail")
	}
	if _, err := Load(writeRegistry(t, "{not json")); err == nil {
		t.Error("a malformed registry must fail")
	}
}

func TestSelect(t *testing.T) {
	all := []Dataset{{ID: "2016"}, {ID: "2017"}}

	got, err := Select(all, nil)
	if err != nil || len(got) != 2 {
		t.Errorf("no ids should select everything: %v %v", got, err)
	}

	got, err = Select(all, []string{"2017"})
	if err != nil || len(got) != 1 || got[0].ID != "2017" {
		t.Errorf("Select(2017) = %v, %v", got, err)
	}

	// A typo must not silently build nothing.
	_, err = Select(all, []string{"2018"})
	if err == nil {
		t.Fatal("an unknown id must fail")
	}
	if !strings.Contains(err.Error(), "2018") || !strings.Contains(err.Error(), "2016") {
		t.Errorf("the error should name the bad id and the known ones, got: %v", err)
	}
}
