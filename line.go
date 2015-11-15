package main

// When files are read in by pkglint, they are interpreted in terms of
// lines. For Makefiles, line continuations are handled properly, allowing
// multiple raw lines to end in a single logical line. For other files
// there is a 1:1 translation.
//
// A difference between the raw and the logical lines is that the
// raw lines include the line end sequence, whereas the logical lines
// do not.
//
// Some methods allow modification of the raw lines contained in the
// logical line, but leave the “text” field untouched. These methods are
// used in the --autofix mode.

import (
	"fmt"
	"io"
	"strings"
)

type RawLine struct {
	lineno int
	textnl string
}

type Line struct {
	fname   string
	lines   string
	text    string
	raw     []RawLine
	changed bool
	before  []RawLine
	after   []RawLine
	extra   map[string]interface{}
}

func NewLine(fname, linenos, text string, rawLines []RawLine) *Line {
	return &Line{fname, linenos, text, rawLines, false, []RawLine{}, []RawLine{}, make(map[string]interface{})}
}

func (self *Line) rawLines() []RawLine {
	return append(self.before, append(self.raw, self.after...)...)
}
func (self *Line) printSource(out io.Writer) {
	if G.opts.PrintSource {
		io.WriteString(out, "\n")
		for _, rawLine := range self.rawLines() {
			fmt.Fprintf(out, "> %s", rawLine.textnl)
		}
	}
}
func (self *Line) fatalf(format string, args ...interface{}) bool {
	self.printSource(G.logErr)
	return fatalf(self.fname, self.lines, format, args...)
}
func (self *Line) errorf(format string, args ...interface{}) bool {
	self.printSource(G.logOut)
	return errorf(self.fname, self.lines, format, args...)
}
func (self *Line) warnf(format string, args ...interface{}) bool {
	self.printSource(G.logOut)
	return warnf(self.fname, self.lines, format, args...)
}
func (self *Line) notef(format string, args ...interface{}) bool {
	self.printSource(G.logOut)
	return notef(self.fname, self.lines, format, args...)
}
func (self *Line) debugf(format string, args ...interface{}) bool {
	self.printSource(G.logOut)
	return debugf(self.fname, self.lines, format, args...)
}
func (self *Line) explain(explanation ...string) {
	if G.opts.Explain {
		io.WriteString(G.logOut, "\n")
		for _, explanationLine := range explanation {
			io.WriteString(G.logOut, "\t"+explanationLine+"\n")
		}
		io.WriteString(G.logOut, "\n")
	} else {
		G.explanationsAvailable = true
	}
}
func (self *Line) String() string {
	return self.fname + ":" + self.lines + ": " + self.text
}

func (self *Line) prependBefore(line string) {
	self.before = append([]RawLine{{0, line + "\n"}}, self.before...)
	self.changed = true
}
func (self *Line) appendBefore(line string) {
	self.before = append(self.before, RawLine{0, line + "\n"})
	self.changed = true
}
func (self *Line) prependAfter(line string) {
	self.after = append([]RawLine{{0, line + "\n"}}, self.after...)
	self.changed = true
}
func (self *Line) appendAfter(line string) {
	self.after = append(self.after, RawLine{0, line + "\n"})
	self.changed = true
}
func (self *Line) delete() {
	self.raw = nil
	self.changed = true
}
func (self *Line) replace(from, to string) {
	for _, rawLine := range self.raw {
		if rawLine.lineno != 0 {
			if replaced := strings.Replace(rawLine.textnl, from, to, 1); replaced != rawLine.textnl {
				rawLine.textnl = replaced
				self.changed = true
			}
		}
	}
}
func (self *Line) replaceRegex(from, to string) {
	for _, rawLine := range self.raw {
		if rawLine.lineno != 0 {
			if replaced := regcomp(from).ReplaceAllString(rawLine.textnl, to); replaced != rawLine.textnl {
				rawLine.textnl = replaced
				self.changed = true
			}
		}
	}
}
func (line *Line) setText(text string) {
	line.raw = []RawLine{{0, text + "\n"}}
	line.changed = true
}
