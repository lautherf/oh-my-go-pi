package stats

import (
	"testing"
)

func TestOpenCloseDB(t *testing.T) {
	db, err := OpenDB(t.TempDir() + "/stats.db")
	if err != nil {
		t.Fatal(err)
	}
	if db == nil {
		t.Fatal("expected non-nil db")
	}
	db.Close()
}

func TestInsertAndQueryMessages(t *testing.T) {
	db := mustOpenDB(t)
	defer db.Close()

	stats := []MessageStats{
		{
			SessionFile: "/tmp/session.jsonl", EntryID: "1", Folder: "/project",
			Model: "gpt-4", Provider: "openai", API: "chat",
			Timestamp: 1000, StopReason: "stop",
			InputTokens: 10, OutputTokens: 20, TotalTokens: 30,
		},
		{
			SessionFile: "/tmp/session.jsonl", EntryID: "2", Folder: "/project",
			Model: "gpt-4", Provider: "openai", API: "chat",
			Timestamp: 2000, StopReason: "error", ErrorMessage: "timeout",
			InputTokens: 5, OutputTokens: 0, TotalTokens: 5,
		},
	}

	n, err := db.InsertMessages(stats)
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Fatalf("expected 2 inserted, got %d", n)
	}

	overall := db.OverallStats(0)
	if overall.TotalRequests != 2 {
		t.Fatalf("expected 2 total requests, got %d", overall.TotalRequests)
	}
	if overall.FailedRequests != 1 {
		t.Fatalf("expected 1 failed, got %d", overall.FailedRequests)
	}
}

func TestOverallStatsWithCutoff(t *testing.T) {
	db := mustOpenDB(t)
	defer db.Close()

	db.InsertMessages([]MessageStats{
		{SessionFile: "a.jsonl", EntryID: "1", Folder: "/p", Model: "m1", Provider: "p1", API: "c", Timestamp: 100, StopReason: "stop", InputTokens: 1, OutputTokens: 1, TotalTokens: 2},
		{SessionFile: "a.jsonl", EntryID: "2", Folder: "/p", Model: "m1", Provider: "p1", API: "c", Timestamp: 200, StopReason: "stop", InputTokens: 1, OutputTokens: 1, TotalTokens: 2},
		{SessionFile: "a.jsonl", EntryID: "3", Folder: "/p", Model: "m1", Provider: "p1", API: "c", Timestamp: 300, StopReason: "stop", InputTokens: 1, OutputTokens: 1, TotalTokens: 2},
	})

	overall := db.OverallStats(250)
	if overall.TotalRequests != 1 {
		t.Fatalf("expected 1 request after cutoff 250, got %d", overall.TotalRequests)
	}
}

func TestStatsByModel(t *testing.T) {
	db := mustOpenDB(t)
	defer db.Close()

	db.InsertMessages([]MessageStats{
		{SessionFile: "a.jsonl", EntryID: "1", Folder: "/p", Model: "gpt-4", Provider: "openai", API: "c", Timestamp: 100, StopReason: "stop", InputTokens: 10, OutputTokens: 20, TotalTokens: 30},
		{SessionFile: "a.jsonl", EntryID: "2", Folder: "/p", Model: "claude-3", Provider: "anthropic", API: "c", Timestamp: 200, StopReason: "stop", InputTokens: 5, OutputTokens: 15, TotalTokens: 20},
		{SessionFile: "a.jsonl", EntryID: "3", Folder: "/p", Model: "gpt-4", Provider: "openai", API: "c", Timestamp: 300, StopReason: "stop", InputTokens: 20, OutputTokens: 30, TotalTokens: 50},
	})

	byModel := db.StatsByModel(0)
	if len(byModel) != 2 {
		t.Fatalf("expected 2 model entries, got %d", len(byModel))
	}
	if byModel[0].Model != "gpt-4" || byModel[0].TotalRequests != 2 {
		t.Fatalf("expected gpt-4 with 2 requests, got %s with %d", byModel[0].Model, byModel[0].TotalRequests)
	}
}

func TestTimeSeries(t *testing.T) {
	db := mustOpenDB(t)
	defer db.Close()

	db.InsertMessages([]MessageStats{
		{SessionFile: "a.jsonl", EntryID: "1", Folder: "/p", Model: "m", Provider: "p", API: "c", Timestamp: 3600000, StopReason: "stop", InputTokens: 1, OutputTokens: 1, TotalTokens: 2},
		{SessionFile: "a.jsonl", EntryID: "2", Folder: "/p", Model: "m", Provider: "p", API: "c", Timestamp: 7200000, StopReason: "stop", InputTokens: 1, OutputTokens: 1, TotalTokens: 2},
	})

	series := db.TimeSeries(0, nil, 3600000)
	if len(series) != 2 {
		t.Fatalf("expected 2 time series points, got %d", len(series))
	}
}

func TestFileOffsetRoundTrip(t *testing.T) {
	db := mustOpenDB(t)
	defer db.Close()

	offset, lmod, err := db.GetFileOffset("/tmp/s.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	if offset != 0 || lmod != 0 {
		t.Fatalf("expected zero offset for missing file")
	}

	err = db.SetFileOffset("/tmp/s.jsonl", 123, 456)
	if err != nil {
		t.Fatal(err)
	}

	offset, lmod, err = db.GetFileOffset("/tmp/s.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	if offset != 123 || lmod != 456 {
		t.Fatalf("expected offset=123, lmod=456, got offset=%d, lmod=%d", offset, lmod)
	}
}

func TestRecentRequests(t *testing.T) {
	db := mustOpenDB(t)
	defer db.Close()

	for i := 0; i < 5; i++ {
		db.InsertMessages([]MessageStats{
			{SessionFile: "a.jsonl", EntryID: itoa(i), Folder: "/p", Model: "m", Provider: "p", API: "c", Timestamp: int64(1000 + i*1000), StopReason: "stop", InputTokens: 1, OutputTokens: 1, TotalTokens: 2},
		})
	}

	recent := db.RecentRequests(3)
	if len(recent) != 3 {
		t.Fatalf("expected 3 recent requests, got %d", len(recent))
	}
	// most recent first
	if recent[0].Timestamp != 5000 {
		t.Fatalf("expected most recent timestamp 5000, got %d", recent[0].Timestamp)
	}
}

func TestMessageCount(t *testing.T) {
	db := mustOpenDB(t)
	defer db.Close()

	db.InsertMessages([]MessageStats{
		{SessionFile: "a.jsonl", EntryID: "1", Folder: "/p", Model: "m", Provider: "p", API: "c", Timestamp: 100, StopReason: "stop", InputTokens: 1, OutputTokens: 1, TotalTokens: 2},
		{SessionFile: "a.jsonl", EntryID: "2", Folder: "/p", Model: "m", Provider: "p", API: "c", Timestamp: 200, StopReason: "stop", InputTokens: 1, OutputTokens: 1, TotalTokens: 2},
	})

	if n := db.MessageCount(); n != 2 {
		t.Fatalf("expected 2 messages, got %d", n)
	}
}

func TestInsertDuplicateEntry(t *testing.T) {
	db := mustOpenDB(t)
	defer db.Close()

	db.InsertMessages([]MessageStats{
		{SessionFile: "a.jsonl", EntryID: "1", Folder: "/p", Model: "m", Provider: "p", API: "c", Timestamp: 100, StopReason: "stop", InputTokens: 1, OutputTokens: 1, TotalTokens: 2},
	})

	n, err := db.InsertMessages([]MessageStats{
		{SessionFile: "a.jsonl", EntryID: "1", Folder: "/p", Model: "m2", Provider: "p2", API: "c", Timestamp: 200, StopReason: "stop", InputTokens: 1, OutputTokens: 1, TotalTokens: 2},
	})
	if err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("expected 0 inserted (duplicate), got %d", n)
	}
}

func mustOpenDB(t *testing.T) *DB {
	t.Helper()
	db, err := OpenDB(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatal(err)
	}
	return db
}

func itoa(i int) string {
	return string(rune('0' + i))
}
