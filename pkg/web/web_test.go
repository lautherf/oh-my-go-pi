package web_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/oh-my-pi/omp/pkg/web"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFetch_URL(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("<html><body><p>hello world</p></body></html>"))
	}))
	defer srv.Close()

	content, err := web.Fetch(context.Background(), srv.URL)
	require.NoError(t, err)
	assert.Contains(t, content, "hello world")
}

func TestFetch_NotFound(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
	}))
	defer srv.Close()

	_, err := web.Fetch(context.Background(), srv.URL+"/nonexistent")
	require.Error(t, err)
}

func TestFetch_ServerError(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	defer srv.Close()

	_, err := web.Fetch(context.Background(), srv.URL)
	require.Error(t, err)
}

func TestFetch_InvalidURL(t *testing.T) {
	t.Parallel()
	_, err := web.Fetch(context.Background(), "not-a-url")
	require.Error(t, err)
}

func TestFetch_EmptyURL(t *testing.T) {
	t.Parallel()
	_, err := web.Fetch(context.Background(), "")
	require.Error(t, err)
}

func TestFetch_NonHTML(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("plain text content"))
	}))
	defer srv.Close()

	content, err := web.Fetch(context.Background(), srv.URL)
	require.NoError(t, err)
	assert.Contains(t, content, "plain text content")
}

func TestSearch(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"results":[{"title":"Go Language","url":"https://go.dev","snippet":"An open source programming language"}]}`))
	}))
	defer srv.Close()

	results, err := web.Search(context.Background(), "golang", web.WithSearchURL(srv.URL))
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "Go Language", results[0].Title)
}

func TestSearch_EmptyQuery(t *testing.T) {
	t.Parallel()
	_, err := web.Search(context.Background(), "")
	require.Error(t, err)
}

func TestSearch_NoResults(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"results":[]}`))
	}))
	defer srv.Close()

	results, err := web.Search(context.Background(), "zzz_nonexistent", web.WithSearchURL(srv.URL))
	require.NoError(t, err)
	assert.Len(t, results, 0)
}
