package cache

import (
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Duration parses YAML duration strings (e.g. "30s") into a time.Duration.
type Duration time.Duration

func (d *Duration) UnmarshalYAML(unmarshal func(any) error) error {
	var s string
	if err := unmarshal(&s); err != nil {
		return err
	}
	parsed, err := time.ParseDuration(s)
	if err != nil {
		return err
	}
	*d = Duration(parsed)
	return nil
}

// EntrySettings holds the tunable parameters for a single named cache.
// Nil fields mean the setting was not configured, at neither the specific
// nor the default level, and should not be applied.
type EntrySettings struct {
	Name    string    `yaml:"name,omitempty"`
	TTL     *Duration `yaml:"ttl,omitempty"`
	Check   *Duration `yaml:"check,omitempty"`
	MaxSize *int      `yaml:"maxSize,omitempty"`
}

// Config configures a set of named caches plus shared defaults.
type Config struct {
	Default EntrySettings   `yaml:"default"`
	Caches  []EntrySettings `yaml:"caches"`
}

// LoadConfig reads a cache Config from a YAML file.
func LoadConfig(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// SettingsFor resolves settings for name: a field set on the matching entry in
// Caches wins, otherwise the Default section is used, otherwise it stays unset.
func (c *Config) SettingsFor(name string) EntrySettings {
	if c == nil {
		return EntrySettings{Name: name}
	}

	var specific EntrySettings
	for i := range c.Caches {
		if c.Caches[i].Name == name {
			specific = c.Caches[i]
			break
		}
	}

	return EntrySettings{
		Name:    name,
		TTL:     firstDuration(specific.TTL, c.Default.TTL),
		Check:   firstDuration(specific.Check, c.Default.Check),
		MaxSize: firstInt(specific.MaxSize, c.Default.MaxSize),
	}
}

// OptionsFromSettings translates resolved EntrySettings into Cache options,
// applying only the settings that were actually configured.
func OptionsFromSettings[K comparable, V any](s EntrySettings) []Option[K, V] {
	var opts []Option[K, V]
	if s.MaxSize != nil {
		opts = append(opts, WithCapacity[K, V](*s.MaxSize))
	}
	if s.TTL != nil && s.Check != nil {
		opts = append(opts, WithTTL[K, V](time.Duration(*s.TTL), time.Duration(*s.Check)))
	}
	return opts
}

func firstDuration(vals ...*Duration) *Duration {
	for _, v := range vals {
		if v != nil {
			return v
		}
	}
	return nil
}

func firstInt(vals ...*int) *int {
	for _, v := range vals {
		if v != nil {
			return v
		}
	}
	return nil
}
