package native

import (
	"unicode"

	"github.com/atotto/clipboard"
)

// CountTokens estimates token count for o200k/cl100k tokenizers.
// Uses byte-pair encoding approximation: common words ~1 token,
// rare/short words ~2-3 tokens, CJK chars ~2-4 tokens each.
func CountTokens(text string) int {
	if text == "" {
		return 0
	}

	count := 0
	inWord := false
	wordLen := 0
	cjk := false

	for _, r := range text {
		if unicode.Is(unicode.Han, r) || unicode.Is(unicode.Hiragana, r) ||
			unicode.Is(unicode.Katakana, r) || unicode.Is(unicode.Hangul, r) {
			// CJK: each char ~2-4 tokens depending on frequency
			if inWord {
				count += tokenEstimate(wordLen, cjk)
				inWord = false
				wordLen = 0
			}
			count += 2
			continue
		}

		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
			if !inWord {
				inWord = true
				wordLen = 0
				cjk = false
			}
			wordLen++
		} else {
			if inWord {
				count += tokenEstimate(wordLen, false)
				inWord = false
				wordLen = 0
			}
			// punctuation/whitespace: rarely tokenized alone
			if r == '\n' {
				count++ // newline is usually a token
			}
		}
	}
	if inWord {
		count += tokenEstimate(wordLen, false)
	}

	if count == 0 {
		count = len(text) / 4
		if count < 1 {
			count = 1
		}
	}
	return count
}

func tokenEstimate(length int, cjk bool) int {
	if cjk {
		return length * 3 / 2
	}
	// BPE approximation:
	//   very common short words -> 1 token
	//   avg English word -> 1-2 tokens
	//   long/rare words -> ceil(length/4)
	switch {
	case length <= 2:
		return 1
	case length <= 4:
		return 1
	case length <= 6:
		return 2
	default:
		return 1 + (length-6+3)/4
	}
}

func CopyToClipboard(text string) error {
	return clipboard.WriteAll(text)
}

func ReadClipboard() (string, error) {
	return clipboard.ReadAll()
}
