package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"netbsd.org/pkglint/regex"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/check.v1"
	"netbsd.org/pkglint/textproc"
)

var equals = check.Equals
var deepEquals = check.DeepEquals

const RcsID = "$" + "NetBSD$"
const MkRcsID = "# $" + "NetBSD$"
const PlistRcsID = "@comment $" + "NetBSD$"

type Suite struct {
	Tester *Tester
}

// Init creates and returns a test helper that allows to:
//
// * create files for the test
//
// * load these files into Line and MkLine objects (for tests spanning multiple files)
//
// * create new in-memory Line and MkLine objects (for simple tests)
//
// * check the files that have been changed by the --autofix feature
//
// * check the pkglint diagnostics
func (s *Suite) Init(c *check.C) *Tester {

	// Note: the check.C object from SetUpTest cannot be used here,
	// and the parameter given here cannot be used in TearDownTest;
	// see https://github.com/go-check/check/issues/22.

	t := s.Tester // Has been initialized by SetUpTest
	if t.checkC != nil {
		panic("Suite.Init must only be called once.")
	}
	t.checkC = c
	return t
}

func (s *Suite) SetUpTest(c *check.C) {
	t := &Tester{checkC: c}
	s.Tester = t

	G = NewPkglint()
	G.Testing = true
	textproc.Testing = true
	G.logOut = NewSeparatorWriter(&t.stdout)
	G.logErr = NewSeparatorWriter(&t.stderr)
	trace.Out = &t.stdout
	G.Pkgsrc = NewPkgsrc(t.File("."))

	t.checkC = c
	t.SetupCommandLine("-Wall") // To catch duplicate warnings
	t.checkC = nil

	G.opts.LogVerbose = true // To detect duplicate work being done
	t.EnableSilentTracing()

	prevdir, err := os.Getwd()
	if err != nil {
		c.Fatalf("Cannot get current working directory: %s", err)
	}
	t.prevdir = prevdir
}

func (s *Suite) TearDownTest(c *check.C) {
	t := s.Tester
	t.checkC = nil // No longer usable; see https://github.com/go-check/check/issues/22

	if err := os.Chdir(t.prevdir); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Cannot chdir back to previous dir: %s", err)
	}

	G = Pkglint{} // unusable because of missing logOut and logErr
	textproc.Testing = false
	if out := t.Output(); out != "" {
		_, _ = fmt.Fprintf(os.Stderr,
			"\nUnchecked output in %q; check with: t.CheckOutputLines(%#v)\n",
			c.TestName(), strings.Split(out, "\n"))
	}
	t.tmpdir = ""
	t.DisableTracing()
}

var _ = check.Suite(new(Suite))

func Test(t *testing.T) { check.TestingT(t) }

// Tester provides utility methods for testing pkglint.
// It is separated from the Suite since the latter contains
// all the test methods, which makes it difficult to find
// a method by auto-completion.
type Tester struct {
	stdout  bytes.Buffer
	stderr  bytes.Buffer
	tmpdir  string
	checkC  *check.C // Only usable during the test method itself
	prevdir string   // The current working directory before the test started
	relcwd  string
}

func (t *Tester) c() *check.C {
	if t.checkC == nil {
		panic("Suite.Init must be called before accessing check.C.")
	}
	return t.checkC
}

// SetupCommandLine simulates a command line for the remainder of the test.
// See Pkglint.ParseCommandLine.
//
// If SetupCommandLine is not called explicitly in a test, the command line
// "-Wall" is used, to provide a high code coverage in the tests.
func (t *Tester) SetupCommandLine(args ...string) {

	// Prevent tracing from being disabled; see EnableSilentTracing.
	prevTracing := trace.Tracing
	defer func() { trace.Tracing = prevTracing }()

	exitcode := G.ParseCommandLine(append([]string{"pkglint"}, args...))
	if exitcode != nil && *exitcode != 0 {
		t.CheckOutputEmpty()
		t.c().Fatalf("Cannot parse command line: %#v", args)
	}
	G.opts.LogVerbose = true // See SetUpTest
}

// SetupVartypes registers a few hundred variables like MASTER_SITES,
// WRKSRC, SUBST_SED.*, so that their data types are known to pkglint.
func (t *Tester) SetupVartypes() {
	G.Pkgsrc.InitVartypes()
}

func (t *Tester) SetupMasterSite(varname string, urls ...string) {
	name2url := &G.Pkgsrc.MasterSiteVarToURL
	url2name := &G.Pkgsrc.MasterSiteURLToVar
	if *name2url == nil {
		*name2url = make(map[string]string)
		*url2name = make(map[string]string)
	}
	(*name2url)[varname] = urls[0]
	for _, url := range urls {
		(*url2name)[url] = varname
	}
}

// SetupOption pretends that the package option is defined in mk/defaults/options.description.
func (t *Tester) SetupOption(name, description string) {
	G.Pkgsrc.PkgOptions[name] = description
}

func (t *Tester) SetupTool(name, varname string, validity Validity) *Tool {
	return G.Pkgsrc.Tools.defTool(name, varname, false, validity)
}

// SetupFileLines creates a temporary file and writes the given lines to it.
// The file is then read in, without considering line continuations.
func (t *Tester) SetupFileLines(relativeFileName string, lines ...string) Lines {
	fileName := t.CreateFileLines(relativeFileName, lines...)
	return Load(fileName, MustSucceed)
}

// SetupFileLines creates a temporary file and writes the given lines to it.
// The file is then read in, handling line continuations for Makefiles.
func (t *Tester) SetupFileMkLines(relativeFileName string, lines ...string) MkLines {
	fileName := t.CreateFileLines(relativeFileName, lines...)
	return LoadMk(fileName, MustSucceed)
}

// SetupPkgsrc sets up a minimal but complete pkgsrc installation in the
// temporary folder, so that pkglint runs without any errors.
// Individual files may be overwritten by calling other Setup* methods.
// This setup is especially interesting for testing Pkglint.Main.
func (t *Tester) SetupPkgsrc() {

	// This file is needed to locate the pkgsrc root directory.
	// See findPkgsrcTopdir.
	t.CreateFileLines("mk/bsd.pkg.mk",
		MkRcsID)

	// See Pkgsrc.loadDocChanges.
	t.CreateFileLines("doc/CHANGES-2018",
		RcsID)

	// See Pkgsrc.loadSuggestedUpdates.
	t.CreateFileLines("doc/TODO",
		RcsID)

	// Some example licenses so that the tests for whole packages
	// don't need to define them on their own.
	t.CreateFileLines("licenses/2-clause-bsd",
		"Redistribution and use in source and binary forms ...")
	t.CreateFileLines("licenses/gnu-gpl-v2",
		"The licenses for most software ...")

	// The MASTER_SITES in the package Makefile are searched here.
	// See Pkgsrc.loadMasterSites.
	t.CreateFileLines("mk/fetch/sites.mk",
		MkRcsID)

	// The options for the PKG_OPTIONS framework must be readable.
	// See Pkgsrc.loadPkgOptions.
	t.CreateFileLines("mk/defaults/options.description")

	// The user-defined variables are read in to check for missing
	// BUILD_DEFS declarations in the package Makefile.
	t.CreateFileLines("mk/defaults/mk.conf",
		MkRcsID)

	// The tool definitions are read in to check for missing
	// USE_TOOLS declarations in the package Makefile.
	// They spread over several files from the pkgsrc infrastructure.
	t.CreateFileLines("mk/tools/bsd.tools.mk",
		".include \"defaults.mk\"")
	t.CreateFileLines("mk/tools/defaults.mk",
		MkRcsID)
	t.CreateFileLines("mk/bsd.prefs.mk", // Some tools are defined here.
		MkRcsID)
}

// SetupCategory makes the given category valid by creating a dummy Makefile.
func (t *Tester) SetupCategory(name string) {
	if _, err := os.Stat(name + "/Makefile"); os.IsNotExist(err) {
		t.CreateFileLines(name+"/Makefile",
			MkRcsID)
	}
}

// SetupPackage sets up all files for a package (including the pkgsrc
// infrastructure) so that it does not produce any warnings. After calling
// this method, individual files can be overwritten as necessary.
//
// The given makefileLines start in line 20. Except if they are variable
// definitions for already existing variables, then they replace that line.
//
// Returns the path to the package, ready to be used with Pkglint.CheckDirent.
func (t *Tester) SetupPackage(pkgpath string, makefileLines ...string) string {
	category := path.Dir(pkgpath)

	t.SetupPkgsrc()
	t.SetupVartypes()
	t.SetupCategory(category)

	t.CreateFileLines(pkgpath+"/DESCR",
		"Package description")
	t.CreateFileLines(pkgpath+"/PLIST",
		PlistRcsID,
		"bin/program")
	t.CreateFileLines(pkgpath+"/distinfo",
		RcsID,
		"",
		"SHA1 (distfile-1.0.tar.gz) = 12341234...",
		"RMD160 (distfile-1.0.tar.gz) = 12341234...",
		"SHA512 (distfile-1.0.tar.gz) = 12341234...",
		"Size (distfile-1.0.tar.gz) = 12341234")

	var mlines []string
	mlines = append(mlines,
		MkRcsID,
		"",
		"DISTNAME=\tdistname-1.0",
		"CATEGORIES=\t"+category,
		"MASTER_SITES=\t# none",
		"",
		"MAINTAINER=\tpkgsrc-users@NetBSD.org",
		"HOMEPAGE=\t# none",
		"COMMENT=\tDummy package",
		"LICENSE=\t2-clause-bsd",
		"")
	for len(mlines) < 19 {
		mlines = append(mlines, "# empty")
	}

line:
	for _, line := range makefileLines {
		if m, prefix := match1(line, `^(\w+=)`); m {
			for i, existingLine := range mlines {
				if hasPrefix(existingLine, prefix) {
					mlines[i] = line
					continue line
				}
			}
		}
		mlines = append(mlines, line)
	}

	mlines = append(mlines,
		"",
		".include \"../../mk/bsd.pkg.mk\"")

	t.CreateFileLines(pkgpath+"/Makefile",
		mlines...)

	return t.File(pkgpath)
}

func (t *Tester) CreateFileLines(relativeFileName string, lines ...string) (fileName string) {
	content := ""
	for _, line := range lines {
		content += line + "\n"
	}

	fileName = t.File(relativeFileName)
	err := os.MkdirAll(path.Dir(fileName), 0777)
	t.c().Assert(err, check.IsNil)

	err = ioutil.WriteFile(fileName, []byte(content), 0666)
	t.c().Check(err, check.IsNil)

	G.fileCache.Evict(fileName)

	return fileName
}

// CreateFileDummyPatch creates a patch file with the given name in the
// temporary directory.
func (t *Tester) CreateFileDummyPatch(relativeFileName string) {
	t.CreateFileLines(relativeFileName,
		RcsID,
		"",
		"Documentation",
		"",
		"--- oldfile",
		"+++ newfile",
		"@@ -1 +1 @@",
		"-old",
		"+new")
}

// File returns the absolute path to the given file in the
// temporary directory. It doesn't check whether that file exists.
// Calls to Tester.Chdir change the base directory for the relative file name.
func (t *Tester) File(relativeFileName string) string {
	if t.tmpdir == "" {
		t.tmpdir = filepath.ToSlash(t.c().MkDir())
	}
	if t.relcwd != "" {
		return cleanpath(relativeFileName)
	}
	return cleanpath(t.tmpdir + "/" + relativeFileName)
}

// Chdir changes the current working directory to the given subdirectory
// of the temporary directory, creating it if necessary.
//
// After this call, all files loaded from the temporary directory via
// SetupFileLines or CreateFileLines or similar methods will use path names
// relative to this directory.
//
// After the test, the previous working directory is restored, so that
// the other tests are unaffected.
//
// As long as this method is not called in a test, the current working
// directory is indeterminate.
func (t *Tester) Chdir(relativeFileName string) {
	if t.relcwd != "" {
		// When multiple calls of Chdir are mixed with calls to CreateFileLines,
		// the resulting Lines and MkLines variables will use relative file names,
		// and these will point to different areas in the file system. This is
		// usually not indented and therefore prevented.
		t.checkC.Fatalf("Chdir must only be called once per test; already in %q.", t.relcwd)
	}

	_ = os.MkdirAll(t.File(relativeFileName), 0700)
	if err := os.Chdir(t.File(relativeFileName)); err != nil {
		t.checkC.Fatalf("Cannot chdir: %s", err)
	}
	t.relcwd = relativeFileName
}

// Remove removes the file from the temporary directory. The file must exist.
func (t *Tester) Remove(relativeFileName string) {
	fileName := t.File(relativeFileName)
	err := os.Remove(fileName)
	t.c().Check(err, check.IsNil)
	G.fileCache.Evict(fileName)
}

// ExpectFatal runs the given action and expects that this action calls
// Line.Fatalf or uses some other way to panic with a pkglintFatal.
//
// Usage:
//  t.ExpectFatal(
//      func() { /* do something that panics */ },
//      "FATAL: ~/Makefile:1: Must not be empty")
func (t *Tester) ExpectFatal(action func(), expectedLines ...string) {
	defer func() {
		r := recover()
		if r == nil {
			panic("Expected a pkglint fatal error, but didn't get one.")
		} else if _, ok := r.(pkglintFatal); ok {
			t.CheckOutputLines(expectedLines...)
		} else {
			panic(r)
		}
	}()

	action()
}

// ExpectFatalMatches runs the given action and expects that this action
// calls Line.Fatalf or uses some other way to panic with a pkglintFatal.
// It then matches the output against a regular expression.
//
// Usage:
//  t.ExpectFatalMatches(
//      func() { /* do something that panics */ },
//      `FATAL: ~/Makefile:1: .*\n`)
func (t *Tester) ExpectFatalMatches(action func(), expected regex.Pattern) {
	defer func() {
		r := recover()
		if r == nil {
			panic("Expected a pkglint fatal error, but didn't get one.")
		} else if _, ok := r.(pkglintFatal); ok {
			t.c().Check(t.Output(), check.Matches, string(expected))
		} else {
			panic(r)
		}
	}()

	action()
}

// Arguments are either (lineno, orignl) or (lineno, orignl, textnl).
func (t *Tester) NewRawLines(args ...interface{}) []*RawLine {
	rawlines := make([]*RawLine, len(args)/2)
	j := 0
	for i := 0; i < len(args); i += 2 {
		lineno := args[i].(int)
		orignl := args[i+1].(string)
		textnl := orignl
		if i+2 < len(args) {
			if s, ok := args[i+2].(string); ok {
				textnl = s
				i++
			}
		}
		rawlines[j] = &RawLine{lineno, orignl, textnl}
		j++
	}
	return rawlines[:j]
}

func (t *Tester) NewLine(fileName string, lineno int, text string) Line {
	textnl := text + "\n"
	rawLine := RawLine{lineno, textnl, textnl}
	return NewLine(fileName, lineno, text, []*RawLine{&rawLine})
}

func (t *Tester) NewMkLine(fileName string, lineno int, text string) MkLine {
	return NewMkLine(t.NewLine(fileName, lineno, text))
}

func (t *Tester) NewShellLine(fileName string, lineno int, text string) *ShellLine {
	return NewShellLine(t.NewMkLine(fileName, lineno, text))
}

// NewLines generates a slice of simple lines,
// i.e. each logical line has exactly one physical line.
// To work with line continuations like in Makefiles,
// use CreateFileLines together with LoadExistingLines.
func (t *Tester) NewLines(fileName string, lines ...string) Lines {
	return t.NewLinesAt(fileName, 1, lines...)
}

// NewLinesAt generates a slice of simple lines,
// i.e. each logical line has exactly one physical line.
// To work with line continuations like in Makefiles,
// use Suite.CreateFileLines together with Suite.LoadExistingLines.
func (t *Tester) NewLinesAt(fileName string, firstLine int, texts ...string) Lines {
	result := make([]Line, len(texts))
	for i, text := range texts {
		textnl := text + "\n"
		result[i] = NewLine(fileName, i+firstLine, text, t.NewRawLines(i+firstLine, textnl))
	}
	return NewLines(fileName, result)
}

// NewMkLines creates new in-memory objects for the given lines,
// as if they were parsed from a Makefile fragment.
// No actual file is created for the lines; see SetupFileMkLines for that.
func (t *Tester) NewMkLines(fileName string, lines ...string) MkLines {
	rawText := ""
	for _, line := range lines {
		rawText += line + "\n"
	}
	return NewMkLines(convertToLogicalLines(fileName, rawText, true))
}

// Returns and consumes the output from both stdout and stderr.
// The temporary directory is replaced with a tilde (~).
func (t *Tester) Output() string {
	stdout := t.stdout.String()
	stderr := t.stderr.String()

	t.stdout.Reset()
	t.stderr.Reset()

	output := stdout + stderr
	if t.tmpdir != "" {
		output = strings.Replace(output, t.tmpdir, "~", -1)
	}
	return output
}

func (t *Tester) CheckOutputEmpty() {
	t.CheckOutputLines( /* none */ )
}

// CheckOutputLines checks that the output up to now equals the given lines.
// After the comparison, the output buffers are cleared so that later
// calls only check against the newly added output.
func (t *Tester) CheckOutputLines(expectedLines ...string) {
	output := t.Output()
	actualLines := strings.Split(output, "\n")
	actualLines = actualLines[:len(actualLines)-1]
	t.c().Check(emptyToNil(actualLines), deepEquals, emptyToNil(expectedLines))
}

// EnableTracing redirects all logging output (which is normally captured
// in an in-memory buffer) additionally to stdout.
// This is useful when stepping through the code, especially
// in combination with SetupCommandLine("--debug").
//
// In JetBrains GoLand, the tracing output is suppressed after the first
// failed check, see https://youtrack.jetbrains.com/issue/GO-6154.
func (t *Tester) EnableTracing() {
	G.logOut = NewSeparatorWriter(io.MultiWriter(os.Stdout, &t.stdout))
	trace.Out = os.Stdout
	trace.Tracing = true
}

// EnableTracingToLog enables the tracing and writes the tracing output
// to the test log that can be examined with Tester.Output.
func (t *Tester) EnableTracingToLog() {
	G.logOut = NewSeparatorWriter(io.MultiWriter(os.Stdout, &t.stdout))
	trace.Out = &t.stdout
	trace.Tracing = true
}

// EnableSilentTracing enables tracing mode, but discards any tracing output.
// This can be used to improve code coverage without any side-effects,
// since tracing output is quite large.
func (t *Tester) EnableSilentTracing() {
	trace.Out = ioutil.Discard
	trace.Tracing = true
}

// DisableTracing logs the output to the buffers again, ready to be
// checked with CheckOutputLines.
func (t *Tester) DisableTracing() {
	G.logOut = NewSeparatorWriter(&t.stdout)
	trace.Tracing = false
	trace.Out = nil
}

// CheckFileLines loads the lines from the temporary file and checks that
// they equal the given lines.
func (t *Tester) CheckFileLines(relativeFileName string, lines ...string) {
	content, err := ioutil.ReadFile(t.File(relativeFileName))
	t.c().Assert(err, check.IsNil)
	text := string(content)
	actualLines := strings.Split(text, "\n")
	actualLines = actualLines[:len(actualLines)-1]
	t.c().Check(emptyToNil(actualLines), deepEquals, emptyToNil(lines))
}

// CheckFileLinesDetab loads the lines from the temporary file and checks
// that they equal the given lines. The loaded file may use tabs or spaces
// for indentation, while the lines in the code use spaces exclusively,
// in order to make the depth of the indentation clearly visible.
func (t *Tester) CheckFileLinesDetab(relativeFileName string, lines ...string) {
	actualLines := Load(t.File(relativeFileName), MustSucceed)

	var detabbed []string
	for _, line := range actualLines.Lines {
		rawText := strings.TrimRight(detab(line.raw[0].orignl), "\n")
		detabbed = append(detabbed, rawText)
	}

	t.c().Check(detabbed, deepEquals, lines)
}
