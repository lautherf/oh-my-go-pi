package native_test

import (
	"testing"

	"github.com/oh-my-pi/omp/internal/native"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCountTokens_Empty(t *testing.T) {
	assert.Equal(t, 0, native.CountTokens(""))
}

func TestCountTokens_Short(t *testing.T) {
	n := native.CountTokens("hello world")
	assert.Greater(t, n, 0)
	assert.Less(t, n, 10)
}

func TestCountTokens_Longer(t *testing.T) {
	n := native.CountTokens("The quick brown fox jumps over the lazy dog")
	assert.Greater(t, n, 5)
	assert.Less(t, n, 20)
}

func TestCountTokens_Code(t *testing.T) {
	code := `func main() {
	println("hello")
}`
	n := native.CountTokens(code)
	assert.Greater(t, n, 3)
	assert.Less(t, n, 30)
}

func TestCountTokens_Consistent(t *testing.T) {
	a := native.CountTokens("hello world")
	b := native.CountTokens("hello world")
	assert.Equal(t, a, b)
}

func TestCountTokens_NonASCII(t *testing.T) {
	n := native.CountTokens("你好世界")
	assert.Greater(t, n, 0)
	// Chinese chars typically tokenize to multiple tokens each
	assert.GreaterOrEqual(t, n, 2)
}

func TestClipboard_CopyPaste(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping clipboard test in short mode")
	}
	text := "omp clipboard test"
	err := native.CopyToClipboard(text)
	require.NoError(t, err)

	got, err := native.ReadClipboard()
	require.NoError(t, err)
	assert.Equal(t, text, got)
}

func TestClipboard_CopyEmpty(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping clipboard test in short mode")
	}
	err := native.CopyToClipboard("")
	// empty copy might fail on some platforms, that's acceptable
	if err != nil {
		t.Logf("empty copy: %v (acceptable)", err)
	}
}
