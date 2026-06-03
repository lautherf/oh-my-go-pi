package ai_test

import (
	"testing"

	"github.com/oh-my-pi/omp/pkg/ai"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegistry_RegisterAndGet(t *testing.T) {
	r := ai.NewRegistry()
	p := ai.NewOllamaProvider("test", "")

	r.Register("test-provider", p)
	got, err := r.Get("test-provider")
	require.NoError(t, err)
	assert.Equal(t, "ollama", got.Name())
}

func TestRegistry_GetNotFound(t *testing.T) {
	r := ai.NewRegistry()
	_, err := r.Get("nonexistent")
	require.Error(t, err)
}

func TestRegistry_List(t *testing.T) {
	r := ai.NewRegistry()
	r.Register("a", ai.NewOllamaProvider("a", ""))
	r.Register("b", ai.NewOllamaProvider("b", ""))

	names := r.List()
	assert.ElementsMatch(t, []string{"a", "b"}, names)
}

func TestRegistry_RegisterDuplicate(t *testing.T) {
	r := ai.NewRegistry()
	r.Register("x", ai.NewOllamaProvider("x", ""))
	// should not panic; last one wins or silently ignored
	r.Register("x", ai.NewOllamaProvider("x", ""))
	assert.Len(t, r.List(), 1)
}

func TestRegistry_AutoDiscover(t *testing.T) {
	r := ai.NewRegistry()
	// Ollama auto-discovery: detect localhost:11434
	count := r.AutoDiscover()
	t.Logf("auto-discovered %d providers", count)
	// In test environment, no Ollama is expected
	// but the function should not error
	assert.GreaterOrEqual(t, count, 0)
}
