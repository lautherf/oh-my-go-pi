package plugin

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// ParsedSpec represents a parsed plugin install specifier.
type ParsedSpec struct {
	PackageName string
	Features    []string // nil = use defaults, [*] = all, [] = none
}

// PluginManifest is the omp/pi field from package.json.
type PluginManifest struct {
	Name        string                    `json:"name,omitempty"`
	Version     string                    `json:"version"`
	Description string                    `json:"description,omitempty"`
	Tools       string                    `json:"tools,omitempty"`
	Hooks       string                    `json:"hooks,omitempty"`
	Extensions  []string                  `json:"extensions,omitempty"`
	Commands    []string                  `json:"commands,omitempty"`
	Features    map[string]PluginFeature  `json:"features,omitempty"`
}

type PluginFeature struct {
	Description string   `json:"description,omitempty"`
	Default     bool     `json:"default,omitempty"`
	Extensions  []string `json:"extensions,omitempty"`
	Tools       []string `json:"tools,omitempty"`
	Hooks       []string `json:"hooks,omitempty"`
	Commands    []string `json:"commands,omitempty"`
}

// InstalledPlugin represents a resolved, installed plugin.
type InstalledPlugin struct {
	Name            string
	Version         string
	Path            string
	Manifest        PluginManifest
	EnabledFeatures []string // nil = use defaults
	Enabled         bool
}

// PluginState is the runtime state per plugin in the lock file.
type PluginState struct {
	Version         string   `json:"version"`
	EnabledFeatures []string `json:"enabledFeatures"` // nil = defaults
	Enabled         bool     `json:"enabled"`
}

// RuntimeConfig is the persisted lock file content.
type RuntimeConfig struct {
	Plugins  map[string]PluginState          `json:"plugins"`
	Settings map[string]map[string]any       `json:"settings"`
}

// Manager manages plugin lifecycle.
type Manager struct {
	rootDir      string
	pluginsDir   string
	runtime      RuntimeConfig
	registry     RegistryClient
}

type ManagerOptions struct {
	RootDir  string // where plugins/ lives (default: ~/.config/omp)
	Registry RegistryClient // nil = use default HTTP registry
}

func NewManager(opts ManagerOptions) (*Manager, error) {
	rootDir := opts.RootDir
	if rootDir == "" {
		home, err := os.UserConfigDir()
		if err != nil {
			return nil, fmt.Errorf("plugin manager: %w", err)
		}
		rootDir = filepath.Join(home, "omp")
	}

	reg := opts.Registry
	if reg == nil {
		reg = NewDefaultRegistryClient()
	}

	m := &Manager{
		rootDir:    rootDir,
		pluginsDir: filepath.Join(rootDir, "plugins"),
		registry:   reg,
		runtime: RuntimeConfig{
			Plugins:  make(map[string]PluginState),
			Settings: make(map[string]map[string]any),
		},
	}

	if err := os.MkdirAll(m.pluginsDir, 0755); err != nil {
		return nil, err
	}
	m.loadRuntimeConfig()
	return m, nil
}

func (m *Manager) Close() {
	m.saveRuntimeConfig()
}

func (m *Manager) loadRuntimeConfig() {
	path := filepath.Join(m.pluginsDir, "omp-plugins.lock.json")
	data, err := os.ReadFile(path)
	if err != nil {
		m.runtime = RuntimeConfig{
			Plugins:  make(map[string]PluginState),
			Settings: make(map[string]map[string]any),
		}
		return
	}
	var cfg RuntimeConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		m.runtime = RuntimeConfig{
			Plugins:  make(map[string]PluginState),
			Settings: make(map[string]map[string]any),
		}
		return
	}
	if cfg.Plugins == nil {
		cfg.Plugins = make(map[string]PluginState)
	}
	if cfg.Settings == nil {
		cfg.Settings = make(map[string]map[string]any)
	}
	m.runtime = cfg
}

func (m *Manager) saveRuntimeConfig() {
	path := filepath.Join(m.pluginsDir, "omp-plugins.lock.json")
	data, _ := json.MarshalIndent(m.runtime, "", "  ")
	os.WriteFile(path, data, 0644)
}

type npmPackageJSON struct {
	Name    string          `json:"name"`
	Version string          `json:"version"`
	OMP     *PluginManifest `json:"omp"`
	PI      *PluginManifest `json:"pi"`
}

func (m *Manager) readPluginPackageJSON(name string) (*npmPackageJSON, error) {
	pkgPath := filepath.Join(m.pluginsDir, name, "package.json")
	data, err := os.ReadFile(pkgPath)
	if err != nil {
		return nil, err
	}
	var pkg npmPackageJSON
	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil, err
	}
	return &pkg, nil
}

func (m *Manager) Install(specString string, opts InstallOptions) (*InstalledPlugin, error) {
	spec := ParseSpec(specString)
	if err := validatePackageName(spec.PackageName); err != nil {
		return nil, err
	}

	if opts.DryRun {
		return &InstalledPlugin{
			Name:    spec.PackageName,
			Version: "0.0.0-dryrun",
			Manifest: PluginManifest{Version: "0.0.0-dryrun"},
			Enabled: true,
		}, nil
	}

	actualName := ExtractPackageName(spec.PackageName)

	meta, err := m.registry.FetchPackage(actualName)
	if err != nil {
		return nil, fmt.Errorf("fetch package %s: %w", actualName, err)
	}

	versionStr := extractVersion(spec.PackageName)
	versionMeta, err := ResolveVersion(meta, versionStr)
	if err != nil {
		return nil, fmt.Errorf("resolve version for %s: %w", actualName, err)
	}

	pluginDir := filepath.Join(m.pluginsDir, actualName)
	if err := os.RemoveAll(pluginDir); err != nil {
		return nil, fmt.Errorf("clean plugin dir: %w", err)
	}

	if err := m.registry.DownloadTarball(versionMeta.Dist.Tarball, pluginDir); err != nil {
		os.RemoveAll(pluginDir)
		return nil, fmt.Errorf("download %s: %w", actualName, err)
	}

	pkg, err := m.readPluginPackageJSON(actualName)
	if err != nil {
		return nil, fmt.Errorf("read package.json after install: %w", err)
	}

	manifest := pkg.OMP
	if manifest == nil {
		manifest = pkg.PI
	}
	if manifest == nil {
		manifest = &PluginManifest{Version: pkg.Version}
	}
	manifest.Version = pkg.Version

	var enabledFeatures []string
	if spec.Features != nil {
		if len(spec.Features) == 1 && spec.Features[0] == "*" {
			enabledFeatures = nil
		} else {
			enabledFeatures = spec.Features
		}
	}

	state := m.runtime.Plugins[pkg.Name]
	state.Version = pkg.Version
	state.Enabled = true
	if enabledFeatures != nil {
		state.EnabledFeatures = enabledFeatures
	}
	m.runtime.Plugins[pkg.Name] = state
	m.saveRuntimeConfig()

	return &InstalledPlugin{
		Name:    pkg.Name,
		Version: pkg.Version,
		Path:    pluginDir,
		Manifest: *manifest,
		EnabledFeatures: state.EnabledFeatures,
		Enabled: true,
	}, nil
}

func (m *Manager) Uninstall(name string) error {
	if err := validatePackageName(name); err != nil {
		return err
	}

	pluginDir := filepath.Join(m.pluginsDir, name)
	if err := os.RemoveAll(pluginDir); err != nil {
		return fmt.Errorf("remove %s: %w", name, err)
	}

	delete(m.runtime.Plugins, name)
	delete(m.runtime.Settings, name)
	m.saveRuntimeConfig()
	return nil
}

func (m *Manager) List() ([]InstalledPlugin, error) {
	var plugins []InstalledPlugin
	err := m.scanPlugins(m.pluginsDir, &plugins)
	if err != nil {
		return nil, nil
	}
	return plugins, nil
}

func (m *Manager) scanPlugins(dir string, plugins *[]InstalledPlugin) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_") {
			continue
		}

		relPath, _ := filepath.Rel(m.pluginsDir, filepath.Join(dir, name))

		pkg, err := m.readPluginPackageJSON(relPath)
		if err != nil {
			if strings.HasPrefix(name, "@") {
				m.scanPlugins(filepath.Join(dir, name), plugins)
			}
			continue
		}

		manifest := pkg.OMP
		if manifest == nil {
			manifest = pkg.PI
		}
		if manifest == nil {
			if strings.HasPrefix(name, "@") {
				m.scanPlugins(filepath.Join(dir, name), plugins)
			}
			continue
		}
		manifest.Version = pkg.Version

		state := m.runtime.Plugins[relPath]

		*plugins = append(*plugins, InstalledPlugin{
			Name:    relPath,
			Version: pkg.Version,
			Path:    filepath.Join(m.pluginsDir, relPath),
			Manifest: *manifest,
			EnabledFeatures: state.EnabledFeatures,
			Enabled: state.Enabled,
		})
	}
	return nil
}

func extractVersion(spec string) string {
	if strings.HasPrefix(spec, "@") {
		// scoped package: @scope/name@version
		parts := strings.SplitN(spec, "@", 3)
		if len(parts) == 3 && parts[2] != "" {
			return parts[2]
		}
		return ""
	}
	idx := strings.LastIndex(spec, "@")
	if idx > 0 {
		return spec[idx+1:]
	}
	return ""
}

func (m *Manager) GetEnabledFeatures(name string) ([]string, error) {
	state, ok := m.runtime.Plugins[name]
	if !ok {
		return nil, fmt.Errorf("plugin %s not found in runtime config", name)
	}
	return state.EnabledFeatures, nil
}

func (m *Manager) SetEnabledFeatures(name string, features []string) error {
	state, ok := m.runtime.Plugins[name]
	if !ok {
		return fmt.Errorf("plugin %s not found in runtime config", name)
	}
	state.EnabledFeatures = features
	m.runtime.Plugins[name] = state
	m.saveRuntimeConfig()
	return nil
}

func (m *Manager) SetEnabled(name string, enabled bool) error {
	state, ok := m.runtime.Plugins[name]
	if !ok {
		return fmt.Errorf("plugin %s not found in runtime config", name)
	}
	state.Enabled = enabled
	m.runtime.Plugins[name] = state
	m.saveRuntimeConfig()
	return nil
}

func (m *Manager) GetSetting(name, key string) (any, bool) {
	if m.runtime.Settings[name] == nil {
		return nil, false
	}
	v, ok := m.runtime.Settings[name][key]
	return v, ok
}

func (m *Manager) SetSetting(name, key string, value any) {
	if m.runtime.Settings[name] == nil {
		m.runtime.Settings[name] = make(map[string]any)
	}
	m.runtime.Settings[name][key] = value
	m.saveRuntimeConfig()
}

func (m *Manager) DeleteSetting(name, key string) {
	if m.runtime.Settings[name] != nil {
		delete(m.runtime.Settings[name], key)
		m.saveRuntimeConfig()
	}
}

type InstallOptions struct {
	Force  bool
	DryRun bool
}

var (
	validPackageRE = regexp.MustCompile(`^(@[a-z0-9-~][a-z0-9-._~]*/)?[a-z0-9-~][a-z0-9-._~]*(@[a-z0-9-._^~>=<]+)?$`)
	shellMetachars = regexp.MustCompile(`[;&|` + "`" + `$(){}[\]<>\\]`)
)

func validatePackageName(name string) error {
	base := ExtractPackageName(name)
	if !validPackageRE.MatchString(base) {
		return fmt.Errorf("invalid package name: %s", name)
	}
	if shellMetachars.MatchString(name) {
		return fmt.Errorf("invalid characters in package name: %s", name)
	}
	return nil
}

func ParseSpec(spec string) ParsedSpec {
	bracketStart := strings.LastIndex(spec, "[")
	bracketEnd := strings.LastIndex(spec, "]")

	if bracketStart == -1 || bracketEnd == -1 || bracketEnd < bracketStart {
		return ParsedSpec{PackageName: spec, Features: nil}
	}

	pkgName := spec[:bracketStart]
	featStr := strings.TrimSpace(spec[bracketStart+1 : bracketEnd])

	if featStr == "*" {
		return ParsedSpec{PackageName: pkgName, Features: []string{"*"}}
	}
	if featStr == "" {
		return ParsedSpec{PackageName: pkgName, Features: []string{}}
	}

	parts := strings.Split(featStr, ",")
	var features []string
	for _, p := range parts {
		if f := strings.TrimSpace(p); f != "" {
			features = append(features, f)
		}
	}
	return ParsedSpec{PackageName: pkgName, Features: features}
}

func FormatSpec(spec ParsedSpec) string {
	if spec.Features == nil {
		return spec.PackageName
	}
	if len(spec.Features) == 1 && spec.Features[0] == "*" {
		return spec.PackageName + "[*]"
	}
	if len(spec.Features) == 0 {
		return spec.PackageName + "[]"
	}
	return spec.PackageName + "[" + strings.Join(spec.Features, ",") + "]"
}

var scopePkgRE = regexp.MustCompile(`^(@[^/]+/[^@]+)`)

func ExtractPackageName(specifier string) string {
	if strings.HasPrefix(specifier, "@") {
		m := scopePkgRE.FindStringSubmatch(specifier)
		if m != nil {
			return m[1]
		}
		return specifier
	}
	idx := strings.LastIndex(specifier, "@")
	if idx > 0 {
		return specifier[:idx]
	}
	return specifier
}
