package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

const (
	appName    = "glab"
	configFile = "config.json"
	hostsFile  = "hosts.json"
)

// Config holds the application configuration.
type Config struct {
	Editor    string `json:"editor,omitempty"`
	Pager     string `json:"pager,omitempty"`
	Browser   string `json:"browser,omitempty"`
	Protocol  string `json:"protocol,omitempty"` // "https" or "ssh"
	GitRemote string `json:"git_remote,omitempty"`
}

// HostConfig stores per-host authentication and settings.
type HostConfig struct {
	Token    string `json:"token"`
	User     string `json:"user,omitempty"`
	Protocol string `json:"protocol,omitempty"`
	APIHost  string `json:"api_host,omitempty"`
}

// HostsConfig maps hostnames to their configurations.
type HostsConfig map[string]*HostConfig

var (
	configDir  string
	configOnce sync.Once
)

// ConfigDir returns the directory where config files are stored.
func ConfigDir() string {
	configOnce.Do(func() {
		if d := os.Getenv("GLAB_CONFIG_DIR"); d != "" {
			configDir = d
			return
		}
		home, err := os.UserHomeDir()
		if err != nil {
			configDir = filepath.Join(".", ".config", appName)
			return
		}
		configDir = filepath.Join(home, ".config", appName)
	})
	return configDir
}

// Load reads the config file from disk.
func Load() (*Config, error) {
	cfg := &Config{
		Protocol:  "https",
		GitRemote: "origin",
	}
	path := filepath.Join(ConfigDir(), configFile)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, fmt.Errorf("reading config: %w", err)
	}
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}
	return cfg, nil
}

// Save writes the config to disk.
func (c *Config) Save() error {
	dir := ConfigDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}
	path := filepath.Join(dir, configFile)
	return os.WriteFile(path, data, 0o644)
}

// Get returns a config value by key name.
func (c *Config) Get(key string) (string, error) {
	switch key {
	case "editor":
		return c.Editor, nil
	case "pager":
		return c.Pager, nil
	case "browser":
		return c.Browser, nil
	case "protocol":
		return c.Protocol, nil
	case "git_remote":
		return c.GitRemote, nil
	default:
		return "", fmt.Errorf("unknown config key: %s", key)
	}
}

// Set updates a config value by key name.
func (c *Config) Set(key, value string) error {
	switch key {
	case "editor":
		c.Editor = value
	case "pager":
		c.Pager = value
	case "browser":
		c.Browser = value
	case "protocol":
		c.Protocol = value
	case "git_remote":
		c.GitRemote = value
	default:
		return fmt.Errorf("unknown config key: %s", key)
	}
	return nil
}

// Keys returns all valid config keys.
func Keys() []string {
	return []string{"editor", "pager", "browser", "protocol", "git_remote"}
}

// LoadHosts reads the hosts configuration from disk.
func LoadHosts() (HostsConfig, error) {
	hosts := make(HostsConfig)
	path := filepath.Join(ConfigDir(), hostsFile)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return hosts, nil
		}
		return nil, fmt.Errorf("reading hosts config: %w", err)
	}
	if err := json.Unmarshal(data, &hosts); err != nil {
		return nil, fmt.Errorf("parsing hosts config: %w", err)
	}
	return hosts, nil
}

// SaveHosts writes the hosts configuration to disk.
func SaveHosts(hosts HostsConfig) error {
	dir := ConfigDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}
	data, err := json.MarshalIndent(hosts, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling hosts config: %w", err)
	}
	path := filepath.Join(dir, hostsFile)
	return os.WriteFile(path, data, 0o600)
}

// DefaultHost returns "gitlab.com" or the value of GITLAB_HOST env var.
func DefaultHost() string {
	if h := os.Getenv("GITLAB_HOST"); h != "" {
		return h
	}
	return "gitlab.com"
}

// TokenForHost returns the authentication token for a given host.
func TokenForHost(host string) (string, string) {
	// Check environment variables first
	if t := os.Getenv("GITLAB_TOKEN"); t != "" {
		return t, "GITLAB_TOKEN"
	}
	if t := os.Getenv("GLAB_TOKEN"); t != "" {
		return t, "GLAB_TOKEN"
	}

	hosts, err := LoadHosts()
	if err != nil {
		return "", ""
	}
	if hc, ok := hosts[host]; ok {
		return hc.Token, host
	}
	return "", ""
}
