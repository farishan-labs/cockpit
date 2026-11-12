package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// Config is the cockpit configuration.
type Config struct {
	Port            int    `toml:"port"`
	DefaultAgent    string `toml:"default_agent"`
	TerminalBackend string `toml:"terminal_backend"`
	DBPath          string `toml:"db_path"`
	EventLogPath    string `toml:"event_log_path"`
	OpenBrowser     bool   `toml:"open_browser"`
	UserMDPath      string `toml:"user_md_path"`
}

// Default returns sensible defaults.
func Default() Config {
	home, _ := os.UserHomeDir()
	base := filepath.Join(home, ".config", "cockpit")
	return Config{
		Port:            7777,
		DefaultAgent:    "opencode",
		TerminalBackend: "tmux",
		DBPath:          filepath.Join(base, "cockpit.db"),
		EventLogPath:    filepath.Join(base, "events.jsonl"),
		OpenBrowser:     true,
		UserMDPath:      filepath.Join(base, "USER.md"),
	}
}

// Dir returns the config directory.
func Dir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "cockpit")
}

// Path returns the config file path.
func Path() string {
	return filepath.Join(Dir(), "config.toml")
}

// Load reads config from disk, creating defaults if missing.
func Load() (Config, error) {
	cfg := Default()
	p := Path()
	if _, err := os.Stat(p); os.IsNotExist(err) {
		if err := os.MkdirAll(Dir(), 0o755); err != nil {
			return cfg, err
		}
		f, err := os.Create(p)
		if err != nil {
			return cfg, err
		}
		f.Close()
		if err := Save(cfg); err != nil {
			return cfg, err
		}
		return cfg, nil
	}
	if _, err := toml.DecodeFile(p, &cfg); err != nil {
		return cfg, fmt.Errorf("decode config: %w", err)
	}
	// fill blanks
	d := Default()
	if cfg.Port == 0 {
		cfg.Port = d.Port
	}
	if cfg.TerminalBackend == "" {
		cfg.TerminalBackend = d.TerminalBackend
	}
	if cfg.DBPath == "" {
		cfg.DBPath = d.DBPath
	}
	if cfg.EventLogPath == "" {
		cfg.EventLogPath = d.EventLogPath
	}
	if cfg.UserMDPath == "" {
		cfg.UserMDPath = d.UserMDPath
	}
	// create parent dir
	if err := os.MkdirAll(filepath.Dir(cfg.UserMDPath), 0o755); err != nil {
		return cfg, err
	}
	return cfg, nil
}

// Save writes config to disk.
func Save(cfg Config) error {
	if err := os.MkdirAll(Dir(), 0o755); err != nil {
		return err
	}
	f, err := os.Create(Path())
	if err != nil {
		return err
	}
	defer f.Close()
	return toml.NewEncoder(f).Encode(cfg)
}
