package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validConfig() *Config {
	return &Config{
		Server:  ServerConfig{Listen: ":8080"},
		Log:     LogConfig{Level: "info"},
		Etcd:    EtcdConfig{Endpoints: []string{"127.0.0.1:2379"}},
		CoreDNS: CoreDNSConfig{Path: "/skydns"},
		JWT:     JWTConfig{Secret: "s", AccessTTL: "30m", RefreshTTL: "168h"},
	}
}

func TestValidateSuccess(t *testing.T) {
	c := validConfig()
	require.NoError(t, c.validate())
}

func TestValidateDefaults(t *testing.T) {
	c := validConfig()
	c.Log.Level = ""
	c.CoreDNS.Path = ""
	require.NoError(t, c.validate())
	assert.Equal(t, "info", c.Log.Level)
	assert.Equal(t, "/skydns", c.CoreDNS.Path)
}

func TestValidateErrors(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*Config)
		errSub string
	}{
		{"missing_listen", func(c *Config) { c.Server.Listen = "" }, "server.listen is required"},
		{"missing_endpoints", func(c *Config) { c.Etcd.Endpoints = nil }, "etcd.endpoints is required"},
		{"missing_secret", func(c *Config) { c.JWT.Secret = "" }, "jwt.secret is required"},
		{"invalid_access_ttl", func(c *Config) { c.JWT.AccessTTL = "abc" }, "invalid access_ttl"},
		{"empty_access_ttl", func(c *Config) { c.JWT.AccessTTL = "" }, "invalid access_ttl"},
		{"invalid_refresh_ttl", func(c *Config) { c.JWT.RefreshTTL = "30" }, "invalid refresh_ttl"},
		{"tls_partial_cert_only", func(c *Config) { c.Etcd.Cert = "/c" }, "etcd TLS config requires"},
		{"tls_partial_cert_key", func(c *Config) { c.Etcd.Cert = "/c"; c.Etcd.Key = "/k" }, "etcd TLS config requires"},
		{"tls_partial_cert_ca", func(c *Config) { c.Etcd.Cert = "/c"; c.Etcd.CA = "/a" }, "etcd TLS config requires"},
		{"tls_partial_key_ca", func(c *Config) { c.Etcd.Key = "/k"; c.Etcd.CA = "/a" }, "etcd TLS config requires"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := validConfig()
			tc.mutate(c)
			err := c.validate()
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.errSub)
		})
	}
}

func TestValidateTLSAllSet(t *testing.T) {
	c := validConfig()
	c.Etcd.Cert = "/c"
	c.Etcd.Key = "/k"
	c.Etcd.CA = "/a"
	require.NoError(t, c.validate())
}

func TestEtcdConfigTLS(t *testing.T) {
	cases := []struct {
		cert, key, ca string
		want          bool
	}{
		{"", "", "", false},
		{"/c", "", "", false},
		{"", "/k", "", false},
		{"", "", "/a", false},
		{"/c", "/k", "", false},
		{"/c", "", "/a", false},
		{"", "/k", "/a", false},
		{"/c", "/k", "/a", true},
	}
	for _, tc := range cases {
		got := EtcdConfig{Cert: tc.cert, Key: tc.key, CA: tc.ca}.TLS()
		assert.Equal(t, tc.want, got, "cert=%q key=%q ca=%q", tc.cert, tc.key, tc.ca)
	}
}

func TestParseTTLs(t *testing.T) {
	cases := []struct {
		ttl    string
		want   time.Duration
		wantOK bool
	}{
		{"30m", 30 * time.Minute, true},
		{"168h", 168 * time.Hour, true},
		{"1h30m", 90 * time.Minute, true},
		{"", 0, false},
		{"abc", 0, false},
		{"30", 0, false},
	}
	for _, tc := range cases {
		j := JWTConfig{AccessTTL: tc.ttl}
		got, err := j.ParseAccessTTL()
		if tc.wantOK {
			require.NoError(t, err, "access_ttl=%q", tc.ttl)
			assert.Equal(t, tc.want, got)
		} else {
			assert.Error(t, err, "access_ttl=%q", tc.ttl)
		}
		j = JWTConfig{RefreshTTL: tc.ttl}
		got, err = j.ParseRefreshTTL()
		if tc.wantOK {
			require.NoError(t, err, "refresh_ttl=%q", tc.ttl)
			assert.Equal(t, tc.want, got)
		} else {
			assert.Error(t, err, "refresh_ttl=%q", tc.ttl)
		}
	}
}

func TestLoadGoldenSamples(t *testing.T) {
	for _, path := range []string{"../../config.example.toml", "../../config.docker.toml"} {
		cfg, err := Load(path)
		require.NoError(t, err, "load %s", path)
		require.NotNil(t, cfg)
		assert.NotEmpty(t, cfg.Server.Listen)
		assert.NotEmpty(t, cfg.JWT.Secret)
		assert.NotEmpty(t, cfg.Etcd.Endpoints)
	}
}

func TestLoadMissingFile(t *testing.T) {
	_, err := Load("nonexistent-config.toml")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "read config")
}

func TestLoadMalformedTOML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.toml")
	require.NoError(t, os.WriteFile(path, []byte("not = valid = toml = ["), 0o644))
	_, err := Load(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse config")
}

func TestLoadValidationFails(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.toml")
	content := `
[server]
listen = ""
[etcd]
endpoints = []
[jwt]
secret = ""
`
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	_, err := Load(path)
	require.Error(t, err)
}

func TestLoadValidRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "good.toml")
	content := `
[server]
listen = ":9090"
[log]
level = "debug"
[etcd]
endpoints = ["127.0.0.1:2379"]
[coredns]
path = "/skydns"
[jwt]
secret = "my-secret"
access_ttl = "15m"
refresh_ttl = "720h"
`
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	cfg, err := Load(path)
	require.NoError(t, err)
	require.NotNil(t, cfg)
	assert.Equal(t, ":9090", cfg.Server.Listen)
	assert.Equal(t, "debug", cfg.Log.Level)
	assert.Equal(t, "/skydns", cfg.CoreDNS.Path)
	assert.Equal(t, "my-secret", cfg.JWT.Secret)
	assert.Equal(t, []string{"127.0.0.1:2379"}, cfg.Etcd.Endpoints)
}
