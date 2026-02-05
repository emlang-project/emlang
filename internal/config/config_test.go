package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseFullConfig(t *testing.T) {
	dir := t.TempDir()
	cfgFile := filepath.Join(dir, ".emlang.yaml")
	content := `lint:
  ignore:
    - "command-without-event"
    - "orphan-exception"
diagram:
  css:
    --trigger-color: "#f0f0f0"
    --command-color: "#ddeeff"
`
	if err := os.WriteFile(cfgFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(cfgFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(cfg.Lint.Ignore) != 2 {
		t.Errorf("expected 2 ignored rules, got %d", len(cfg.Lint.Ignore))
	}
	if cfg.Lint.Ignore[0] != "command-without-event" {
		t.Errorf("expected first ignore rule 'command-without-event', got %q", cfg.Lint.Ignore[0])
	}
	if len(cfg.Diagram.CSS) != 2 {
		t.Errorf("expected 2 CSS overrides, got %d", len(cfg.Diagram.CSS))
	}
	if cfg.Diagram.CSS["--trigger-color"] != "#f0f0f0" {
		t.Errorf("unexpected --trigger-color: %q", cfg.Diagram.CSS["--trigger-color"])
	}
}

func TestParseMinimalConfig(t *testing.T) {
	dir := t.TempDir()
	cfgFile := filepath.Join(dir, ".emlang.yaml")
	content := `lint:
  ignore: []
`
	if err := os.WriteFile(cfgFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(cfgFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(cfg.Lint.Ignore) != 0 {
		t.Errorf("expected no ignored rules, got %d", len(cfg.Lint.Ignore))
	}
	if len(cfg.Diagram.CSS) != 0 {
		t.Errorf("expected no CSS overrides, got %d", len(cfg.Diagram.CSS))
	}
}

func TestLoadNoFileReturnsDefaults(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	t.Setenv("EMLANG_CONFIG", "")

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_ = cfg // should load without error
}

func TestLoadMissingExplicitPathErrors(t *testing.T) {
	_, err := Load("/nonexistent/path/.emlang.yaml")
	if err == nil {
		t.Fatal("expected error for missing explicit path")
	}
}

func TestLoadEnvVar(t *testing.T) {
	dir := t.TempDir()
	cfgFile := filepath.Join(dir, "custom.yaml")
	content := `lint:
  ignore:
    - "from-env"
`
	if err := os.WriteFile(cfgFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("EMLANG_CONFIG", cfgFile)

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(cfg.Lint.Ignore) != 1 || cfg.Lint.Ignore[0] != "from-env" {
		t.Errorf("expected ignore rule from env, got %v", cfg.Lint.Ignore)
	}
}

func TestLoadFlagOverridesEnv(t *testing.T) {
	dir := t.TempDir()

	envFile := filepath.Join(dir, "env.yaml")
	if err := os.WriteFile(envFile, []byte(`lint:
  ignore:
    - "from-env"
`), 0644); err != nil {
		t.Fatal(err)
	}

	flagFile := filepath.Join(dir, "flag.yaml")
	if err := os.WriteFile(flagFile, []byte(`lint:
  ignore:
    - "from-flag"
`), 0644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("EMLANG_CONFIG", envFile)

	cfg, err := Load(flagFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(cfg.Lint.Ignore) != 1 || cfg.Lint.Ignore[0] != "from-flag" {
		t.Errorf("expected ignore rule 'from-flag', got %v", cfg.Lint.Ignore)
	}
}

func TestLoadMissingEnvPathErrors(t *testing.T) {
	t.Setenv("EMLANG_CONFIG", "/nonexistent/env-config.yaml")

	_, err := Load("")
	if err == nil {
		t.Fatal("expected error for missing env path")
	}
}

func TestLoadInvalidYAMLErrors(t *testing.T) {
	dir := t.TempDir()
	cfgFile := filepath.Join(dir, ".emlang.yaml")
	if err := os.WriteFile(cfgFile, []byte(`{invalid: yaml: [`), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(cfgFile)
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}
