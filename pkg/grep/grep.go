package grep

import (
	"bufio"
	"errors"
	"os"
	"path/filepath"
	"regexp"

	"github.com/gobwas/glob"
)

type Match struct {
	Path       string
	LineNumber int
	Line       string
	Lines      []string // context lines if requested
}

type Options struct {
	CaseSensitive bool
	Include       string
	Exclude       string
	Context       int
}

func Search(root, pattern string, opts *Options) ([]Match, error) {
	if pattern == "" {
		return nil, errors.New("empty pattern")
	}

	opts = applyDefaults(opts)

	re, err := compileRegex(pattern, opts.CaseSensitive)
	if err != nil {
		return nil, err
	}

	var includeGlob, excludeGlob glob.Glob
	if opts.Include != "" {
		includeGlob, err = glob.Compile(opts.Include)
		if err != nil {
			return nil, err
		}
	}
	if opts.Exclude != "" {
		excludeGlob, err = glob.Compile(opts.Exclude)
		if err != nil {
			return nil, err
		}
	}

	var matches []Match
	err = filepath.Walk(root, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return nil // skip inaccessible
		}
		if fi.IsDir() {
			return nil
		}
		if shouldSkip(path, fi, includeGlob, excludeGlob) {
			return nil
		}

		fileMatches, err := searchFile(path, re, opts.Context)
		if err != nil {
			return nil
		}
		matches = append(matches, fileMatches...)
		return nil
	})

	return matches, err
}

func applyDefaults(opts *Options) *Options {
	if opts == nil {
		opts = &Options{}
	}
	return opts
}

func compileRegex(pattern string, caseSensitive bool) (*regexp.Regexp, error) {
	if caseSensitive {
		return regexp.Compile(pattern)
	}
	return regexp.Compile("(?i)" + pattern)
}

func shouldSkip(path string, fi os.FileInfo, include, exclude glob.Glob) bool {
	if include != nil && !include.Match(fi.Name()) {
		return true
	}
	if exclude != nil && exclude.Match(fi.Name()) {
		return true
	}
	return false
}

func searchFile(path string, re *regexp.Regexp, context int) ([]Match, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var matches []Match
	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	for i, line := range lines {
		if re.MatchString(line) {
			m := Match{
				Path:       path,
				LineNumber: i + 1,
				Line:       line,
			}
			if context > 0 {
				start := i - context
				if start < 0 {
					start = 0
				}
				end := i + context + 1
				if end > len(lines) {
					end = len(lines)
				}
				m.Lines = lines[start:end]
			}
			matches = append(matches, m)
		}
	}

	return matches, nil
}

