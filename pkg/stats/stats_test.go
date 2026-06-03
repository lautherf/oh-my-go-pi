package stats

import (
	"testing"
)

func TestComputeUserMetrics_Empty(t *testing.T) {
	m := ComputeUserMetrics("")
	if m.Chars != 0 || m.Words != 0 || m.Profanity != 0 {
		t.Fatalf("expected zero metrics for empty string, got %+v", m)
	}
}

func TestComputeUserMetrics_Whitespace(t *testing.T) {
	m := ComputeUserMetrics("  \n  \t  ")
	if m.Chars != 0 {
		t.Fatalf("expected zero chars for whitespace, got %d", m.Chars)
	}
}

func TestComputeUserMetrics_CharsAndWords(t *testing.T) {
	m := ComputeUserMetrics("hello world")
	if m.Chars != 11 {
		t.Fatalf("expected 11 chars, got %d", m.Chars)
	}
	if m.Words != 2 {
		t.Fatalf("expected 2 words, got %d", m.Words)
	}
}

func TestComputeUserMetrics_Yelling(t *testing.T) {
	m := ComputeUserMetrics("THIS IS ALL CAPS ANGRY TEXT")
	if m.Yelling == 0 {
		t.Fatal("expected yelling > 0 for all-caps sentence")
	}
}

func TestComputeUserMetrics_NoYelling_ShortAcronym(t *testing.T) {
	m := ComputeUserMetrics("OK that works now")
	if m.Yelling != 0 {
		t.Fatalf("expected no yelling for 'OK', got %d", m.Yelling)
	}
}

func TestComputeUserMetrics_Profanity(t *testing.T) {
	m := ComputeUserMetrics("this is fucking bullshit")
	if m.Profanity == 0 {
		t.Fatal("expected profanity > 0")
	}
}

func TestComputeUserMetrics_CodeFencesStripped(t *testing.T) {
	m := ComputeUserMetrics("here is some code\n```\nvar x = 1\n```\n")
	if m.Profanity != 0 {
		t.Fatal("code fences should be stripped; profanity should be 0")
	}
}

func TestComputeUserMetrics_Anguish_Drama(t *testing.T) {
	m := ComputeUserMetrics("what???!!")
	if m.Anguish == 0 {
		t.Fatal("expected anguish > 0 for drama runs")
	}
}

func TestComputeUserMetrics_Anguish_DotRuns(t *testing.T) {
	m := ComputeUserMetrics("I see....")
	if m.Anguish == 0 {
		t.Fatal("expected anguish > 0 for dot runs")
	}
}

func TestComputeUserMetrics_Negation_Lead(t *testing.T) {
	m := ComputeUserMetrics("no that's not what I meant")
	if m.Negation == 0 {
		t.Fatal("expected negation > 0")
	}
}

func TestComputeUserMetrics_Repetition(t *testing.T) {
	m := ComputeUserMetrics("like I said, this is still broken")
	if m.Repetition == 0 {
		t.Fatal("expected repetition > 0")
	}
}

func TestComputeUserMetrics_Blame(t *testing.T) {
	m := ComputeUserMetrics("you didn't do what I asked")
	if m.Blame == 0 {
		t.Fatal("expected blame > 0")
	}
}

func TestComputeUserMetrics_LongFormattedMessageNoSignals(t *testing.T) {
	msg := `I noticed the issue with the error handling in the database module.
The connection pool was not being properly drained during shutdown,
which caused the intermittent timeouts you saw in the logs.

To fix this, we should add a graceful shutdown handler that closes
all idle connections before terminating the process.`
	m := ComputeUserMetrics(msg)
	if m.Anguish != 0 || m.Negation != 0 {
		t.Fatalf("expected zero frustration signals for long formatted message, got anguish=%d negation=%d", m.Anguish, m.Negation)
	}
}

func TestComputeUserMetrics_ProseOnlyScored(t *testing.T) {
	// 2-line prose after stripping code: should still get scored
	m := ComputeUserMetrics("```\ncode block\n```\nno this is wrong!")
	if m.Negation == 0 {
		t.Fatal("expected negation > 0; 2-line prose should be scored")
	}
}

func TestComputeUserMetrics_ThreeLineProseZeroSignals(t *testing.T) {
	m := ComputeUserMetrics("line one\nline two\nline three\n")
	if m.Anguish != 0 {
		t.Fatal("expected anguish=0 for 3-line prose")
	}
}

func TestComputeUserMetrics_BlameStopImperative(t *testing.T) {
	m := ComputeUserMetrics("Stop ignoring my requests!")
	if m.Blame == 0 {
		t.Fatal("expected blame > 0 for 'Stop ignoring'")
	}
}
