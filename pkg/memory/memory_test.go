package memory_test

import (
	"context"
	"testing"

	"github.com/oh-my-pi/omp/pkg/memory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewStore(t *testing.T) {
	s, err := memory.New(":memory:")
	require.NoError(t, err)
	defer s.Close()
}

func TestStore_Remember(t *testing.T) {
	s, err := memory.New(":memory:")
	require.NoError(t, err)
	defer s.Close()

	err = s.Remember(context.Background(), "the user likes Go programming")
	require.NoError(t, err)
}

func TestStore_Recall(t *testing.T) {
	s, err := memory.New(":memory:")
	require.NoError(t, err)
	defer s.Close()

	require.NoError(t, s.Remember(context.Background(), "the user likes Go programming"))
	require.NoError(t, s.Remember(context.Background(), "the project uses TDD"))
	require.NoError(t, s.Remember(context.Background(), "the answer is 42"))

	results, err := s.Recall(context.Background(), "Go programming")
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(results), 1)
	assert.Contains(t, results[0].Content, "Go")
}

func TestStore_RecallNoMatch(t *testing.T) {
	s, err := memory.New(":memory:")
	require.NoError(t, err)
	defer s.Close()

	require.NoError(t, s.Remember(context.Background(), "only this exists"))

	results, err := s.Recall(context.Background(), "zzz_nonexistent")
	require.NoError(t, err)
	assert.Len(t, results, 0)
}

func TestStore_RecallWithLimit(t *testing.T) {
	s, err := memory.New(":memory:")
	require.NoError(t, err)
	defer s.Close()

	for i := 0; i < 10; i++ {
		mem := "test memory number"
		if i%2 == 0 {
			mem += " important"
		}
		require.NoError(t, s.Remember(context.Background(), mem))
	}

	results, err := s.Recall(context.Background(), "important")
	require.NoError(t, err)
	assert.LessOrEqual(t, len(results), 5)
}

func TestStore_MultipleSessions(t *testing.T) {
	s, err := memory.New(":memory:")
	require.NoError(t, err)
	defer s.Close()

	require.NoError(t, s.Remember(context.Background(), "session1 data"))
	require.NoError(t, s.Remember(context.Background(), "session2 data"))

	results, err := s.Recall(context.Background(), "session")
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(results), 2)
}

func TestStore_Close(t *testing.T) {
	s, err := memory.New(":memory:")
	require.NoError(t, err)
	assert.NoError(t, s.Close())
}

func TestStore_RememberAfterClose(t *testing.T) {
	s, err := memory.New(":memory:")
	require.NoError(t, err)
	s.Close()

	err = s.Remember(context.Background(), "should fail")
	require.Error(t, err)
}
