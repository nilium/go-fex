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

// Command fex is an implementation of Jordan Sissel's fex(1), found at
// <https://github.com/jordansissel/fex>. This implementation may not be
// entirely compatible with the original fex, as it uses RE2 regular expressions
// is sensitive to UTF-8 arguments and input, and accepts some arguments that
// fex(1) may not (such as `--1` or `/\/$/`.
//
// fex is a field extractor tool, similar to cut or awk, but with less friction
// for common uses of either.
//
// More information and examples can be found by running `fex -h`.
package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"

	"go.spiff.io/go-fex/internal/fex"
)

var version = "DEVELOPMENT"

func main() {
	prog := "fex"
	if base := filepath.Base(os.Args[0]); base != "" {
		prog = base
	}
	bufout := bufio.NewWriter(os.Stdout)
	argv := os.Args[1:]
	fex := &fex.Fex{
		Name:    prog,
		Version: version,
		Stdin:   os.Stdin,
		Stdout:  bufout,
		Stderr:  os.Stderr,
	}
	status := fex.Run(argv)
	if err := bufout.Flush(); err != nil {
		fmt.Fprintf(os.Stderr, "Error flushing output: %v\n", err)
		status |= 1
	}
	os.Exit(status)
}
