// Copyright 2007-2011 Jordan Sissel
// Copyright 2018 Noel Cower (Go implementation)
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package fex

// NOTE: This source file borrows heavily from the original fex.c, including the
// general structure and semantics of the program.

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

// Fex holds main program state for fex, including the program name (used to
// print usage) and stdin/stdout/stderr IOs. Its primary method is Run.
type Fex struct {
	Name    string
	Version string
	Stdin   io.Reader
	Stdout  io.Writer
	Stderr  io.Writer
}

// Run processes Fex's stdin and arguments and writes to either stdout upon
// success or stderr on failure (or if the first argument is a help flag).
//
// In the event of errors, it is possible for some output to be written to
// stdout and stderr.
func (f *Fex) Run(argv []string) int {
	if len(argv) == 0 {
		f.Usage()
		return 2
	}

	switch argv[0] {
	case "-h", "--help":
		f.Usage()
		return 2
	case "-v", "--version":
		f.write(f.Version + "\n")
		return 0
	}

	var (
		rd  = bufio.NewReader(f.Stdin)
		ops = make([]Extractor, len(argv))
	)

	// Parse extractors
	for i, arg := range argv {
		op, err := CompileExtractor(arg)
		if err != nil {
			f.errorf("Error parsing extract %d: %q: %v", i+1, arg, err)
			return 1
		}
		ops[i] = op
	}

	// Run all lines through extractors
	for {
		line, ioerr := rd.ReadString('\n')
		if line == "" && ioerr == io.EOF {
			break
		} else if ioerr != nil && ioerr != io.EOF {
			f.errorf("IO error: %v", ioerr)
		}
		if err := f.processLine(line, ops); err != nil {
			f.errorf("%v", err)
		}
		if ioerr == io.EOF {
			break
		}
	}

	return 0
}

// Usage writes formatted usage text to stderr.
func (f *Fex) Usage() {
	f.errorf(usageFormat, f.Name)
}

func (f *Fex) processLine(line string, ops []Extractor) error {
	line = strings.TrimSuffix(line, "\n")
	line = strings.TrimSuffix(line, "\r")
	written := 0
	for i, op := range ops {
		field, err := op.Extract(line)
		if err != nil {
			return err
		}
		if i > 0 {
			written += f.write(" ")
		}
		written += f.write(field)
	}
	if written > 0 {
		f.write("\n")
	}
	return nil
}

func (f *Fex) errorf(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintln(f.Stderr, msg)
}

func (f *Fex) write(s string) (n int) {
	if s != "" {
		n, _ = io.WriteString(f.Stdout, s)
	}
	return n
}

// Extractor is a sequence of selectors, used to progressively select pieces of
// text.
type Extractor []Selector

// Extract tokenizes and extracts fields from s, returning either a new string
// or an error.
func (e Extractor) Extract(s string) (result string, err error) {
	result = s
	for _, ex := range e {
		result, err = ex.Extract(result)
		if err != nil {
			return "", err
		}
	}
	return result, nil
}

// Selector is an individual part of an extraction string, such as "N",
// "_{?N:M}", or " /Rx/".  A selector tokenizes a string, filters the tokens,
// and returns a new string based on its delimiter.
type Selector struct {
	delim    string
	tokenize func(delim, s string) []string
	filter   Filter
}

func (sel *Selector) Extract(s string) (string, error) {
	fields := sel.tokenize(sel.delim, s)
	fields, err := sel.filter.Select(fields, s)
	if err != nil {
		return "", err
	}
	return strings.Join(fields, sel.delim), nil
}

func CompileExtractor(arg string) (Extractor, error) {
	var (
		ex   Extractor
		sr   = []rune(arg)
		offs = make([]int, len(sr)+1)
		n    = len(sr)
		i    = n - 1
	)

	{ // Compute string offsets of runes
		i := 0
		for off := range arg {
			offs[i] = off
			i++
		}
		// Could omit zeroeth index here but not worth it
		offs[i] = len(arg)
	}

	// Functions for working with the input argument
	var (
		slice = func(i, j int) string {
			return arg[offs[i]:offs[j]]
		}
		find = func(r rune) int {
			for q := i - 1; q >= 0; q-- {
				if sr[q] == r {
					return q
				}
			}
			return -1
		}
		findf = func(f func(r rune) bool) int {
			for q := i - 1; q >= 0; q-- {
				if f(sr[q]) {
					return q
				}
			}
			return -1
		}
	)

	// Walk rune sequence
	for ; i >= 0; i-- {
		var (
			r         = sr[i]
			tokenizer = GreedySplit
			filter    Filter
			err       error
		)

		switch {
		case r == '}': // group selector
			start := find('{')
			if start == -1 {
				return nil, fmt.Errorf("extractor has unmatched '}' at character %d", i+1)
			}
			sub := slice(start+1, i)
			if len(sub) > 0 && sub[0] == '?' {
				tokenizer = NonGreedySplit
				sub = sub[1:]
			}

			var group Group
			group, err = ParseGroup(sub)
			if err != nil {
				return nil, err
			}
			filter = group
			i = start - 1

		case r == '/': // regexp selector
			var chunks []string
			start := find('/')
			for ; start != -1; start = find('/') {
				if chunk := slice(start+1, i); chunk != "" {
					chunks = append(chunks, chunk)
				}
				i = start

				escaped := start > 1 && sr[start-1] == '\\'
				if !escaped {
					i--
					break
				}
				i = findf(func(r rune) bool { return r != '\\' }) + 1
				escapes := start - i
				if escapes%2 == 0 {
					// Not escaped -- delimiter is a backslash
					break
				}
				chunks = append(chunks, slice(start-(escapes-1)/2, start+1))
			}

			for j := len(chunks)/2 - 1; j >= 0; j-- {
				opp := len(chunks) - 1 - j
				chunks[j], chunks[opp] = chunks[opp], chunks[j]
			}
			rx := strings.Join(chunks, "")

			filter, err = NewRegexpFilter(rx)
			if err != nil {
				return nil, err
			}

		case unicode.IsDigit(r): // Simple selector
			start := findf(func(r rune) bool { return !unicode.IsDigit(r) })
			if start > -1 && sr[start] == '-' {
				start--
			}
			digits := slice(start+1, i+1)
			var fr FieldRange
			fr, err = ParseFieldRange(digits)
			if err != nil {
				return nil, err
			}
			filter = fr
			i = start

		default:
			return nil, fmt.Errorf("unexpected %q in selector", r)
		}

		sep := " "
		if i > 0 && sr[i-1] == '\\' {
			switch sr[i] {
			case '\\':
				sep = "\\"
			case 'a':
				sep = "\a"
			case 'b':
				sep = "\b"
			case 'f':
				sep = "\f"
			case 'n':
				sep = "\n"
			case 'r':
				sep = "\r"
			case 't':
				sep = "\t"
			case 'v':
				sep = "\v"
			case 'z':
				sep = "\x00"
			case 'e':
				sep = "\x1B"
			default:
				sep = slice(i, i+1)
			}
			i--
		} else if i >= 0 {
			sep = slice(i, i+1)
		}

		ex = append(ex, Selector{
			delim:    sep,
			tokenize: tokenizer,
			filter:   filter,
		})
	}

	for j := len(ex)/2 - 1; j >= 0; j-- {
		opp := len(ex) - 1 - j
		ex[j], ex[opp] = ex[opp], ex[j]
	}

	return ex, nil
}

// Filter selects a subset of fields plus the zero string to extract from the
// input. For selectors that can return the original string, the zero string is
// provided.
type Filter interface {
	Select(fields []string, zero string) ([]string, error)
}

// Group is a collection of field ranges, such as {1} or {1,4:5} or {-2:-1}.
// It is not responsible for distinguishing between greedy and non-greedy
// groupings.
type Group []FieldRange

func (g Group) Select(fields []string, zero string) ([]string, error) {
	fs := make([]string, 0, 4)
	for _, fr := range g {
		selected, err := fr.Select(fields, zero)
		if err != nil {
			return nil, err
		}
		fs = append(fs, selected...)
	}
	return fs, nil
}

func ParseGroup(s string) (Group, error) {
	specs := strings.Split(s, ",")
	g := make(Group, len(specs))
	for i, spec := range specs {
		fr, err := ParseFieldRange(spec)
		if err != nil {
			return nil, fmt.Errorf("cannot parse %q: %v", spec, err)
		}
		g[i] = fr
	}
	return g, nil
}

// FieldRange is an inclusive range of [Start, End], where indices at Start
// begin with 1.  A FieldRange with a negative start or end (or both) is
// considered relative, and cannot be considered valid until an absolute
// FieldRange has been created using abs.
//
// The empty FieldRange refers to {0:0}, {0}, or just 0 as a field index
// (meaning the original string).
type FieldRange struct {
	Start int
	End   int
}

func ParseFieldRange(s string) (FieldRange, error) {
	n := strings.IndexByte(s, ':')
	if n == -1 {
		i, err := strconv.Atoi(s)
		if err != nil {
			return FieldRange{}, err
		}
		return FieldRange{Start: i, End: i}, nil
	}

	var (
		start, end = s[:n], s[n+1:]
		err        error
		f          FieldRange
	)

	if start == "" && end == "" {
		return FieldRange{}, nil
	} else if start == "" {
		f.Start = 1
	} else if f.Start, err = strconv.Atoi(start); err != nil {
		return FieldRange{}, err
	}

	if end == "" {
		f.End = -1
	} else if f.End, err = strconv.Atoi(end); err != nil {
		return FieldRange{}, err
	}

	if f.Start > f.End && ((f.Start < 0 && f.End < 0) || (f.Start > 0 && f.End > 0)) {
		return FieldRange{},
			fmt.Errorf("start > end is invalid: %d > %d", f.Start, f.End)
	} else if (f.Start == 0 || f.End == 0) && f.Start != f.End {
		return FieldRange{},
			fmt.Errorf("start or end cannot be 0 when the other is not 0: %d and %d", f.Start, f.End)
	}

	return f, err
}

func (r FieldRange) Select(fields []string, zero string) ([]string, error) {
	if r.Start == 0 && r.End == 0 {
		return []string{zero}, nil
	}

	r = r.abs(fields)
	if !r.isValid() {
		return nil, nil
	}

	start, end := r.Start-1, r.End // [start, end)
	if n := len(fields); start > n {
		return nil, nil
	} else if end > n {
		end = n
	}
	fs := make([]string, end-start)
	copy(fs, fields[start:])
	return fs, nil
}

func (r FieldRange) abs(fields []string) FieldRange {
	n := len(fields)
	r.Start = abs(r.Start, n)
	r.End = abs(r.End, n)
	if r.End < r.Start {
		r.End = r.Start
	}
	return r
}

// isValid returns whether a FieldRange is valid.
// A FieldRange with negative or zero'd offsets is not valid.
func (r FieldRange) isValid() bool {
	return r.Start > 0 && r.End > 0 && r.Start <= r.End
}

// RegexpFilter selects fields matching its regular expression.
// Unlike fex proper, this uses Go's RE2 regular expressions, so it's not
// promised to be backwards compatible.
type RegexpFilter regexp.Regexp

func NewRegexpFilter(s string) (*RegexpFilter, error) {

	rx, err := regexp.Compile(s)
	if err != nil {
		return nil, err
	}
	return (*RegexpFilter)(rx), nil
}

func (r *RegexpFilter) regexp() *regexp.Regexp {
	return (*regexp.Regexp)(r)
}

func (r *RegexpFilter) Select(fields []string, _ string) ([]string, error) {
	rx := r.regexp()
	fs := make([]string, 0, len(fields))
	for _, f := range fields {
		if rx.MatchString(f) {
			fs = append(fs, f)
		}
	}
	return fs, nil
}

// Split functions

type splitFunc func(delim, s string) []string

// greedySplit splits s along a delimiter, omitting empty splits from the
// resulting slice.
//
// For example, ":foo:" split by ":" will produce []string{"foo"}.
// This can result in empty slices.
func GreedySplit(delim, s string) []string {
	fn := func(r rune) bool {
		for _, dr := range delim {
			if r == dr {
				return true
			}
		}
		return false
	}
	return strings.FieldsFunc(s, fn)
}

// nonGreedySplit splits s along a delimiter, retaining empty slices.
// For example, ":foo:" split by ":" will produce []string{"", "foo", ""}.
func NonGreedySplit(delim, s string) []string {
	return strings.Split(s, delim)
}

// Utility functions

// abs returns an offset relative to a range [1, length].
// Negative rel values are offsets from the end of a sequence, going down.
// Positive (including zero) rel values are offsets from the start of
// a sequence, going up. The result is not bounds-checked.
func abs(rel, length int) int {
	if rel < 0 {
		rel = length + 1 + rel
	}
	return rel
}
