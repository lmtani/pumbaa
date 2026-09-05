package config

import (
	"errors"
	"strings"
	"testing"
)

func TestNormalizeHostURL(t *testing.T) {
	tests := []struct {
		raw  string
		want string
	}{
		{"http://localhost:8000", "http://localhost:8000"},
		{"http://localhost:8000/", "http://localhost:8000"},
		{"localhost:8000", "http://localhost:8000"},
		{" https://cromwell.example.com/ ", "https://cromwell.example.com"},
		{"", ""},
	}
	for _, tt := range tests {
		if got := NormalizeHostURL(tt.raw); got != tt.want {
			t.Errorf("NormalizeHostURL(%q) = %q, want %q", tt.raw, got, tt.want)
		}
	}
}

func TestResolveHostAliasWinsOverURLGuess(t *testing.T) {
	cfg := &FileConfig{Hosts: map[string]string{"prod": "https://cromwell.example.com"}}

	ref, err := cfg.ResolveHost("prod")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ref.Alias != "prod" || ref.URL != "https://cromwell.example.com" {
		t.Errorf("got %+v, want alias prod", ref)
	}
	if ref.Display() != "prod" {
		t.Errorf("Display() = %q, want the alias", ref.Display())
	}
}

func TestResolveHostAcceptsURL(t *testing.T) {
	cfg := &FileConfig{}

	ref, err := cfg.ResolveHost("cromwell.internal:8000")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ref.Alias != "" {
		t.Errorf("alias = %q, want empty for a URL", ref.Alias)
	}
	if ref.URL != "http://cromwell.internal:8000" {
		t.Errorf("URL = %q, want the normalized URL", ref.URL)
	}
	if ref.Display() != ref.URL {
		t.Errorf("Display() = %q, want the URL when there is no alias", ref.Display())
	}
}

func TestResolveHostUnknownAliasSuggests(t *testing.T) {
	cfg := &FileConfig{Hosts: map[string]string{"prod": "https://a", "staging": "https://b"}}

	_, err := cfg.ResolveHost("prd")
	var unknown *UnknownHostError
	if !errors.As(err, &unknown) {
		t.Fatalf("got %v, want UnknownHostError", err)
	}
	if !strings.Contains(err.Error(), `did you mean "prod"`) {
		t.Errorf("message %q does not suggest the near match", err.Error())
	}
	if !strings.Contains(err.Error(), "staging") {
		t.Errorf("message %q does not list the registered hosts", err.Error())
	}
}

func TestResolveHostDefaultPrecedence(t *testing.T) {
	tests := []struct {
		name string
		cfg  *FileConfig
		want HostRef
	}{
		{
			name: "default alias",
			cfg: &FileConfig{
				Hosts:        map[string]string{"prod": "https://cromwell.example.com"},
				DefaultHost:  "prod",
				CromwellHost: "http://legacy:8000",
			},
			want: HostRef{Alias: "prod", URL: "https://cromwell.example.com"},
		},
		{
			name: "legacy single-host setting when no default alias",
			cfg:  &FileConfig{CromwellHost: "http://legacy:8000"},
			want: HostRef{URL: "http://legacy:8000"},
		},
		{
			name: "built-in default",
			cfg:  &FileConfig{},
			want: HostRef{URL: DefaultCromwellHost},
		},
		{
			name: "default pointing at a removed alias falls through",
			cfg:  &FileConfig{DefaultHost: "gone", CromwellHost: "http://legacy:8000"},
			want: HostRef{URL: "http://legacy:8000"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.cfg.ResolveHost("")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestAddHostMakesFirstOneDefault(t *testing.T) {
	cfg := &FileConfig{}

	if err := cfg.AddHost("local", "localhost:8000"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.DefaultHost != "local" {
		t.Errorf("DefaultHost = %q, want the only registered host", cfg.DefaultHost)
	}
	if cfg.Hosts["local"] != "http://localhost:8000" {
		t.Errorf("stored URL = %q, want it normalized", cfg.Hosts["local"])
	}

	if err := cfg.AddHost("prod", "https://cromwell.example.com"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.DefaultHost != "local" {
		t.Errorf("DefaultHost = %q, adding a second host must not steal the default", cfg.DefaultHost)
	}
}

func TestAddHostRejectsAmbiguousAlias(t *testing.T) {
	cfg := &FileConfig{}
	for _, alias := range []string{"", "http://x", "a.b", "has space"} {
		if err := cfg.AddHost(alias, "http://x"); err == nil {
			t.Errorf("AddHost(%q) succeeded, want rejection", alias)
		}
	}
	if err := cfg.AddHost("prod", ""); err == nil {
		t.Error("AddHost with an empty URL succeeded, want rejection")
	}
}

func TestRemoveHostMovesDefault(t *testing.T) {
	cfg := &FileConfig{}
	_ = cfg.AddHost("local", "http://localhost:8000")
	_ = cfg.AddHost("prod", "https://cromwell.example.com")

	if err := cfg.RemoveHost("local"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.DefaultHost != "prod" {
		t.Errorf("DefaultHost = %q, want the remaining host", cfg.DefaultHost)
	}

	if err := cfg.RemoveHost("prod"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.DefaultHost != "" {
		t.Errorf("DefaultHost = %q, want empty once nothing is registered", cfg.DefaultHost)
	}
	if err := cfg.RemoveHost("prod"); err == nil {
		t.Error("removing an unregistered host succeeded, want an error")
	}
}

func TestSetDefaultHostRequiresRegistration(t *testing.T) {
	cfg := &FileConfig{}
	if err := cfg.SetDefaultHost("prod"); err == nil {
		t.Error("SetDefaultHost on an unregistered alias succeeded, want an error")
	}
	if err := cfg.SetValue("default_host", "prod"); err == nil {
		t.Error("config set default_host on an unregistered alias succeeded, want an error")
	}
}
