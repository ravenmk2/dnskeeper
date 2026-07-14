package config

import (
	"fmt"
	"os"
	"time"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Server  ServerConfig  `toml:"server"`
	Log     LogConfig     `toml:"log"`
	Etcd    EtcdConfig    `toml:"etcd"`
	CoreDNS CoreDNSConfig `toml:"coredns"`
	JWT     JWTConfig     `toml:"jwt"`
}

type ServerConfig struct {
	Listen string `toml:"listen"`
}

type LogConfig struct {
	Level string `toml:"level"`
}

type EtcdConfig struct {
	Endpoints []string `toml:"endpoints"`
	Username  string   `toml:"username"`
	Password  string   `toml:"password"`
	Cert      string   `toml:"cert"`
	Key       string   `toml:"key"`
	CA        string   `toml:"ca"`
}

func (c EtcdConfig) TLS() bool {
	return c.Cert != "" && c.Key != "" && c.CA != ""
}

type CoreDNSConfig struct {
	Path string `toml:"path"`
}

type JWTConfig struct {
	Secret     string `toml:"secret"`
	AccessTTL  string `toml:"access_ttl"`
	RefreshTTL string `toml:"refresh_ttl"`
}

func (j JWTConfig) ParseAccessTTL() (time.Duration, error) {
	d, err := time.ParseDuration(j.AccessTTL)
	if err != nil {
		return 0, fmt.Errorf("invalid access_ttl: %w", err)
	}
	return d, nil
}

func (j JWTConfig) ParseRefreshTTL() (time.Duration, error) {
	d, err := time.ParseDuration(j.RefreshTTL)
	if err != nil {
		return 0, fmt.Errorf("invalid refresh_ttl: %w", err)
	}
	return d, nil
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	var cfg Config
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (c *Config) validate() error {
	if c.Server.Listen == "" {
		return fmt.Errorf("server.listen is required")
	}
	if c.Log.Level == "" {
		c.Log.Level = "info"
	}
	if len(c.Etcd.Endpoints) == 0 {
		return fmt.Errorf("etcd.endpoints is required")
	}
	if c.CoreDNS.Path == "" {
		c.CoreDNS.Path = "/skydns"
	}
	if c.JWT.Secret == "" {
		return fmt.Errorf("jwt.secret is required")
	}
	if _, err := c.JWT.ParseAccessTTL(); err != nil {
		return err
	}
	if _, err := c.JWT.ParseRefreshTTL(); err != nil {
		return err
	}
	tlsCount := 0
	if c.Etcd.Cert != "" {
		tlsCount++
	}
	if c.Etcd.Key != "" {
		tlsCount++
	}
	if c.Etcd.CA != "" {
		tlsCount++
	}
	if tlsCount > 0 && tlsCount < 3 {
		return fmt.Errorf("etcd TLS config requires all of cert, key, and ca")
	}
	return nil
}
