package workspace

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveLoadAndDedupe(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("APPDATA", dir)
	t.Setenv("XDG_CONFIG_HOME", dir)

	cfg := Add(Config{}, "/tmp/a", "/tmp/a", "/tmp/b")
	if len(cfg.Repos) != 2 {
		t.Fatalf("Add não deduplicou: %+v", cfg.Repos)
	}
	cfg.Active = 1
	if err := Save(cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(loaded.Repos) != 2 || loaded.Active != 1 {
		t.Fatalf("loaded = %+v", loaded)
	}

	path, err := Path()
	if err != nil {
		t.Fatalf("Path: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("workspace não foi criado: %v", err)
	}
	if filepath.Dir(path) != filepath.Join(dir, "gitcontrol") {
		t.Fatalf("path = %q, esperava dentro de gitcontrol", path)
	}
}

func TestLoadMissingReturnsEmpty(t *testing.T) {
	t.Setenv("APPDATA", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(cfg.Repos) != 0 || cfg.Active != 0 {
		t.Fatalf("cfg = %+v, esperava vazio", cfg)
	}
}
