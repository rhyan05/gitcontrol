// Package workspace persiste a lista de repositórios abertos na TUI.
package workspace

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

// Config é o workspace salvo pelo gitcontrol.
type Config struct {
	Repos  []string `json:"repos"`
	Active int      `json:"active"`
}

// Path devolve o caminho do workspace no diretório de configuração do usuário.
func Path() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "gitcontrol", "workspace.json"), nil
}

// Load lê o workspace. Arquivo inexistente devolve Config vazio.
func Load() (Config, error) {
	path, err := Path()
	if err != nil {
		return Config{}, err
	}

	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return Config{}, nil
	}
	if err != nil {
		return Config{}, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}
	cfg = Add(cfg)
	if cfg.Active >= len(cfg.Repos) {
		cfg.Active = 0
	}
	return cfg, nil
}

// Save grava o workspace criando o diretório se necessário.
func Save(cfg Config) error {
	path, err := Path()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// Add normaliza caminhos, remove duplicados e vazios.
func Add(cfg Config, paths ...string) Config {
	seen := map[string]bool{}
	repos := make([]string, 0, len(cfg.Repos)+len(paths))

	for _, path := range append(append([]string{}, cfg.Repos...), paths...) {
		path = strings.TrimSpace(path)
		if path == "" {
			continue
		}
		if abs, err := filepath.Abs(path); err == nil {
			path = abs
		}
		if seen[path] {
			continue
		}
		seen[path] = true
		repos = append(repos, path)
	}

	cfg.Repos = repos
	if cfg.Active < 0 || cfg.Active >= len(repos) {
		cfg.Active = 0
	}
	return cfg
}
