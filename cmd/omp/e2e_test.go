package main

import (
	"os/exec"
	"strings"
	"testing"
)

func buildBinary(t *testing.T) string {
	t.Helper()
	binary := t.TempDir() + "/omp"
	cmd := exec.Command("go", "build", "-o", binary, ".")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("build failed: %v\n%s", err, out)
	}
	return binary
}

func runBinary(t *testing.T, binary string, args ...string) (string, error) {
	t.Helper()
	cmd := exec.Command(binary, args...)
	cmd.Stdin = nil
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func TestCLI_Help(t *testing.T) {
	binary := buildBinary(t)
	out, err := runBinary(t, binary, "--help")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "omp") {
		t.Fatal("expected help to contain 'omp'")
	}
}

func TestCLI_Version(t *testing.T) {
	binary := buildBinary(t)
	out, err := runBinary(t, binary, "--version")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "0.0.1") {
		t.Fatal("expected version 0.0.1")
	}
}

func TestCLI_PluginHelp(t *testing.T) {
	binary := buildBinary(t)
	out, err := runBinary(t, binary, "plugin", "--help")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "install") && !strings.Contains(out, "list") {
		t.Fatal("expected plugin help to mention install/list")
	}
}

func TestCLI_PluginList(t *testing.T) {
	binary := buildBinary(t)
	out, err := runBinary(t, binary, "plugin", "list")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "No plugins installed") {
		t.Fatalf("expected 'No plugins installed', got: %s", out)
	}
}

func TestCLI_StatsHelp(t *testing.T) {
	binary := buildBinary(t)
	out, err := runBinary(t, binary, "stats", "--help")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "serve") {
		t.Fatal("expected stats help to mention serve")
	}
}

func TestCLI_UnknownCommand(t *testing.T) {
	binary := buildBinary(t)
	_, err := runBinary(t, binary, "unknown-command")
	if err == nil {
		t.Fatal("expected error for unknown command")
	}
}

func TestCLI_PprofFlag(t *testing.T) {
	binary := buildBinary(t)
	out, err := runBinary(t, binary, "--pprof", ":0", "--help")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "omp") {
		t.Fatal("expected help output")
	}
}

func TestCLI_InteractiveFailsWithoutOllama(t *testing.T) {
	binary := buildBinary(t)
	_, err := runBinary(t, binary)
	// Should fail because no Ollama is running
	if err == nil {
		t.Fatal("expected error when running without Ollama")
	}
}
