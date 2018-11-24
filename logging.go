package main

import (
	"fmt"
	"io"
	"strings"
)

type LogLevel struct {
	TraditionalName string
	GccName         string
}

var (
	Fatal           = &LogLevel{"FATAL", "fatal"}
	Error           = &LogLevel{"ERROR", "error"}
	Warn            = &LogLevel{"WARN", "warning"}
	Note            = &LogLevel{"NOTE", "note"}
	AutofixLogLevel = &LogLevel{"AUTOFIX", "autofix"}
)

var dummyLine = NewLineMulti("", 0, 0, "", nil)

// Explain outputs an explanation for the preceding diagnostic
// if the --explain option is given. Otherwise it just records
// that an explanation is available.
func (pkglint *Pkglint) Explain(explanation ...string) {

	// TODO: Add automatic word wrapping so that the pkglint source
	// code doesn't need to be concerned with manual line wrapping.

	if pkglint.Testing {
		for _, s := range explanation {
			if l := tabWidth(s); l > 68 && contains(s, " ") {
				lastSpace := strings.LastIndexByte(s[:68], ' ')
				pkglint.logErr.Printf("Long explanation line: %s\nBreak after: %s\n", s, s[:lastSpace])
			}
			if m, before := match1(s, `(.+)\. [^ ]`); m {
				if !matches(before, `\d$|e\.g`) {
					// TODO: Find out why this rule exists. It's the same as in
					// the NetBSD manual pages, but seems otherwise unnecessary.
					pkglint.logErr.Printf("Short space after period: %s\n", s)
				}
			}
			if hasSuffix(s, " ") || hasSuffix(s, "\t") {
				pkglint.logErr.Printf("Trailing whitespace: %q\n", s)
			}
		}
	}

	if !pkglint.explainNext {
		return
	}
	pkglint.explanationsAvailable = true
	if !pkglint.Opts.Explain {
		return
	}

	if !pkglint.explained.FirstTimeSlice(explanation...) {
		return
	}

	pkglint.logOut.WriteLine("")
	for _, explanationLine := range explanation {
		pkglint.logOut.WriteLine("\t" + explanationLine)
	}
	pkglint.logOut.WriteLine("")

}

type pkglintFatal struct{}

// SeparatorWriter writes output, occasionally separated by an
// empty line. This is used for separating the diagnostics when
// --source is combined with --show-autofix, where each
// log message consists of multiple lines.
type SeparatorWriter struct {
	out            io.Writer
	needSeparator  bool
	wroteSomething bool
}

func NewSeparatorWriter(out io.Writer) *SeparatorWriter {
	return &SeparatorWriter{out, false, false}
}

func (wr *SeparatorWriter) WriteLine(text string) {
	wr.Write(text)
	_, _ = io.WriteString(wr.out, "\n")
}

func (wr *SeparatorWriter) Write(text string) {
	if wr.needSeparator && wr.wroteSomething {
		_, _ = io.WriteString(wr.out, "\n")
		wr.needSeparator = false
	}
	n, err := io.WriteString(wr.out, text)
	if err == nil && n > 0 {
		wr.wroteSomething = true
	}
}

func (wr *SeparatorWriter) Printf(format string, args ...interface{}) {
	wr.Write(fmt.Sprintf(format, args...))
}

func (wr *SeparatorWriter) Separate() {
	wr.needSeparator = true
}
