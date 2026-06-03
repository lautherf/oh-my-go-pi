package plugin

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseSpec_Plain(t *testing.T) {
	spec := ParseSpec("my-plugin")
	if spec.PackageName != "my-plugin" {
		t.Fatalf("expected packageName=my-plugin, got %s", spec.PackageName)
	}
	if spec.Features != nil {
		t.Fatalf("expected features=nil, got %v", spec.Features)
	}
}

func TestParseSpec_WithFeatures(t *testing.T) {
	spec := ParseSpec("my-plugin[search,web]")
	if spec.PackageName != "my-plugin" {
		t.Fatalf("expected packageName=my-plugin, got %s", spec.PackageName)
	}
	if len(spec.Features) != 2 || spec.Features[0] != "search" || spec.Features[1] != "web" {
		t.Fatalf("expected [search, web], got %v", spec.Features)
	}
}

func TestParseSpec_AllFeatures(t *testing.T) {
	spec := ParseSpec("my-plugin[*]")
	if spec.PackageName != "my-plugin" {
		t.Fatalf("expected packageName=my-plugin, got %s", spec.PackageName)
	}
	if spec.Features == nil || len(spec.Features) != 1 || spec.Features[0] != "*" {
		t.Fatalf("expected features=[*], got %v", spec.Features)
	}
}

func TestParseSpec_NoOptionalFeatures(t *testing.T) {
	spec := ParseSpec("my-plugin[]")
	if len(spec.Features) != 0 {
		t.Fatalf("expected empty features, got %v", spec.Features)
	}
}

func TestParseSpec_ScopedWithVersion(t *testing.T) {
	spec := ParseSpec("@scope/pkg@1.2.3[feat]")
	if spec.PackageName != "@scope/pkg@1.2.3" {
		t.Fatalf("expected @scope/pkg@1.2.3, got %s", spec.PackageName)
	}
	if len(spec.Features) != 1 || spec.Features[0] != "feat" {
		t.Fatalf("expected [feat], got %v", spec.Features)
	}
}

func TestFormatSpec(t *testing.T) {
	tests := []struct {
		spec ParsedSpec
		want string
	}{
		{ParsedSpec{PackageName: "pkg", Features: nil}, "pkg"},
		{ParsedSpec{PackageName: "pkg", Features: []string{"*"}}, "pkg[*]"},
		{ParsedSpec{PackageName: "pkg", Features: []string{"a", "b"}}, "pkg[a,b]"},
		{ParsedSpec{PackageName: "pkg", Features: []string{}}, "pkg[]"},
	}
	for _, tt := range tests {
		got := FormatSpec(tt.spec)
		if got != tt.want {
			t.Fatalf("FormatSpec(%+v) = %s, want %s", tt.spec, got, tt.want)
		}
	}
}

func TestExtractPackageName(t *testing.T) {
	tests := []struct {
		spec string
		want string
	}{
		{"lodash@4.17.21", "lodash"},
		{"@scope/pkg@1.0.0", "@scope/pkg"},
		{"@scope/pkg", "@scope/pkg"},
		{"my-plugin", "my-plugin"},
	}
	for _, tt := range tests {
		got := ExtractPackageName(tt.spec)
		if got != tt.want {
			t.Fatalf("Extract(%s) = %s, want %s", tt.spec, got, tt.want)
		}
	}
}

func TestManager_InitCreatesDirs(t *testing.T) {
	dir := t.TempDir()
	m, err := NewManager(ManagerOptions{RootDir: dir})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(dir, "plugins")); err != nil {
		t.Fatalf("plugins dir should exist: %v", err)
	}
	m.Close()
}

func TestManager_ListEmpty(t *testing.T) {
	dir := t.TempDir()
	m, err := NewManager(ManagerOptions{RootDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()

	plugins, err := m.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(plugins) != 0 {
		t.Fatalf("expected empty list, got %d plugins", len(plugins))
	}
}

func TestManager_EnableDisable(t *testing.T) {
	dir := t.TempDir()
	m, err := NewManager(ManagerOptions{RootDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()

	// Manually set up a plugin in runtime config
	m.runtime.Plugins["test-plugin"] = PluginState{
		Version:         "1.0.0",
		Enabled:         true,
		EnabledFeatures: nil,
	}
	m.saveRuntimeConfig()

	err = m.SetEnabled("test-plugin", false)
	if err != nil {
		t.Fatal(err)
	}

	if m.runtime.Plugins["test-plugin"].Enabled {
		t.Fatal("expected plugin to be disabled")
	}

	err = m.SetEnabled("test-plugin", true)
	if err != nil {
		t.Fatal(err)
	}

	if !m.runtime.Plugins["test-plugin"].Enabled {
		t.Fatal("expected plugin to be enabled")
	}
}

func TestManager_Features(t *testing.T) {
	dir := t.TempDir()
	m, err := NewManager(ManagerOptions{RootDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()

	m.runtime.Plugins["test-plugin"] = PluginState{
		Version:         "1.0.0",
		Enabled:         true,
		EnabledFeatures: nil,
	}
	m.saveRuntimeConfig()

	features, err := m.GetEnabledFeatures("test-plugin")
	if err != nil {
		t.Fatal(err)
	}
	if features != nil {
		t.Fatalf("expected nil features, got %v", features)
	}

	err = m.SetEnabledFeatures("test-plugin", []string{"search", "web"})
	if err != nil {
		t.Fatal(err)
	}

	features, _ = m.GetEnabledFeatures("test-plugin")
	if len(features) != 2 || features[0] != "search" {
		t.Fatalf("expected [search, web], got %v", features)
	}
}

func TestValidatePackageName(t *testing.T) {
	valid := []string{"lodash", "@scope/pkg", "my-plugin", "plugin@1.2.3", "@scope/pkg@1.0.0"}
	for _, name := range valid {
		if err := validatePackageName(name); err != nil {
			t.Fatalf("expected valid: %s, got error: %v", name, err)
		}
	}

	invalid := []string{"", "rm -rf /", "$(whoami)", "package;rm", "../escape"}
	for _, name := range invalid {
		if err := validatePackageName(name); err == nil {
			t.Fatalf("expected invalid: %s, but no error", name)
		}
	}
}
