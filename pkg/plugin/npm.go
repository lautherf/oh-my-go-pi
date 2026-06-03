package plugin

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type RegistryClient interface {
	FetchPackage(name string) (*NpmPackageMetadata, error)
	DownloadTarball(tarballURL, destDir string) error
}

type NpmPackageMetadata struct {
	Name     string                      `json:"name"`
	DistTags map[string]string           `json:"dist-tags"`
	Versions map[string]NpmVersionMeta   `json:"versions"`
}

type NpmVersionMeta struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Dist    NpmDistInfo `json:"dist"`
}

type NpmDistInfo struct {
	Tarball string `json:"tarball"`
}

type HTTPRegistryClient struct {
	BaseURL    string
	HTTPClient *http.Client
}

func NewDefaultRegistryClient() *HTTPRegistryClient {
	return &HTTPRegistryClient{
		BaseURL: "https://registry.npmjs.org",
		HTTPClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *HTTPRegistryClient) FetchPackage(name string) (*NpmPackageMetadata, error) {
	u := c.BaseURL + "/" + url.PathEscape(name)
	resp, err := c.HTTPClient.Get(u)
	if err != nil {
		return nil, fmt.Errorf("fetch %s: %w", name, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("fetch %s: HTTP %d: %s", name, resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var meta NpmPackageMetadata
	if err := json.NewDecoder(resp.Body).Decode(&meta); err != nil {
		return nil, fmt.Errorf("decode %s: %w", name, err)
	}
	return &meta, nil
}

func (c *HTTPRegistryClient) DownloadTarball(tarballURL, destDir string) error {
	resp, err := c.HTTPClient.Get(tarballURL)
	if err != nil {
		return fmt.Errorf("download tarball: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download tarball: HTTP %d", resp.StatusCode)
	}

	gr, err := gzip.NewReader(resp.Body)
	if err != nil {
		return fmt.Errorf("gzip: %w", err)
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("tar: %w", err)
		}

		// Strip the package/ prefix from npm tarballs
		name := stripTopDir(hdr.Name)
		if name == "" {
			continue
		}

		target := filepath.Join(destDir, name)
		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}
			f, err := os.Create(target)
			if err != nil {
				return err
			}
			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				return err
			}
			f.Close()
		}
	}
	return nil
}

func stripTopDir(name string) string {
	parts := strings.SplitN(name, "/", 2)
	if len(parts) < 2 {
		return ""
	}
	return parts[1]
}

func ResolveVersion(meta *NpmPackageMetadata, spec string) (*NpmVersionMeta, error) {
	if spec == "" || spec == "latest" {
		tag, ok := meta.DistTags["latest"]
		if !ok {
			return nil, fmt.Errorf("no latest tag for %s", meta.Name)
		}
		return resolveExact(meta, tag)
	}
	return resolveExact(meta, spec)
}

func resolveExact(meta *NpmPackageMetadata, version string) (*NpmVersionMeta, error) {
	v, ok := meta.Versions[version]
	if !ok {
		return nil, fmt.Errorf("version %s not found for %s", version, meta.Name)
	}
	return &v, nil
}
