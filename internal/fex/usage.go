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

const usageFormat = `Usage: %s <extract1> [extract...]

Extract syntax is one or more selectors, formatted as:

    <delimiter><field number(s)>

Fields start at 1, same as AWK. A field number of 0 selects the whole
string, unchanged.

You can specify multiple fields with curly braces and numbers split by
commas. Also valid in curly braces {} are number ranges. Number ranges
are similar to python array slices, split by colon.

The first separator is implied as space (' '), but can be overridden.
For example, to select the last dash-separated field, you can use --1.

You can match fields by a regexp (regular expression) by following the
separator with a /regexp/. If the separator is a backslash, it must be
escaped by writing two backslashes. Forward slashes and backslashes in
the regexp can also be escaped.

Regular expressions are RE2. To use a backslash separator with a regexp
RE2 syntax: <https://github.com/google/re2/wiki/Syntax>.

Some examples:

    1.1        First split by ' ', then first by '.'.
               'foo.bar baz' by '1.1' outputs 'foo'.

    0:{1,-1}   Output the first and last split by ':'.
               'foo:bar:baz:fizz' by '0:{1,-1}' outputs 'foo:fizz'.

    {1:3}      Output tokens 1 through 3.
               'foo bar baz fizz' by '{1:3}' outputs 'foo bar baz'.

    :/home/    First split by ':' and yield only fields matching the
               regexp /home/.

    \\/addr/   First split by '\' ('\\' to escape it) and yield only
               fields matching the regexp /addr/.

Make sure you quote your extractions, or your shell may perform
some unintended expansion.`
