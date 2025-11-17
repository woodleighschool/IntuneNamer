package config

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// Config holds device naming configuration.
type Config struct {
	Version          int              `yaml:"version"`
	Settings         Settings         `yaml:"settings"`
	Exclude          ExcludeConfig    `yaml:"exclude"`
	StaticNames      []StaticName     `yaml:"staticNames"`
	MetadataOverlays []MetadataConfig `yaml:"metadata"`
	RuleDefinitions  []RuleConfig     `yaml:"rules"`
}

type Settings struct {
	DuplicatePolicy DuplicatePolicy `yaml:"duplicatePolicy"`
}

type DuplicatePolicy struct {
	Scope      string          `yaml:"scope"`
	OnConflict string          `yaml:"onConflict"`
	Suffix     DuplicateSuffix `yaml:"suffix"`
}

type DuplicateSuffix struct {
	Format string `yaml:"format"`
	Min    int    `yaml:"min"`
	Max    int    `yaml:"max"`
}

type ExcludeConfig struct {
	Serials   []string `yaml:"serials"`
	Users     []string `yaml:"users"`
	DeviceIDs []string `yaml:"deviceIds"`
}

type StaticName struct {
	Serial  string `yaml:"serial"`
	Name    string `yaml:"name"`
	Enforce bool   `yaml:"enforce"`
}

type MetadataConfig struct {
	Name     string            `yaml:"name"`
	Priority int               `yaml:"priority"`
	Match    MetadataMatch     `yaml:"match"`
	Values   map[string]string `yaml:"values"`
}

type MetadataMatch struct {
	AnyGroup   []string
	AllGroup   []string
	Attributes MatcherConfig
}

type RuleConfig struct {
	Name            string           `yaml:"name"`
	Priority        int              `yaml:"priority"`
	Match           MatcherConfig    `yaml:"match"`
	Template        string           `yaml:"template"`
	MaxLength       int              `yaml:"maxLength"`
	DuplicatePolicy *DuplicatePolicy `yaml:"duplicatePolicy"`
	StopProcessing  bool             `yaml:"stopProcessing"`
}

// MatcherConfig maps attributes to allowed values.
type MatcherConfig map[string]StringList

// StringList handles both single values and arrays in YAML.
type StringList []string

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	cfg.applyDefaults()
	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func (c *Config) validate() error {
	if c.Version <= 0 {
		return fmt.Errorf("config version must be set (found %d)", c.Version)
	}
	if len(c.RuleDefinitions) == 0 {
		return fmt.Errorf("at least one rule must be defined")
	}
	if err := c.Settings.DuplicatePolicy.validate(); err != nil {
		return fmt.Errorf("settings.duplicatePolicy: %w", err)
	}
	for i, r := range c.RuleDefinitions {
		if strings.TrimSpace(r.Template) == "" {
			return fmt.Errorf("rules[%d] template cannot be empty", i)
		}
		if r.DuplicatePolicy != nil {
			if err := r.DuplicatePolicy.validate(); err != nil {
				return fmt.Errorf("rules[%d].duplicatePolicy: %w", i, err)
			}
		}
	}
	return nil
}

func (c *Config) applyDefaults() {
	if c.Version == 0 {
		c.Version = 1
	}
	c.Settings.DuplicatePolicy.applyDefaults()
	for i := range c.RuleDefinitions {
		if c.RuleDefinitions[i].DuplicatePolicy != nil {
			c.RuleDefinitions[i].DuplicatePolicy.applyDefaults()
		}
	}
}

func (s *StringList) UnmarshalYAML(value *yaml.Node) error {
	switch value.Kind {
	case yaml.ScalarNode:
		*s = StringList{strings.TrimSpace(value.Value)}
	case yaml.SequenceNode:
		items := make([]string, 0, len(value.Content))
		for _, node := range value.Content {
			if node == nil {
				continue
			}
			items = append(items, strings.TrimSpace(node.Value))
		}
		*s = items
	default:
		return fmt.Errorf("string list must be scalar or sequence")
	}
	return nil
}

func (m *MetadataMatch) UnmarshalYAML(value *yaml.Node) error {
	if value == nil {
		return nil
	}
	if value.Kind != yaml.MappingNode {
		return fmt.Errorf("metadata match must be a map")
	}
	attrs := make(MatcherConfig)
	for i := 0; i < len(value.Content); i += 2 {
		keyNode := value.Content[i]
		valNode := value.Content[i+1]
		if keyNode == nil || valNode == nil {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(keyNode.Value))
		switch key {
		case "anygroup":
			var list StringList
			if err := valNode.Decode(&list); err != nil {
				return fmt.Errorf("anyGroup: %w", err)
			}
			m.AnyGroup = append(m.AnyGroup, list...)
		case "allgroup":
			var list StringList
			if err := valNode.Decode(&list); err != nil {
				return fmt.Errorf("allGroup: %w", err)
			}
			m.AllGroup = append(m.AllGroup, list...)
		case "attributes":
			var set MatcherConfig
			if err := valNode.Decode(&set); err != nil {
				return fmt.Errorf("attributes: %w", err)
			}
			for attrKey, list := range set {
				attrs[attrKey] = list
			}
		default:
			var list StringList
			if err := valNode.Decode(&list); err != nil {
				return fmt.Errorf("match %s: %w", key, err)
			}
			attrs[key] = list
		}
	}
	if len(attrs) == 0 {
		attrs = nil
	}
	m.Attributes = attrs
	return nil
}

func (p *DuplicatePolicy) applyDefaults() {
	if p == nil {
		return
	}
	p.Scope = strings.ToLower(strings.TrimSpace(p.Scope))
	if p.Scope == "" {
		p.Scope = "global"
	}
	p.OnConflict = strings.ToLower(strings.TrimSpace(p.OnConflict))
	if p.OnConflict == "" {
		p.OnConflict = "append_suffix"
	}
	if p.Suffix.Min <= 0 {
		p.Suffix.Min = 1
	}
	if p.Suffix.Max <= 0 {
		p.Suffix.Max = 9
	}
	if strings.TrimSpace(p.Suffix.Format) == "" {
		p.Suffix.Format = "-%d"
	}
}

func (p *DuplicatePolicy) validate() error {
	if p.OnConflict == "" {
		return fmt.Errorf("onConflict must be specified")
	}
	switch p.OnConflict {
	case "append_suffix", "skip", "error", "overwrite":
	default:
		return fmt.Errorf("unsupported onConflict strategy %q", p.OnConflict)
	}
	switch p.Scope {
	case "global", "per-user", "per-platform":
	default:
		return fmt.Errorf("unsupported scope %q", p.Scope)
	}
	if p.OnConflict == "append_suffix" {
		if p.Suffix.Min > p.Suffix.Max {
			return fmt.Errorf("suffix min (%d) cannot exceed max (%d)", p.Suffix.Min, p.Suffix.Max)
		}
		if !strings.Contains(p.Suffix.Format, "%") {
			return fmt.Errorf("suffix format must contain a %% placeholder (got %q)", p.Suffix.Format)
		}
	}
	return nil
}

// AppConfig contains application settings.
type AppConfig struct {
	// Azure AD configuration
	TenantID     string `mapstructure:"tenant_id"`
	ClientID     string `mapstructure:"client_id"`
	ClientSecret string `mapstructure:"client_secret"`

	// Application configuration
	ConfigPath    string        `mapstructure:"config"`
	Once          bool          `mapstructure:"once"`
	PollInterval  time.Duration `mapstructure:"poll_interval"`
	LogLevel      string        `mapstructure:"log_level"`
	DryRun        bool          `mapstructure:"dry_run"`
	MaxNameLength int           `mapstructure:"max_name_length"`
}

func LoadAppConfig() (*AppConfig, error) {
	v := viper.GetViper()

	v.SetDefault("config", "config.yaml")
	v.SetDefault("once", false)
	v.SetDefault("poll_interval", "5m")
	v.SetDefault("log_level", "info")
	v.SetDefault("dry_run", false)
	v.SetDefault("max_name_length", 63)
	v.SetDefault("tenant_id", "")
	v.SetDefault("client_id", "")
	v.SetDefault("client_secret", "")

	var cfg AppConfig
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal app config: %w", err)
	}

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("app config validation failed: %w", err)
	}

	return &cfg, nil
}

func (c *AppConfig) validate() error {
	var errors []string

	if c.TenantID == "" {
		errors = append(errors, "TENANT_ID is required")
	}
	if c.ClientID == "" {
		errors = append(errors, "CLIENT_ID is required")
	}
	if c.ClientSecret == "" {
		errors = append(errors, "CLIENT_SECRET is required")
	}

	// Validate log level
	validLogLevels := []string{"debug", "info", "warn", "error"}
	levelValid := false
	for _, level := range validLogLevels {
		if strings.ToLower(c.LogLevel) == level {
			levelValid = true
			break
		}
	}
	if !levelValid {
		errors = append(errors, fmt.Sprintf("LOG_LEVEL must be one of: %s", strings.Join(validLogLevels, ", ")))
	}

	if c.PollInterval <= 0 {
		errors = append(errors, "poll interval must be greater than zero")
	}

	if c.MaxNameLength <= 0 {
		errors = append(errors, "max-name-length must be a positive integer")
	}

	if len(errors) > 0 {
		return fmt.Errorf("validation errors:\n  - %s", strings.Join(errors, "\n  - "))
	}

	return nil
}

func (c *AppConfig) GetLogLevel() slog.Level {
	switch strings.ToLower(c.LogLevel) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func (c *AppConfig) GetTimeout() time.Duration {
	return 5 * time.Minute
}

func (c *AppConfig) IsOneshot() bool {
	return c.Once
}
