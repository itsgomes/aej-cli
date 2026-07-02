package config

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestLoadClassifiesMissingConfiguration(t *testing.T) {
	setHome(t, t.TempDir())

	_, err := Load()
	if !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("Load() error = %v, want ErrNotConfigured", err)
	}
}

func TestLoadClassifiesIncompleteConfiguration(t *testing.T) {
	setHome(t, t.TempDir())

	dir, err := configDir()
	if err != nil {
		t.Fatalf("configDir() error = %v", err)
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte("jira_url: https://example.atlassian.net\napi_token: token\n"), 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	_, err = Load()
	if !errors.Is(err, ErrIncomplete) {
		t.Fatalf("Load() error = %v, want ErrIncomplete", err)
	}
}

func TestNormalizeJiraURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "host without scheme", input: "example.atlassian.net", want: "https://example.atlassian.net"},
		{name: "trailing slash", input: " https://example.atlassian.net/ ", want: "https://example.atlassian.net"},
		{name: "insecure HTTP", input: "http://example.atlassian.net", wantErr: true},
		{name: "embedded credentials", input: "https://user:token@example.atlassian.net", wantErr: true},
		{name: "path", input: "https://example.atlassian.net/jira", wantErr: true},
		{name: "query", input: "https://example.atlassian.net?debug=true", wantErr: true},
		{name: "fragment", input: "https://example.atlassian.net#fragment", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := NormalizeJiraURL(tt.input)
			if tt.wantErr {
				if !errors.Is(err, ErrInvalidURL) {
					t.Fatalf("NormalizeJiraURL() error = %v, want ErrInvalidURL", err)
				}
				return
			}

			if err != nil {
				t.Fatalf("NormalizeJiraURL() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("NormalizeJiraURL() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestLoadFromEnvironmentWithoutConfigFile(t *testing.T) {
	setHome(t, t.TempDir())
	t.Setenv("AEJ_JIRA_URL", "environment.atlassian.net")
	t.Setenv("AEJ_EMAIL", "dev@example.com")
	t.Setenv("AEJ_API_TOKEN", "environment-token")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.JiraURL != "https://environment.atlassian.net" {
		t.Errorf("JiraURL = %q, want normalized environment URL", cfg.JiraURL)
	}
	if cfg.Email != "dev@example.com" || cfg.APIToken != "environment-token" {
		t.Errorf("credentials = (%q, %q), want environment credentials", cfg.Email, cfg.APIToken)
	}
}

func TestSavePersistsNormalizedConfiguration(t *testing.T) {
	setHome(t, t.TempDir())

	err := Save(Config{
		JiraURL:  "example.atlassian.net/",
		Email:    " dev@example.com ",
		APIToken: " token ",
	})
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if loaded.JiraURL != "https://example.atlassian.net" || loaded.Email != "dev@example.com" || loaded.APIToken != "token" {
		t.Errorf("Load() = %#v, want normalized persisted configuration", loaded)
	}

	if runtime.GOOS != "windows" {
		dir, err := configDir()
		if err != nil {
			t.Fatalf("configDir() error = %v", err)
		}
		info, err := os.Stat(filepath.Join(dir, "config.yaml"))
		if err != nil {
			t.Fatalf("Stat() error = %v", err)
		}
		if got := info.Mode().Perm(); got != 0600 {
			t.Errorf("config permissions = %o, want 600", got)
		}
	}
}

func setHome(t *testing.T, dir string) {
	t.Helper()
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir)
	t.Setenv("AEJ_JIRA_URL", "")
	t.Setenv("AEJ_EMAIL", "")
	t.Setenv("AEJ_API_TOKEN", "")
}
