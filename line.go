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
// logical line, but leave the Text field untouched. These methods are
// used in the --autofix mode.

import (
	"fmt"
	"io"
	"path"
	"strconv"
)

type Line = *LineImpl

type RawLine struct {
	Lineno int
	orignl string
	textnl string
}

func (rline *RawLine) String() string {
	return strconv.Itoa(rline.Lineno) + ":" + rline.textnl
}

type LineImpl struct {
	Filename  string
	firstLine int32 // Zero means not applicable, -1 means EOF
	lastLine  int32 // Usually the same as firstLine, may differ in Makefiles
	Text      string
	raw       []*RawLine
	autofix   *Autofix
}

func NewLine(fname string, lineno int, text string, rawLines []*RawLine) Line {
	return NewLineMulti(fname, lineno, lineno, text, rawLines)
}

// NewLineMulti is for logical Makefile lines that end with backslash.
func NewLineMulti(fname string, firstLine, lastLine int, text string, rawLines []*RawLine) Line {
	return &LineImpl{fname, int32(firstLine), int32(lastLine), text, rawLines, nil}
}

// NewLineEOF creates a dummy line for logging, with the "line number" EOF.
func NewLineEOF(fname string) Line {
	return NewLineMulti(fname, -1, 0, "", nil)
}

// NewLineWhole creates a dummy line for logging messages that affect a file as a whole.
func NewLineWhole(fname string) Line {
	return NewLine(fname, 0, "", nil)
}

func (line *LineImpl) Linenos() string {
	switch {
	case line.firstLine == -1:
		return "EOF"
	case line.firstLine == 0:
		return ""
	case line.firstLine == line.lastLine:
		return strconv.Itoa(int(line.firstLine))
	default:
		return strconv.Itoa(int(line.firstLine)) + "--" + strconv.Itoa(int(line.lastLine))
	}
}

func (line *LineImpl) ReferenceFrom(other Line) string {
	if line.Filename != other.Filename {
		return cleanpath(relpath(path.Dir(other.Filename), line.Filename)) + ":" + line.Linenos()
	}
	return "line " + line.Linenos()
}

func (line *LineImpl) IsMultiline() bool {
	return line.firstLine > 0 && line.firstLine != line.lastLine
}

func (line *LineImpl) printSource(out io.Writer) {
	if G.opts.PrintSource {
		io.WriteString(out, "\n")

		printDiff := func(rawLines []*RawLine) {
			for _, rawLine := range rawLines {
				if rawLine.textnl != rawLine.orignl {
					if rawLine.orignl != "" {
						io.WriteString(out, "- "+rawLine.orignl)
					}
					if rawLine.textnl != "" {
						io.WriteString(out, "+ "+rawLine.textnl)
					}
				} else {
					io.WriteString(out, "> "+rawLine.orignl)
				}
			}
		}

		if line.autofix != nil {
			for _, before := range line.autofix.linesBefore {
				io.WriteString(out, "+ "+before)
			}
			printDiff(line.autofix.lines)
			for _, after := range line.autofix.linesAfter {
				io.WriteString(out, "+ "+after)
			}
		} else {
			printDiff(line.raw)
		}
	}
}

func (line *LineImpl) Fatalf(format string, args ...interface{}) {
	line.printSource(G.logErr)
	logs(llFatal, line.Filename, line.Linenos(), format, fmt.Sprintf(format, args...))
}

func (line *LineImpl) Errorf(format string, args ...interface{}) {
	line.printSource(G.logOut)
	logs(llError, line.Filename, line.Linenos(), format, fmt.Sprintf(format, args...))
}

func (line *LineImpl) Warnf(format string, args ...interface{}) {
	line.printSource(G.logOut)
	logs(llWarn, line.Filename, line.Linenos(), format, fmt.Sprintf(format, args...))
}

func (line *LineImpl) Notef(format string, args ...interface{}) {
	line.printSource(G.logOut)
	logs(llNote, line.Filename, line.Linenos(), format, fmt.Sprintf(format, args...))
}

func (line *LineImpl) String() string {
	return line.Filename + ":" + line.Linenos() + ": " + line.Text
}

// Autofix returns a builder object for automatically fixing the line.
// After building the object, call Apply to actually apply the changes to the line.
//
// The changed lines are not written back to disk immediately.
// This is done by SaveAutofixChanges.
//
func (line *LineImpl) Autofix() *Autofix {
	if line.autofix == nil {
		line.autofix = NewAutofix(line)
	}
	return line.autofix
}
