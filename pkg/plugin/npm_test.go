package plugin

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestExtractVersion(t *testing.T) {
	tests := []struct {
		spec string
		want string
	}{
		{"lodash@4.17.21", "4.17.21"},
		{"lodash", ""},
		{"@scope/pkg@1.0.0", "1.0.0"},
		{"@scope/pkg", ""},
		{"my-plugin@latest", "latest"},
	}
	for _, tt := range tests {
		got := extractVersion(tt.spec)
		if got != tt.want {
			t.Fatalf("extractVersion(%q) = %q, want %q", tt.spec, got, tt.want)
		}
	}
}

func TestResolveVersion(t *testing.T) {
	meta := &NpmPackageMetadata{
		Name: "test-pkg",
		DistTags: map[string]string{"latest": "2.0.0"},
		Versions: map[string]NpmVersionMeta{
			"1.0.0": {Name: "test-pkg", Version: "1.0.0", Dist: NpmDistInfo{Tarball: "http://example.com/1.0.0.tgz"}},
			"2.0.0": {Name: "test-pkg", Version: "2.0.0", Dist: NpmDistInfo{Tarball: "http://example.com/2.0.0.tgz"}},
		},
	}

	t.Run("latest", func(t *testing.T) {
		v, err := ResolveVersion(meta, "")
		if err != nil {
			t.Fatal(err)
		}
		if v.Version != "2.0.0" {
			t.Fatalf("expected 2.0.0, got %s", v.Version)
		}
	})

	t.Run("exact", func(t *testing.T) {
		v, err := ResolveVersion(meta, "1.0.0")
		if err != nil {
			t.Fatal(err)
		}
		if v.Version != "1.0.0" {
			t.Fatalf("expected 1.0.0, got %s", v.Version)
		}
	})

	t.Run("not found", func(t *testing.T) {
		_, err := ResolveVersion(meta, "3.0.0")
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestHTTPRegistryClient_FetchPackage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/test-pkg" {
			meta := NpmPackageMetadata{
				Name:     "test-pkg",
				DistTags: map[string]string{"latest": "1.0.0"},
				Versions: map[string]NpmVersionMeta{
					"1.0.0": {Name: "test-pkg", Version: "1.0.0", Dist: NpmDistInfo{Tarball: serverURL(r) + "/test-pkg/-/test-pkg-1.0.0.tgz"}},
				},
			}
			json.NewEncoder(w).Encode(meta)
		} else if r.URL.Path == "/not-found" {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("Not Found"))
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer server.Close()

	client := &HTTPRegistryClient{BaseURL: server.URL, HTTPClient: server.Client()}

	t.Run("success", func(t *testing.T) {
		meta, err := client.FetchPackage("test-pkg")
		if err != nil {
			t.Fatal(err)
		}
		if meta.Name != "test-pkg" {
			t.Fatalf("expected test-pkg, got %s", meta.Name)
		}
	})

	t.Run("not found", func(t *testing.T) {
		_, err := client.FetchPackage("not-found")
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func serverURL(r *http.Request) string {
	return fmt.Sprintf("http://%s", r.Host)
}

func makeTestTarball(pkgName string, files map[string]string) []byte {
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	// npm tarballs have a top-level dir like "package/"
	for path, content := range files {
		hdr := &tar.Header{
			Name:     "package/" + path,
			Size:     int64(len(content)),
			Typeflag: tar.TypeReg,
			Mode:     0644,
		}
		tw.WriteHeader(hdr)
		tw.Write([]byte(content))
	}

	tw.Close()
	gw.Close()
	return buf.Bytes()
}

func TestHTTPRegistryClient_DownloadTarball(t *testing.T) {
	pkgJSON := `{"name":"test-pkg","version":"1.0.0","omp":{"name":"test-pkg","version":"1.0.0","description":"Test","hooks":"main.js"}}`
	tarball := makeTestTarball("test-pkg", map[string]string{
		"package.json": pkgJSON,
		"main.js":      "module.exports = {};",
	})
	tarballStr := string(tarball)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(tarballStr))
	}))
	defer server.Close()

	client := &HTTPRegistryClient{HTTPClient: server.Client()}

	destDir := t.TempDir()
	err := client.DownloadTarball(server.URL, destDir)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(destDir, "package.json")); err != nil {
		t.Fatal("expected package.json in destDir")
	}
	if _, err := os.Stat(filepath.Join(destDir, "main.js")); err != nil {
		t.Fatal("expected main.js in destDir")
	}
}

func TestManager_InstallWithMockRegistry(t *testing.T) {
	pkgJSON := `{"name":"@test/my-plugin","version":"1.2.3","omp":{"name":"my-plugin","version":"1.2.3","description":"A test plugin","hooks":"index.js"}}`
	tarball := makeTestTarball("package", map[string]string{
		"package.json": pkgJSON,
		"index.js":     "module.exports = {activate: () => {}};",
	})

	var tarballURL string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/@test/my-plugin":
			meta := NpmPackageMetadata{
				Name:     "@test/my-plugin",
				DistTags: map[string]string{"latest": "1.2.3"},
				Versions: map[string]NpmVersionMeta{
					"1.2.3": {Name: "@test/my-plugin", Version: "1.2.3", Dist: NpmDistInfo{Tarball: tarballURL}},
				},
			}
			json.NewEncoder(w).Encode(meta)
		case r.URL.Path == "/tarball":
			w.Write(tarball)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	tarballURL = server.URL + "/tarball"
	defer server.Close()

	reg := &HTTPRegistryClient{BaseURL: server.URL, HTTPClient: server.Client()}
	dir := t.TempDir()
	m, err := NewManager(ManagerOptions{RootDir: dir, Registry: reg})
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()

	p, err := m.Install("@test/my-plugin", InstallOptions{})
	if err != nil {
		t.Fatal(err)
	}

	if p.Name != "@test/my-plugin" {
		t.Fatalf("expected @test/my-plugin, got %s", p.Name)
	}
	if p.Version != "1.2.3" {
		t.Fatalf("expected 1.2.3, got %s", p.Version)
	}
	if !p.Enabled {
		t.Fatal("expected plugin to be enabled")
	}

	// Should appear in list
	plugins, err := m.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(plugins) != 1 {
		t.Fatalf("expected 1 plugin, got %d", len(plugins))
	}
	if plugins[0].Name != "@test/my-plugin" {
		t.Fatalf("expected @test/my-plugin, got %s", plugins[0].Name)
	}
}

func TestManager_InstallDryRun(t *testing.T) {
	dir := t.TempDir()
	m, err := NewManager(ManagerOptions{RootDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()

	p, err := m.Install("dry-run-plugin", InstallOptions{DryRun: true})
	if err != nil {
		t.Fatal(err)
	}
	if p.Name != "dry-run-plugin" {
		t.Fatalf("expected dry-run-plugin, got %s", p.Name)
	}
}

func TestManager_InstallInvalidName(t *testing.T) {
	dir := t.TempDir()
	m, err := NewManager(ManagerOptions{RootDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()

	_, err = m.Install("rm -rf /", InstallOptions{})
	if err == nil {
		t.Fatal("expected error for invalid name")
	}
}

func TestManager_UninstallWithRemoval(t *testing.T) {
	dir := t.TempDir()

	// Create a fake plugin directory
	pluginDir := filepath.Join(dir, "plugins", "test-pkg")
	if err := os.MkdirAll(pluginDir, 0755); err != nil {
		t.Fatal(err)
	}
	pkgJSON := `{"name":"test-pkg","version":"1.0.0","omp":{"name":"test-pkg","version":"1.0.0"}}`
	if err := os.WriteFile(filepath.Join(pluginDir, "package.json"), []byte(pkgJSON), 0644); err != nil {
		t.Fatal(err)
	}

	m, err := NewManager(ManagerOptions{RootDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()

	// Add to runtime config
	m.runtime.Plugins["test-pkg"] = PluginState{Version: "1.0.0", Enabled: true}
	m.saveRuntimeConfig()

	if err := m.Uninstall("test-pkg"); err != nil {
		t.Fatal(err)
	}

	// Directory should be removed
	if _, err := os.Stat(pluginDir); !os.IsNotExist(err) {
		t.Fatal("expected plugin directory to be removed")
	}

	// Should not appear in list
	plugins, err := m.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(plugins) != 0 {
		t.Fatalf("expected 0 plugins, got %d", len(plugins))
	}
}

func TestManager_ListWithPlugins(t *testing.T) {
	dir := t.TempDir()

	// Create two plugins
	plugins := []struct {
		name    string
		version string
	}{
		{"plugin-a", "1.0.0"},
		{"plugin-b", "2.0.0"},
	}
	for _, p := range plugins {
		pDir := filepath.Join(dir, "plugins", p.name)
		os.MkdirAll(pDir, 0755)
		pkgJSON := fmt.Sprintf(`{"name":"%s","version":"%s","omp":{"name":"%s","version":"%s","description":"Test"}}`, p.name, p.version, p.name, p.version)
		os.WriteFile(filepath.Join(pDir, "package.json"), []byte(pkgJSON), 0644)
	}

	m, err := NewManager(ManagerOptions{RootDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()

	m.runtime.Plugins["plugin-a"] = PluginState{Version: "1.0.0", Enabled: true}
	m.runtime.Plugins["plugin-b"] = PluginState{Version: "2.0.0", Enabled: false}
	m.saveRuntimeConfig()

	list, err := m.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 plugins, got %d", len(list))
	}
}

func TestManager_ListSkipsNonPluginDirs(t *testing.T) {
	dir := t.TempDir()

	// Create something that looks like a dir but is not a plugin
	os.MkdirAll(filepath.Join(dir, "plugins", ".hidden"), 0755)
	os.MkdirAll(filepath.Join(dir, "plugins", "_private"), 0755)

	// Legit plugin with proper package.json
	pDir2 := filepath.Join(dir, "plugins", "real-plugin")
	os.MkdirAll(pDir2, 0755)
	os.WriteFile(filepath.Join(pDir2, "package.json"), []byte(`{"name":"real-plugin","version":"1.0.0","omp":{"name":"real-plugin","version":"1.0.0"}}`), 0644)

	// Dir with package.json but no omp/pi field
	pDir3 := filepath.Join(dir, "plugins", "non-omp")
	os.MkdirAll(pDir3, 0755)
	os.WriteFile(filepath.Join(pDir3, "package.json"), []byte(`{"name":"non-omp","version":"1.0.0"}`), 0644)

	m, err := NewManager(ManagerOptions{RootDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()

	list, err := m.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].Name != "real-plugin" {
		t.Fatalf("expected 1 plugin (real-plugin), got %d: %+v", len(list), list)
	}
}

func TestStripTopDir(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{"package/", ""},
		{"package/file.txt", "file.txt"},
		{"package/dir/file.txt", "dir/file.txt"},
		{"other/file.txt", "file.txt"},
		{"no-slash", ""},
	}
	for _, tt := range tests {
		got := stripTopDir(tt.name)
		if got != tt.want {
			t.Fatalf("stripTopDir(%q) = %q, want %q", tt.name, got, tt.want)
		}
	}
}
