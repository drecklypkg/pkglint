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
	"strconv"
	"strings"
)

type RawLine struct {
	lineno int
	orignl string
	textnl string
}

func (rline *RawLine) String() string {
	return strconv.Itoa(rline.lineno) + ":" + rline.textnl
}

type Line struct {
	fname          string
	firstLine      int32 // Zero means not applicable, -1 means EOF
	lastLine       int32 // Usually the same as firstLine, may differ in Makefiles
	text           string
	raw            []*RawLine
	changed        bool
	before         []*RawLine
	after          []*RawLine
	autofixMessage *string
}

func NewLine(fname string, lineno int, text string, rawLines []*RawLine) *Line {
	return NewLineMulti(fname, lineno, lineno, text, rawLines)
}

// NewLineMulti is for logical Makefile lines that end with backslash.
func NewLineMulti(fname string, firstLine, lastLine int, text string, rawLines []*RawLine) *Line {
	return &Line{fname, int32(firstLine), int32(lastLine), text, rawLines, false, nil, nil, nil}
}

// NewLineEof creates a dummy line for logging.
func NewLineEof(fname string) *Line {
	return NewLineMulti(fname, -1, 0, "", nil)
}

func (ln *Line) rawLines() []*RawLine {
	switch { // prevent inlining
	}
	return append(append(append([]*RawLine(nil), ln.before...), ln.raw...), ln.after...)
}

func (ln *Line) linenos() string {
	switch {
	case ln.firstLine == -1:
		return "EOF"
	case ln.firstLine == 0:
		return ""
	case ln.firstLine == ln.lastLine:
		return strconv.Itoa(int(ln.firstLine))
	default:
		return strconv.Itoa(int(ln.firstLine)) + "--" + strconv.Itoa(int(ln.lastLine))
	}
}

func (ln *Line) IsMultiline() bool {
	return ln.firstLine > 0 && ln.firstLine != ln.lastLine
}

func (ln *Line) printSource(out io.Writer) {
	if G.opts.PrintSource {
		io.WriteString(out, "\n")
		for _, rawLine := range ln.rawLines() {
			if rawLine.textnl != rawLine.orignl {
				io.WriteString(out, "- "+rawLine.orignl)
				io.WriteString(out, "+ "+rawLine.textnl)
			} else {
				io.WriteString(out, "> "+rawLine.orignl)
			}
		}
	}
}

func (ln *Line) fatalf(format string, args ...interface{}) {
	ln.printSource(G.logErr)
	fatalf(ln.fname, ln.linenos(), format, args...)
}

func (ln *Line) errorf(format string, args ...interface{}) {
	ln.printSource(G.logOut)
	errorf(ln.fname, ln.linenos(), format, args...)
	ln.logAutofix()
}
func (ln *Line) error0(format string)             { ln.errorf(format) }
func (ln *Line) error1(format, arg1 string)       { ln.errorf(format, arg1) }
func (ln *Line) error2(format, arg1, arg2 string) { ln.errorf(format, arg1, arg2) }

func (ln *Line) warnf(format string, args ...interface{}) {
	ln.printSource(G.logOut)
	warnf(ln.fname, ln.linenos(), format, args...)
	ln.logAutofix()
}
func (ln *Line) warn0(format string)             { ln.warnf(format) }
func (ln *Line) warn1(format, arg1 string)       { ln.warnf(format, arg1) }
func (ln *Line) warn2(format, arg1, arg2 string) { ln.warnf(format, arg1, arg2) }

func (ln *Line) notef(format string, args ...interface{}) {
	ln.printSource(G.logOut)
	notef(ln.fname, ln.linenos(), format, args...)
	ln.logAutofix()
}
func (ln *Line) note0(format string)             { ln.notef(format) }
func (ln *Line) note1(format, arg1 string)       { ln.notef(format, arg1) }
func (ln *Line) note2(format, arg1, arg2 string) { ln.notef(format, arg1, arg2) }

func (ln *Line) debugf(format string, args ...interface{}) {
	ln.printSource(G.logOut)
	debugf(ln.fname, ln.linenos(), format, args...)
	ln.logAutofix()
}
func (ln *Line) debug1(format, arg1 string)       { ln.debugf(format, arg1) }
func (ln *Line) debug2(format, arg1, arg2 string) { ln.debugf(format, arg1, arg2) }

func (ln *Line) String() string {
	return ln.fname + ":" + ln.linenos() + ": " + ln.text
}

func (ln *Line) logAutofix() {
	if ln.autofixMessage != nil {
		autofixf(ln.fname, ln.linenos(), "%s", *ln.autofixMessage)
		ln.autofixMessage = nil
	}
}

func (ln *Line) autofixInsertBefore(line string) bool {
	if G.opts.PrintAutofix || G.opts.Autofix {
		ln.before = append(ln.before, &RawLine{0, "", line + "\n"})
	}
	return ln.noteAutofix("Inserting a line %q before this line.", line)
}

func (ln *Line) autofixInsertAfter(line string) bool {
	if G.opts.PrintAutofix || G.opts.Autofix {
		ln.after = append(ln.after, &RawLine{0, "", line + "\n"})
	}
	return ln.noteAutofix("Inserting a line %q after this line.", line)
}

func (ln *Line) autofixDelete() bool {
	if G.opts.PrintAutofix || G.opts.Autofix {
		ln.raw = nil
	}
	return ln.noteAutofix("Deleting this line.")
}

func (ln *Line) autofixReplace(from, to string) bool {
	for _, rawLine := range ln.raw {
		if rawLine.lineno != 0 {
			if replaced := strings.Replace(rawLine.textnl, from, to, 1); replaced != rawLine.textnl {
				if G.opts.PrintAutofix || G.opts.Autofix {
					rawLine.textnl = replaced
				}
				return ln.noteAutofix("Replacing %q with %q.", from, to)
			}
		}
	}
	return false
}

func (ln *Line) autofixReplaceRegexp(from, to string) bool {
	for _, rawLine := range ln.raw {
		if rawLine.lineno != 0 {
			if replaced := regcomp(from).ReplaceAllString(rawLine.textnl, to); replaced != rawLine.textnl {
				if G.opts.PrintAutofix || G.opts.Autofix {
					rawLine.textnl = replaced
				}
				return ln.noteAutofix("Replacing regular expression %q with %q.", from, to)
			}
		}
	}
	return false
}

func (ln *Line) noteAutofix(format string, args ...interface{}) (hasBeenFixed bool) {
	if ln.firstLine < 1 {
		return false
	}
	ln.changed = true
	if G.opts.Autofix {
		autofixf(ln.fname, ln.linenos(), format, args...)
		return true
	}
	if G.opts.PrintAutofix {
		msg := fmt.Sprintf(format, args...)
		ln.autofixMessage = &msg
	}
	return false
}

func (ln *Line) checkAbsolutePathname(text string) {
	if G.opts.DebugTrace {
		defer tracecall1("Line.checkAbsolutePathname", text)()
	}

	// In the GNU coding standards, DESTDIR is defined as a (usually
	// empty) prefix that can be used to install files to a different
	// location from what they have been built for. Therefore
	// everything following it is considered an absolute pathname.
	//
	// Another context where absolute pathnames usually appear is in
	// assignments like "bindir=/bin".
	if m, path := match1(text, `(?:^|\$[{(]DESTDIR[)}]|[\w_]+\s*=\s*)(/(?:[^"'\s]|"[^"*]"|'[^']*')*)`); m {
		if matches(path, `^/\w`) {
			checkwordAbsolutePathname(ln, path)
		}
	}
}

func (line *Line) checkLength(maxlength int) {
	if len(line.text) > maxlength {
		line.warnf("Line too long (should be no more than %d characters).", maxlength)
		explain3(
			"Back in the old time, terminals with 80x25 characters were common.",
			"And this is still the default size of many terminal emulators.",
			"Moderately short lines also make reading easier.")
	}
}

func (line *Line) checkValidCharacters(reChar string) {
	rest := regcomp(reChar).ReplaceAllString(line.text, "")
	if rest != "" {
		uni := ""
		for _, c := range rest {
			uni += fmt.Sprintf(" %U", c)
		}
		line.warn1("Line contains invalid characters (%s).", uni[1:])
	}
}

func (line *Line) checkTrailingWhitespace() {
	if hasSuffix(line.text, " ") || hasSuffix(line.text, "\t") {
		if !line.autofixReplaceRegexp(`\s+\n$`, "\n") {
			line.note0("Trailing white-space.")
			explain2(
				"When a line ends with some white-space, that space is in most cases",
				"irrelevant and can be removed.")
		}
	}
}

func checklineRcsid(line *Line, prefixRe, suggestedPrefix string) bool {
	if G.opts.DebugTrace {
		defer tracecall2("checklineRcsid", prefixRe, suggestedPrefix)()
	}

	if matches(line.text, `^`+prefixRe+`\$`+`NetBSD(?::[^\$]+)?\$$`) {
		return true
	}

	if !line.autofixInsertBefore(suggestedPrefix + "$" + "NetBSD$") {
		line.error1("Expected %q.", suggestedPrefix+"$"+"NetBSD$")
		explain3(
			"Several files in pkgsrc must contain the CVS Id, so that their current",
			"version can be traced back later from a binary package. This is to",
			"ensure reproducible builds, for example for finding bugs.")
	}
	return false
}
