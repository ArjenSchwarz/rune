package config

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	tests := map[string]struct {
		writeConfig bool
		content     string
		wantEnabled bool
		wantErr     bool
	}{
		"loads from current directory .rune.yml": {
			writeConfig: true,
			content: `
discovery:
  enabled: true
  template: "tasks/{branch}.md"
`,
			wantEnabled: true,
		},
		"returns default config when no file exists": {
			writeConfig: false,
			wantEnabled: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			resetConfigCache()
			t.Cleanup(func() { resetConfigCache() })

			// Isolate: work in a temp directory with a fake HOME
			tempDir := t.TempDir()
			fakeHome := t.TempDir()
			t.Setenv("HOME", fakeHome)

			// Init a git repo so getRepoRoot (which shells out to git rev-parse) works
			cmd := exec.Command("git", "-C", tempDir, "init")
			if err := cmd.Run(); err != nil {
				t.Fatalf("git init failed: %v", err)
			}

			t.Chdir(tempDir)

			if tc.writeConfig {
				if err := os.WriteFile(filepath.Join(tempDir, ".rune.yml"), []byte(tc.content), 0644); err != nil {
					t.Fatal(err)
				}
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
	// Use a fake HOME so the test is deterministic and never touches the real home
	fakeHome := t.TempDir()
	t.Setenv("HOME", fakeHome)

	tests := map[string]struct {
		input string
		want  string
	}{
		"expands tilde": {
			input: "~/config/file.yml",
			want:  filepath.Join(fakeHome, "config/file.yml"),
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
			got := expandHome(tc.input)
			if got != tc.want {
				t.Errorf("expandHome(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestConfigPrecedence(t *testing.T) {
	resetConfigCache()

	// Isolate: use temp directories so we never touch the real home
	tempDir := t.TempDir()
	fakeHome := t.TempDir()
	t.Setenv("HOME", fakeHome)
	t.Cleanup(func() { resetConfigCache() })

	// Init a git repo so getRepoRoot (which shells out to git rev-parse) works
	cmd := exec.Command("git", "-C", tempDir, "init")
	if err := cmd.Run(); err != nil {
		t.Fatalf("git init failed: %v", err)
	}
	t.Chdir(tempDir)

	// Create config in the "repo root" (should take precedence)
	localContent := `
discovery:
  enabled: true
  template: "local/{branch}.md"
`
	if err := os.WriteFile(filepath.Join(tempDir, ".rune.yml"), []byte(localContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a home config (should be ignored due to lower precedence)
	homeConfigDir := filepath.Join(fakeHome, ".config", "rune")
	if err := os.MkdirAll(homeConfigDir, 0755); err != nil {
		t.Fatal(err)
	}
	homeContent := `
discovery:
  enabled: false
  template: "home/{branch}.md"
`
	if err := os.WriteFile(filepath.Join(homeConfigDir, "config.yml"), []byte(homeContent), 0644); err != nil {
		t.Fatal(err)
	}

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

// TestLoadConfigFromSubdirectory verifies that config loading finds .rune.yml
// at the repo root when the CWD is a subdirectory. This is a regression test
// for T-482: loadConfigUncached uses relative path "./.rune.yml" which fails
// when CWD is not the repo root.
func TestLoadConfigFromSubdirectory(t *testing.T) {
	resetConfigCache()

	// Create a temp directory simulating a repo root
	tempDir := t.TempDir()
	t.Cleanup(func() { resetConfigCache() })

	// Create .rune.yml at the "repo root" with a distinctive template
	// that differs from the default, so we can prove it was actually loaded
	content := `
discovery:
  enabled: true
  template: "custom/{branch}/tasks.md"
`
	if err := os.WriteFile(filepath.Join(tempDir, ".rune.yml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a subdirectory and chdir into it
	subDir := filepath.Join(tempDir, "src", "pkg")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Initialize a git repo so getRepoRoot (which shells out to git rev-parse) works
	cmd := exec.Command("git", "-C", tempDir, "init")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	t.Chdir(subDir)

	cfg, err := loadConfigUncached()
	if err != nil {
		t.Fatalf("Expected config to load from subdirectory, got error: %v", err)
	}
	if !cfg.Discovery.Enabled {
		t.Error("Expected Discovery.Enabled to be true")
	}
	if cfg.Discovery.Template != "custom/{branch}/tasks.md" {
		t.Errorf("Expected template 'custom/{branch}/tasks.md' (from .rune.yml), got %q (likely default config — file not found from subdirectory)", cfg.Discovery.Template)
	}
}

// TestLoadConfigUncachedInvalidYAML verifies that when .rune.yml exists but
// contains invalid YAML, loadConfigUncached returns an error instead of
// silently falling back to defaults. Regression test for T-556.
func TestLoadConfigUncachedInvalidYAML(t *testing.T) {
	resetConfigCache()
	tempDir := t.TempDir()
	t.Cleanup(func() { resetConfigCache() })

	// Initialize git repo so getRepoRoot (which shells out to git rev-parse) works
	cmd := exec.Command("git", "-C", tempDir, "init")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	// Write invalid YAML to .rune.yml
	invalidYAML := `discovery:
  enabled: not-a-boolean
  template: [this is invalid
`
	if err := os.WriteFile(filepath.Join(tempDir, ".rune.yml"), []byte(invalidYAML), 0644); err != nil {
		t.Fatal(err)
	}

	t.Chdir(tempDir)

	cfg, err := loadConfigUncached()
	if err == nil {
		t.Fatalf("expected error for invalid YAML, got config: %+v", cfg)
	}
	if !contains(err.Error(), "parsing config file") {
		t.Errorf("error should mention parsing, got: %v", err)
	}
}

// TestLoadConfigUncachedUnknownFields verifies that when .rune.yml contains
// unknown fields, loadConfigUncached returns an error. Regression test for T-556.
func TestLoadConfigUncachedUnknownFields(t *testing.T) {
	resetConfigCache()
	tempDir := t.TempDir()
	t.Cleanup(func() { resetConfigCache() })

	// Initialize git repo so getRepoRoot (which shells out to git rev-parse) works
	cmd := exec.Command("git", "-C", tempDir, "init")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	// Write YAML with unknown field (typo in field name)
	unknownFieldYAML := `discovry:
  enabled: true
  template: "tasks/{branch}.md"
`
	if err := os.WriteFile(filepath.Join(tempDir, ".rune.yml"), []byte(unknownFieldYAML), 0644); err != nil {
		t.Fatal(err)
	}

	t.Chdir(tempDir)

	cfg, err := loadConfigUncached()
	if err == nil {
		t.Fatalf("expected error for unknown field 'discovry', got config: %+v", cfg)
	}
	if !contains(err.Error(), "parsing config file") {
		t.Errorf("error should mention parsing, got: %v", err)
	}
}

// TestLoadConfigFileUnknownFields verifies that loadConfigFile rejects
// YAML with unknown fields. Regression test for T-556.
func TestLoadConfigFileUnknownFields(t *testing.T) {
	tests := map[string]struct {
		content    string
		wantErr    bool
		wantErrMsg string
	}{
		"unknown top-level field": {
			content: `unknown_key: true
discovery:
  enabled: true
`,
			wantErr:    true,
			wantErrMsg: "parsing config file",
		},
		"unknown nested field": {
			content: `discovery:
  enabled: true
  tempalte: "typo/{branch}.md"
`,
			wantErr:    true,
			wantErrMsg: "parsing config file",
		},
		"valid fields still work": {
			content: `discovery:
  enabled: true
  template: "tasks/{branch}.md"
`,
			wantErr: false,
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

			_, err = loadConfigFile(tmpfile.Name())
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error but got none")
				}
				if tc.wantErrMsg != "" && !contains(err.Error(), tc.wantErrMsg) {
					t.Errorf("error = %v, want error containing %q", err, tc.wantErrMsg)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			}
		})
	}
}

// TestLoadConfigUncachedMissingFileStillDefaults verifies that when no
// .rune.yml exists, defaults are still returned (not broken by T-556 fix).
func TestLoadConfigUncachedMissingFileStillDefaults(t *testing.T) {
	resetConfigCache()
	tempDir := t.TempDir()
	t.Cleanup(func() { resetConfigCache() })

	// Initialize git repo but do NOT create .rune.yml
	cmd := exec.Command("git", "-C", tempDir, "init")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	t.Chdir(tempDir)

	cfg, err := loadConfigUncached()
	if err != nil {
		t.Fatalf("expected default config, got error: %v", err)
	}
	if !cfg.Discovery.Enabled {
		t.Error("default config should have Discovery.Enabled = true")
	}
	if cfg.Discovery.Template != "specs/{branch}/tasks.md" {
		t.Errorf("default template = %q, want %q", cfg.Discovery.Template, "specs/{branch}/tasks.md")
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
