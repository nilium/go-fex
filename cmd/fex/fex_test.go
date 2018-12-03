package main

import (
	"bytes"
	"strings"
	"testing"
)

// nonEmpty is a special value that indicates an output shouldn't be empty but not what its contents
// are.
const nonEmpty = "\x00NON-EMPTY\x00"

// TestCase is a simple Fex test case that ensures that argumetns and input produce expected output
// and status codes.
//
// If WantErr's value is nonEmpty, stderr output is not checked except to ensure that it's not
// empty.
type TestCase struct {
	Args    []string
	Status  int
	Input   string
	Want    string
	WantErr string
}

func (tc *TestCase) Run(t *testing.T) {
	var (
		stdout = new(bytes.Buffer)
		stderr = new(bytes.Buffer)
		input  = strings.NewReader(tc.Input)
		fex    = &Fex{
			Name:   "fex",
			Stdin:  input,
			Stdout: stdout,
			Stderr: stderr,
		}
	)

	t.Logf("Args = %q", tc.Args)
	status := fex.Run(tc.Args)
	if status != tc.Status {
		t.Errorf("fex.Run(...) = %d; want %d", status, tc.Status)
	}

	if strout := stdout.String(); strout != tc.Want {
		t.Errorf("Unexpected stdout output:\n--- Output ---\n%s--- Wanted ---\n%s",
			strout, tc.Want)
	}

	switch strerr := stderr.String(); tc.WantErr {
	case nonEmpty:
		if strerr == "" {
			t.Errorf("Unexpected stderr output: got %q; want non-empty string", strerr)
		}
	case strerr:
		// Nop
	default:
		t.Errorf("Unexpected stderr output:\n--- Output ---\n%s--- Wanted ---\n%s",
			strerr, tc.WantErr)
	}
}

func wantLines(lines ...string) string {
	return strings.Join(lines, "\n") + "\n"
}

// testCases run by TestInputs.
var testCases = map[string]*TestCase{
	// Check that -h, -help, and --help all return status code 2 and don't print anything to
	// stdout.
	"HelpShort": &TestCase{
		Args:    []string{`-h`, `1`},
		Status:  2,
		Input:   "1 a 2 b 3 c 4 d 5 e",
		WantErr: nonEmpty,
	},

	"HelpShortLong": &TestCase{
		Args:    []string{`-help`, `1`},
		Status:  2,
		Input:   "1 a 2 b 3 c 4 d 5 e",
		WantErr: nonEmpty,
	},

	"HelpLong": &TestCase{
		Args:    []string{`--help`, `1`},
		Status:  2,
		Input:   "1 a 2 b 3 c 4 d 5 e",
		WantErr: nonEmpty,
	},

	//
	// Test selectors
	//

	"Basic": &TestCase{
		Args: []string{`1`, `3`},
		Input: wantLines(
			`1 2 3 4 5`,
			`foo bar baz fizz pizza`,
			`foo    bar     baz`,
			`1 1234 asdfasd.faw.gfaw.t.23r.1234'lJF#@ JJ@#FJJj`,
		),
		Want: wantLines(
			`1 3`,
			`foo baz`,
			`foo baz`,
			`1 asdfasd.faw.gfaw.t.23r.1234'lJF#@`,
		),
	},

	"MultiFields": &TestCase{
		Args: []string{`{1,2,-1}`},
		Input: wantLines(
			`1 2 3 4 5 6 7 8 9 10`,
			`1 2 3 4 5 6 7 8 9 10`,
			`foo bar baz fizz`,
		),
		Want: wantLines(
			`1 2 10`,
			`1 2 10`,
			`foo bar fizz`,
		),
	},

	"MultiSplit": &TestCase{
		Args: []string{`a{1,2,3}`},
		Input: wantLines(
			`fooabuzzaflorbadiss`,
			`1a2a3a4a5`,
		),
		Want: wantLines(
			`fooabuzzaflorb`,
			`1a2a3`,
		),
	},

	"OtherSplit": &TestCase{
		Args:  []string{`a1`, `b1`, `c1`, `d2`},
		Input: "abcdefgh",
		Want:  "bcdefgh a ab efgh\n",
	},

	"NumRange": &TestCase{
		Args: []string{`{1:3}`},
		Input: wantLines(
			`20 21 22 23 24 25 26 27 28 29 30 31 32 33 34 35 36 37 38 39 40`,
			`1 2 3 4 5`,
		),
		Want: wantLines(
			`20 21 22`,
			`1 2 3`,
		),
	},

	"ZeroRange": &TestCase{
		Args: []string{`{0}`, `{0:0}`, `0`},
		Input: wantLines(
			`a b c`,
		),
		Want: wantLines(
			`a b c a b c a b c`,
		),
	},

	"ZeroRangePreserved": &TestCase{
		Args: []string{`:{0}`, `:{0:0}`, `:0`},
		Input: wantLines(
			`:a::b::c:`,
		),
		Want: wantLines(
			`:a::b::c: :a::b::c: :a::b::c:`,
		),
	},

	"ImpossibleField": &TestCase{
		Args: []string{`123456789`},
		Input: wantLines(
			`a b c d e f g`,
		),
	},

	"ImpossibleRelativeField": &TestCase{
		Args: []string{`-123456789`},
		Input: wantLines(
			`a b c d e f g`,
		),
	},

	"OutOfRange": &TestCase{
		Args: []string{`{-5:100}`},
		Input: wantLines(
			`a b c d e f g`,
		),
		Want: wantLines(
			`c d e f g`,
		),
	},

	// Regular expressions
	"Regexp": &TestCase{
		Args: []string{`/-/--1`, ` /-/`},
		Input: wantLines(
			`a-b c d e-f`,
			`foobar baz big-turtle`,
		),
		Want: wantLines(
			`f a-b e-f`,
			`turtle big-turtle`,
		),
	},

	"RegexpEscape-ForwardSlash": &TestCase{
		Args:  []string{` /\w\// -1/1`},
		Input: "foo/bar baz/ /what\n",
		Want:  "baz\n",
	},

	"RegexpEscape-Backslash": &TestCase{
		Args:  []string{` /\w\\/ -1\1`},
		Input: "foo\\bar baz\\ \\what\n",
		Want:  "baz\n",
	},

	"RegexpEscape-BackslashDelim": &TestCase{
		Args:  []string{`\\/\//\-1/1`},
		Input: "baz/\\foo/bar\\what\n",
		Want:  "foo\n",
	},

	// Tokenizing options {...} and {?...}
	"Greedy": &TestCase{
		Args:  []string{`:{3}`},
		Input: "foo:::bar:::baz:::fizz\n",
		Want:  "baz\n",
	},

	"NonGreedy": &TestCase{
		Args:  []string{`:{?4}`},
		Input: "foo:::bar:::baz:::fizz\n",
		Want:  "bar\n",
	},

	// Invalid ranges
	"BadRelativeRange": &TestCase{
		Args:   []string{`{-2:-3}`},
		Status: 1,
		Input: wantLines(
			`a b c d e f g`,
		),
		WantErr: wantLines(
			`Error parsing extract 1: "{-2:-3}": cannot parse "-2:-3": start > end is invalid: -2 > -3`,
		),
	},

	"BadAbsoluteRange": &TestCase{
		Args:   []string{`{1,3:1}`},
		Status: 1,
		Input: wantLines(
			`a b c d e f g`,
		),
		WantErr: wantLines(
			`Error parsing extract 1: "{1,3:1}": cannot parse "3:1": start > end is invalid: 3 > 1`,
		),
	},
}

func TestInputs(t *testing.T) {
	for name, tc := range testCases {
		tc := tc
		t.Run(name, tc.Run)
	}
}
