package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	tests := map[string]struct {
		setup       func(t *testing.T) string
		cleanup     func(string)
		wantEnabled bool
		wantErr     bool
	}{
		"loads from current directory .rune.yml": {
			setup: func(t *testing.T) string {
				resetConfigCache() // Reset cache before test
				content := `
discovery:
  enabled: true
  template: "tasks/{branch}.md"
`
				err := os.WriteFile(".rune.yml", []byte(content), 0644)
				if err != nil {
					t.Fatal(err)
				}
				return ".rune.yml"
			},
			cleanup: func(path string) {
				os.Remove(path)
			},
			wantEnabled: true,
		},
		"returns default config when no file exists": {
			setup: func(t *testing.T) string {
				resetConfigCache() // Reset cache before test
				// Ensure no config files exist
				os.Remove(".rune.yml")
				return ""
			},
			cleanup:     func(string) {},
			wantEnabled: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			path := tc.setup(t)
			if tc.cleanup != nil {
				defer tc.cleanup(path)
			}

			cfg, err := LoadConfig()
			if tc.wantErr && err == nil {
				t.Fatal("expected error but got none")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if cfg == nil {
				t.Fatal("expected config but got nil")
			}
			if cfg.Discovery.Enabled != tc.wantEnabled {
				t.Errorf("Discovery.Enabled = %v, want %v", cfg.Discovery.Enabled, tc.wantEnabled)
			}
		})
	}
}

func TestLoadConfigFile(t *testing.T) {
	tests := map[string]struct {
		content     string
		wantEnabled bool
		wantErr     bool
		wantErrMsg  string
	}{
		"valid YAML": {
			content: `
discovery:
  enabled: true
  template: "custom/{branch}/tasks.md"
`,
			wantEnabled: true,
		},
		"invalid YAML": {
			content: `
discovery:
  enabled: not-a-boolean
  template: [this is invalid
`,
			wantErr:    true,
			wantErrMsg: "parsing config file",
		},
		"empty file uses defaults": {
			content:     "",
			wantEnabled: false,
		},
		"partial config uses defaults": {
			content: `
discovery:
  enabled: true
`,
			wantEnabled: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			tmpfile, err := os.CreateTemp("", "config-*.yml")
			if err != nil {
				t.Fatal(err)
			}
			defer os.Remove(tmpfile.Name())

			if _, err := tmpfile.Write([]byte(tc.content)); err != nil {
				t.Fatal(err)
			}
			if err := tmpfile.Close(); err != nil {
				t.Fatal(err)
			}

			cfg, err := loadConfigFile(tmpfile.Name())
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error but got none")
				}
				if tc.wantErrMsg != "" && !contains(err.Error(), tc.wantErrMsg) {
					t.Errorf("error = %v, want error containing %q", err, tc.wantErrMsg)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if cfg.Discovery.Enabled != tc.wantEnabled {
				t.Errorf("Discovery.Enabled = %v, want %v", cfg.Discovery.Enabled, tc.wantEnabled)
			}
			if cfg.Discovery.Template == "" {
				t.Error("Discovery.Template should have default value")
			}
		})
	}
}

func TestLoadConfigFileNotFound(t *testing.T) {
	_, err := loadConfigFile("/non/existent/file.yml")
	if err == nil {
		t.Fatal("expected error for non-existent file")
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := defaultConfig()
	if cfg == nil {
		t.Fatal("defaultConfig returned nil")
	}
	if !cfg.Discovery.Enabled {
		t.Error("default config should have Discovery.Enabled = true")
	}
	if cfg.Discovery.Template != "specs/{branch}/tasks.md" {
		t.Errorf("Discovery.Template = %q, want %q", cfg.Discovery.Template, "specs/{branch}/tasks.md")
	}
}

func TestExpandHome(t *testing.T) {
	tests := map[string]struct {
		input string
		want  string
	}{
		"expands tilde": {
			input: "~/config/file.yml",
			want:  filepath.Join(os.Getenv("HOME"), "config/file.yml"),
		},
		"no tilde": {
			input: "/absolute/path/file.yml",
			want:  "/absolute/path/file.yml",
		},
		"relative path": {
			input: "./relative/file.yml",
			want:  "./relative/file.yml",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Set HOME env var for consistent testing
			home := os.Getenv("HOME")
			if home == "" {
				home, _ = os.UserHomeDir()
			}

			got := expandHome(tc.input)

			// For tilde expansion, check that it starts with home dir
			if tc.input[:2] == "~/" {
				if !contains(got, home) {
					t.Errorf("expandHome(%q) = %q, should contain home dir %q", tc.input, got, home)
				}
			} else {
				if got != tc.want {
					t.Errorf("expandHome(%q) = %q, want %q", tc.input, got, tc.want)
				}
			}
		})
	}
}

func TestConfigPrecedence(t *testing.T) {
	resetConfigCache() // Reset cache before test

	// Create config in current directory
	localContent := `
discovery:
  enabled: true
  template: "local/{branch}.md"
`
	err := os.WriteFile(".rune.yml", []byte(localContent), 0644)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(".rune.yml")

	// Also create a home config (which should be ignored)
	homeDir, _ := os.UserHomeDir()
	homeConfigDir := filepath.Join(homeDir, ".config", "rune")
	os.MkdirAll(homeConfigDir, 0755)
	homeConfigPath := filepath.Join(homeConfigDir, "config.yml")
	homeContent := `
discovery:
  enabled: false
  template: "home/{branch}.md"
`
	err = os.WriteFile(homeConfigPath, []byte(homeContent), 0644)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(homeConfigPath)

	// Load config - should use local file
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cfg.Discovery.Enabled {
		t.Error("should use local config with enabled=true")
	}
	if cfg.Discovery.Template != "local/{branch}.md" {
		t.Errorf("Discovery.Template = %q, want %q", cfg.Discovery.Template, "local/{branch}.md")
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || (len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
