package main

import (
	"fmt"
	"netbsd.org/pkglint/getopt"
	"netbsd.org/pkglint/histogram"
	"netbsd.org/pkglint/regex"
	tracePkg "netbsd.org/pkglint/trace"
	"os"
	"os/user"
	"path"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"strings"
)

const confMake = "@BMAKE@"
const confVersion = "@VERSION@"

// Pkglint contains all global variables of this Go package.
// The rest of the global state is in the other packages:
//  regex.Profiling    (not thread-local)
//  regex.res          (and related variables; not thread-safe)
//  textproc.Testing   (not thread-local; harmless)
//  tracing.Tracing    (not thread-safe)
//  tracing.Out        (not thread-safe)
//  tracing.traceDepth (not thread-safe)
type Pkglint struct {
	Opts   CmdOpts  // Command line options.
	Pkgsrc *Pkgsrc  // Global data, mostly extracted from mk/*, never nil.
	Pkg    *Package // The package that is currently checked, or nil.
	Mk     MkLines  // The Makefile (or fragment) that is currently checked, or nil.

	Todo            []string // The files or directories that still need to be checked.
	Wip             bool     // Is the currently checked item from pkgsrc-wip?
	Infrastructure  bool     // Is the currently checked item from the pkgsrc infrastructure?
	Testing         bool     // Is pkglint in self-testing mode (only during development)?
	CurrentUsername string   // For checking against OWNER and MAINTAINER
	CvsEntriesDir   string   // Cached to avoid I/O
	CvsEntriesLines Lines

	errors                int
	warnings              int
	explainNext           bool
	logged                Once
	explanationsAvailable bool
	explanationsGiven     map[string]bool
	autofixAvailable      bool
	logOut                *SeparatorWriter
	logErr                *SeparatorWriter

	loghisto  *histogram.Histogram
	loaded    *histogram.Histogram
	res       regex.Registry
	fileCache *FileCache
}

func NewPkglint() Pkglint {
	return Pkglint{
		res:       regex.NewRegistry(),
		fileCache: NewFileCache(200)}
}

type CmdOpts struct {
	CheckAlternatives,
	CheckBuildlink3,
	CheckDescr,
	CheckDistinfo,
	CheckExtra,
	CheckGlobal,
	CheckInstall,
	CheckMakefile,
	CheckMessage,
	CheckMk,
	CheckOptions,
	CheckPatches,
	CheckPlist bool

	WarnAbsname,
	WarnDirectcmd,
	WarnExtra,
	WarnOrder,
	WarnPerm,
	WarnPlistDepr,
	WarnPlistSort,
	WarnQuoting,
	WarnSpace,
	WarnStyle,
	WarnTypes bool

	Explain,
	Autofix,
	GccOutput,
	ShowHelp,
	DumpMakefile,
	Import,
	LogVerbose,
	Profiling,
	Quiet,
	Recursive,
	ShowAutofix,
	ShowSource,
	ShowVersion bool

	LogOnly []string

	args []string
}

type Hash struct {
	hash string
	line Line
}

// G is the abbreviation for "global state";
// these are the only global variable in this Go package
var (
	G     = NewPkglint()
	trace tracePkg.Tracer
	exit  = os.Exit // Indirect access, to allow main() to be tested.
)

func main() {
	G.logOut = NewSeparatorWriter(os.Stdout)
	G.logErr = NewSeparatorWriter(os.Stderr)
	trace.Out = os.Stdout
	exitcode := G.Main(os.Args...)
	if G.Opts.Profiling {
		G = Pkglint{} // Free all memory.
		runtime.GC()  // Detect possible memory leaks.
	}
	exit(exitcode)
}

// Main runs the main program with the given arguments.
// argv[0] is the program name.
//
// Note: during tests, calling this method disables tracing
// because the command line option --debug sets trace.Tracing
// back to false.
//
// It also discards the -Wall option that is used by default in other tests.
func (pkglint *Pkglint) Main(argv ...string) (exitcode int) {
	defer func() {
		if r := recover(); r != nil {
			if _, ok := r.(pkglintFatal); ok {
				exitcode = 1
			} else {
				panic(r)
			}
		}
	}()

	if exitcode := pkglint.ParseCommandLine(argv); exitcode != nil {
		return *exitcode
	}

	if pkglint.Opts.Profiling {
		f, err := os.Create("pkglint.pprof")
		if err != nil {
			dummyLine.Fatalf("Cannot create profiling file: %s", err)
		}
		defer f.Close()

		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()

		pkglint.res.Profiling()
		pkglint.loghisto = histogram.New()
		pkglint.loaded = histogram.New()
		defer func() {
			pkglint.logOut.Write("")
			pkglint.loghisto.PrintStats(pkglint.logOut.out, "loghisto", -1)
			pkglint.res.PrintStats(pkglint.logOut.out)
			pkglint.loaded.PrintStats(pkglint.logOut.out, "loaded", 10)
			pkglint.logOut.WriteLine(fmt.Sprintf("fileCache: %d hits, %d misses", pkglint.fileCache.hits, pkglint.fileCache.misses))
		}()
	}

	for _, arg := range pkglint.Opts.args {
		pkglint.Todo = append(pkglint.Todo, filepath.ToSlash(arg))
	}
	if len(pkglint.Todo) == 0 {
		pkglint.Todo = []string{"."}
	}

	firstArg := pkglint.Todo[0]
	if fileExists(firstArg) {
		firstArg = path.Dir(firstArg)
	}
	relTopdir := findPkgsrcTopdir(firstArg)
	if relTopdir == "" {
		// If the first argument to pkglint is not inside a pkgsrc tree,
		// pkglint doesn't know where to load the infrastructure files from,
		// and these are needed for virtually every single check.
		// Therefore, the only sensible thing to do is to quit immediately.
		dummyLine.Fatalf("%q must be inside a pkgsrc tree.", firstArg)
	}

	pkglint.Pkgsrc = NewPkgsrc(firstArg + "/" + relTopdir)
	pkglint.Pkgsrc.LoadInfrastructure()

	currentUser, err := user.Current()
	if err == nil {
		// On Windows, this is `Computername\Username`.
		pkglint.CurrentUsername = replaceAll(currentUser.Username, `^.*\\`, "")
	}

	for len(pkglint.Todo) > 0 {
		item := pkglint.Todo[0]
		pkglint.Todo = pkglint.Todo[1:]
		pkglint.CheckDirent(item)
	}

	pkglint.Pkgsrc.checkToplevelUnusedLicenses()

	pkglint.ShowSummary()
	if pkglint.errors != 0 {
		return 1
	}
	return 0
}

func (pkglint *Pkglint) ParseCommandLine(args []string) *int {
	gopts := &pkglint.Opts
	opts := getopt.NewOptions()

	check := opts.AddFlagGroup('C', "check", "check,...", "enable or disable specific checks")
	opts.AddFlagVar('d', "debug", &trace.Tracing, false, "log verbose call traces for debugging")
	opts.AddFlagVar('e', "explain", &gopts.Explain, false, "explain the diagnostics or give further help")
	opts.AddFlagVar('f', "show-autofix", &gopts.ShowAutofix, false, "show what pkglint can fix automatically")
	opts.AddFlagVar('F', "autofix", &gopts.Autofix, false, "try to automatically fix some errors")
	opts.AddFlagVar('g', "gcc-output-format", &gopts.GccOutput, false, "mimic the gcc output format")
	opts.AddFlagVar('h', "help", &gopts.ShowHelp, false, "show a detailed usage message")
	opts.AddFlagVar('I', "dumpmakefile", &gopts.DumpMakefile, false, "dump the Makefile after parsing")
	opts.AddFlagVar('i', "import", &gopts.Import, false, "prepare the import of a wip package")
	opts.AddFlagVar('m', "log-verbose", &gopts.LogVerbose, false, "allow the same diagnostic more than once")
	opts.AddStrList('o', "only", &gopts.LogOnly, "only log diagnostics containing the given text")
	opts.AddFlagVar('p', "profiling", &gopts.Profiling, false, "profile the executing program")
	opts.AddFlagVar('q', "quiet", &gopts.Quiet, false, "don't show a summary line when finishing")
	opts.AddFlagVar('r', "recursive", &gopts.Recursive, false, "check subdirectories, too")
	opts.AddFlagVar('s', "source", &gopts.ShowSource, false, "show the source lines together with diagnostics")
	opts.AddFlagVar('V', "version", &gopts.ShowVersion, false, "show the version number of pkglint")
	warn := opts.AddFlagGroup('W', "warning", "warning,...", "enable or disable groups of warnings")

	check.AddFlagVar("ALTERNATIVES", &gopts.CheckAlternatives, true, "check ALTERNATIVES files")
	check.AddFlagVar("bl3", &gopts.CheckBuildlink3, true, "check buildlink3.mk files")
	check.AddFlagVar("DESCR", &gopts.CheckDescr, true, "check DESCR file")
	check.AddFlagVar("distinfo", &gopts.CheckDistinfo, true, "check distinfo file")
	check.AddFlagVar("extra", &gopts.CheckExtra, false, "check various additional files")
	check.AddFlagVar("global", &gopts.CheckGlobal, false, "inter-package checks")
	check.AddFlagVar("INSTALL", &gopts.CheckInstall, true, "check INSTALL and DEINSTALL scripts")
	check.AddFlagVar("Makefile", &gopts.CheckMakefile, true, "check Makefiles")
	check.AddFlagVar("MESSAGE", &gopts.CheckMessage, true, "check MESSAGE file")
	check.AddFlagVar("mk", &gopts.CheckMk, true, "check other .mk files")
	check.AddFlagVar("options", &gopts.CheckOptions, true, "check options.mk files")
	check.AddFlagVar("patches", &gopts.CheckPatches, true, "check patches")
	check.AddFlagVar("PLIST", &gopts.CheckPlist, true, "check PLIST files")

	warn.AddFlagVar("absname", &gopts.WarnAbsname, false, "warn about use of absolute filenames")
	warn.AddFlagVar("directcmd", &gopts.WarnDirectcmd, true, "warn about use of direct command names instead of Make variables")
	warn.AddFlagVar("extra", &gopts.WarnExtra, false, "enable some extra warnings")
	warn.AddFlagVar("order", &gopts.WarnOrder, true, "warn if Makefile entries are unordered")
	warn.AddFlagVar("perm", &gopts.WarnPerm, false, "warn about unforeseen variable definition and use")
	warn.AddFlagVar("plist-depr", &gopts.WarnPlistDepr, false, "warn about deprecated paths in PLISTs")
	warn.AddFlagVar("plist-sort", &gopts.WarnPlistSort, false, "warn about unsorted entries in PLISTs")
	warn.AddFlagVar("quoting", &gopts.WarnQuoting, false, "warn about quoting issues")
	warn.AddFlagVar("space", &gopts.WarnSpace, false, "warn about inconsistent use of whitespace")
	warn.AddFlagVar("style", &gopts.WarnStyle, false, "warn about stylistic issues")
	warn.AddFlagVar("types", &gopts.WarnTypes, true, "do some simple type checking in Makefiles")

	remainingArgs, err := opts.Parse(args)
	if err != nil {
		_, _ = fmt.Fprintf(pkglint.logErr.out, "%s\n\n", err)
		opts.Help(pkglint.logErr.out, "pkglint [options] dir...")
		exitcode := 1
		return &exitcode
	}
	gopts.args = remainingArgs

	if gopts.ShowHelp {
		opts.Help(pkglint.logOut.out, "pkglint [options] dir...")
		exitcode := 0
		return &exitcode
	}

	if pkglint.Opts.ShowVersion {
		_, _ = fmt.Fprintf(pkglint.logOut.out, "%s\n", confVersion)
		exitcode := 0
		return &exitcode
	}

	return nil
}

func (pkglint *Pkglint) ShowSummary() {
	if !pkglint.Opts.Quiet && !pkglint.Opts.Autofix {
		if pkglint.errors != 0 || pkglint.warnings != 0 {
			pkglint.logOut.Printf("%d %s and %d %s found.\n",
				pkglint.errors, ifelseStr(pkglint.errors == 1, "error", "errors"),
				pkglint.warnings, ifelseStr(pkglint.warnings == 1, "warning", "warnings"))
		} else {
			pkglint.logOut.WriteLine("Looks fine.")
		}
		if pkglint.explanationsAvailable && !pkglint.Opts.Explain {
			pkglint.logOut.WriteLine("(Run \"pkglint -e\" to show explanations.)")
		}
		if pkglint.autofixAvailable && !pkglint.Opts.ShowAutofix {
			pkglint.logOut.WriteLine("(Run \"pkglint -fs\" to show what can be fixed automatically.)")
		}
		if pkglint.autofixAvailable && !pkglint.Opts.Autofix {
			pkglint.logOut.WriteLine("(Run \"pkglint -F\" to automatically fix some issues.)")
		}
	}
}

func (pkglint *Pkglint) CheckDirent(filename string) {
	if trace.Tracing {
		defer trace.Call1(filename)()
	}

	st, err := os.Lstat(filename)
	if err != nil || !st.Mode().IsDir() && !st.Mode().IsRegular() {
		NewLineWhole(filename).Errorf("No such file or directory.")
		return
	}
	isDir := st.Mode().IsDir()
	isReg := st.Mode().IsRegular()

	dir := ifelseStr(isReg, path.Dir(filename), filename)
	pkgsrcRel := pkglint.Pkgsrc.ToRel(dir)
	pkglint.Wip = matches(pkgsrcRel, `^wip(/|$)`)
	pkglint.Infrastructure = matches(pkgsrcRel, `^mk(/|$)`)
	pkgsrcdir := findPkgsrcTopdir(dir)
	if pkgsrcdir == "" {
		NewLineWhole(filename).Errorf("Cannot determine the pkgsrc root directory for %q.", cleanpath(dir))
		return
	}

	switch {
	case isDir && isEmptyDir(filename):
		return
	case isReg:
		pkglint.Checkfile(filename)
		return
	}

	switch pkgsrcdir {
	case "../..":
		pkglint.checkdirPackage(dir)
	case "..":
		CheckdirCategory(dir)
	case ".":
		CheckdirToplevel(dir)
	default:
		NewLineWhole(filename).Errorf("Cannot check directories outside a pkgsrc tree.")
	}
}

// checkdirPackage checks a complete pkgsrc package, including each
// of the files individually, and also when seen in combination.
func (pkglint *Pkglint) checkdirPackage(dir string) {
	if trace.Tracing {
		defer trace.Call1(dir)()
	}

	pkglint.Pkg = NewPackage(dir)
	defer func() { pkglint.Pkg = nil }()
	pkg := pkglint.Pkg

	// Load the package Makefile and all included files,
	// to collect all used and defined variables and similar data.
	mklines := pkg.loadPackageMakefile()
	if mklines == nil {
		return
	}

	files := dirglob(pkg.File("."))
	if pkg.Pkgdir != "." {
		files = append(files, dirglob(pkg.File(pkg.Pkgdir))...)
	}
	if pkglint.Opts.CheckExtra {
		files = append(files, dirglob(pkg.File(pkg.Filesdir))...)
	}
	files = append(files, dirglob(pkg.File(pkg.Patchdir))...)
	if pkg.DistinfoFile != pkg.vars.fallback["DISTINFO_FILE"] {
		files = append(files, pkg.File(pkg.DistinfoFile))
	}

	haveDistinfo := false
	havePatches := false

	// Determine the used variables and PLIST directories before checking any of the Makefile fragments.
	for _, filename := range files {
		basename := path.Base(filename)
		if (hasPrefix(basename, "Makefile.") || hasSuffix(filename, ".mk")) &&
			!matches(filename, `patch-`) &&
			!contains(filename, pkg.Pkgdir+"/") &&
			!contains(filename, pkg.Filesdir+"/") {
			if fragmentMklines := LoadMk(filename, MustSucceed); fragmentMklines != nil {
				fragmentMklines.DetermineUsedVariables()
			}
		}
		if hasPrefix(basename, "PLIST") {
			pkg.loadPlistDirs(filename)
		}
	}

	for _, filename := range files {
		if containsVarRef(filename) {
			if trace.Tracing {
				trace.Stepf("Skipping file %q because the name contains an unresolved variable.", filename)
			}
			continue
		}

		if path.Base(filename) == "Makefile" {
			if st, err := os.Lstat(filename); err == nil {
				pkglint.checkExecutable(filename, st)
			}
			if pkglint.Opts.CheckMakefile {
				pkg.checkfilePackageMakefile(filename, mklines)
			}
		} else {
			pkglint.Checkfile(filename)
		}
		if contains(filename, "/patches/patch-") {
			havePatches = true
		} else if hasSuffix(filename, "/distinfo") {
			haveDistinfo = true
		}
		pkg.checkLocallyModified(filename)
	}

	if pkg.Pkgdir == "." && pkglint.Opts.CheckDistinfo && pkglint.Opts.CheckPatches {
		if havePatches && !haveDistinfo {
			NewLineWhole(pkg.File(pkg.DistinfoFile)).Warnf("File not found. Please run %q.", bmake("makepatchsum"))
		}
	}
}

// Assertf checks that the condition is true. Otherwise it terminates the
// process with a fatal error message, prefixed with "Pkglint internal error".
//
// This method must only be used for programming errors.
// For runtime errors, use dummyLine.Fatalf.
func (pkglint *Pkglint) Assertf(cond bool, format string, args ...interface{}) {
	if !cond {
		panic("Pkglint internal error: " + fmt.Sprintf(format, args...))
	}
}

// diag logs a diagnostic. These are filtered by the --only command line option,
// and duplicates are suppressed unless the --log-verbose command line option is given.
//
// See logf for logging arbitrary messages.
func (pkglint *Pkglint) diag(line Line, level *LogLevel, format string, args []interface{}) {
	if pkglint.Opts.ShowAutofix || pkglint.Opts.Autofix {
		// In these two cases, the only interesting diagnostics are those that can
		// be fixed automatically. These are logged by Autofix.Apply.
		return
	}

	pkglint.explainNext = pkglint.shallBeLogged(format)
	if !pkglint.explainNext {
		return
	}

	if pkglint.Opts.ShowSource {
		line.showSource(pkglint.logOut)
	}
	pkglint.logf(level, line.Filename, line.Linenos(), format, fmt.Sprintf(format, args...))
	if pkglint.Opts.ShowSource {
		pkglint.logOut.Separate()
	}
}

// shallBeLogged tests whether a diagnostic with the given format should
// be logged.
//
// It only inspects the --only arguments; duplicates are handled in main.logf.
func (pkglint *Pkglint) shallBeLogged(format string) bool {
	if len(G.Opts.LogOnly) == 0 {
		return true
	}

	for _, substr := range G.Opts.LogOnly {
		if contains(format, substr) {
			return true
		}
	}
	return false
}

func (pkglint *Pkglint) logf(level *LogLevel, filename, lineno, format, msg string) bool {
	// TODO: Only ever output ASCII, no matter what's in the message.

	if filename != "" {
		filename = cleanpath(filename)
	}
	if pkglint.Testing && format != AutofixFormat && !hasSuffix(format, ": %s") && !hasSuffix(format, ". %s") {
		pkglint.Assertf(hasSuffix(format, "."), "Diagnostic format %q must end in a period.", format)
	}

	// XXX: Allow to override this check, to log arbitrary messages, not only diagnostics; see diag().
	if !pkglint.Opts.LogVerbose && format != AutofixFormat && !pkglint.logged.FirstTime(path.Clean(filename)+":"+lineno+": "+msg) {
		pkglint.explainNext = false
		return false
	}

	var text, sep string
	if !pkglint.Opts.GccOutput {
		text += sep + level.TraditionalName + ":"
		sep = " "
	}
	if filename != "" {
		text += sep + filename
		sep = ": "
		if lineno != "" {
			text += ":" + lineno
		}
	}
	if pkglint.Opts.GccOutput {
		text += sep + level.GccName + ":"
		sep = " "
	}
	if pkglint.Opts.Profiling && format != AutofixFormat && level != Fatal {
		pkglint.loghisto.Add(format, 1)
	}
	text += sep + msg + "\n"

	out := pkglint.logOut
	if level == Fatal {
		out = pkglint.logErr
	}

	out.Write(text)

	switch level {
	case Fatal:
		panic(pkglintFatal{})
	case Error:
		pkglint.errors++
	case Warn:
		pkglint.warnings++
	}
	return true
}

// Returns the pkgsrc top-level directory, relative to the given file or directory.
func findPkgsrcTopdir(filename string) string {
	for _, dir := range [...]string{".", "..", "../..", "../../.."} {
		if fileExists(filename + "/" + dir + "/mk/bsd.pkg.mk") {
			return dir
		}
	}
	return ""
}

func resolveVariableRefs(text string) (resolved string) {
	if !contains(text, "${") {
		return text
	}

	visited := make(map[string]bool) // To prevent endless loops

	replacer := func(m string) string {
		varname := m[2 : len(m)-1]
		if !visited[varname] {
			visited[varname] = true
			if G.Pkg != nil {
				if value, ok := G.Pkg.vars.Value(varname); ok {
					return value
				}
			}
			if G.Mk != nil {
				if value, ok := G.Mk.vars.Value(varname); ok {
					return value
				}
			}
		}
		return "${" + varname + "}"
	}

	str := text
	for {
		replaced := replaceAllFunc(str, `\$\{([\w.]+)\}`, replacer)
		if replaced == str {
			if trace.Tracing && str != text {
				trace.Stepf("resolveVariableRefs %q => %q", text, replaced)
			}
			return replaced
		}
		str = replaced
	}
}

func CheckfileExtra(filename string) {
	if trace.Tracing {
		defer trace.Call1(filename)()
	}

	if lines := Load(filename, NotEmpty|LogErrors); lines != nil {
		ChecklinesTrailingEmptyLines(lines)
	}
}

func ChecklinesDescr(lines Lines) {
	if trace.Tracing {
		defer trace.Call1(lines.FileName)()
	}

	for _, line := range lines.Lines {
		ck := LineChecker{line}
		ck.CheckLength(80)
		ck.CheckTrailingWhitespace()
		ck.CheckValidCharacters()
		if contains(line.Text, "${") {
			line.Notef("Variables are not expanded in the DESCR file.")
		}
	}
	ChecklinesTrailingEmptyLines(lines)

	if maxLines := 24; lines.Len() > maxLines {
		line := lines.Lines[maxLines]

		line.Warnf("File too long (should be no more than %d lines).", maxLines)
		Explain(
			"The DESCR file should fit on a traditional terminal of 80x25",
			"characters.  It is also intended to give a _brief_ summary about",
			"the package's contents.")
	}

	SaveAutofixChanges(lines)
}

func ChecklinesMessage(lines Lines) {
	if trace.Tracing {
		defer trace.Call1(lines.FileName)()
	}

	explanation := []string{
		"A MESSAGE file should consist of a header line, having 75 \"=\"",
		"characters, followed by a line containing only the RCS Id, then an",
		"empty line, your text and finally the footer line, which is the",
		"same as the header line."}

	if lines.Len() < 3 {
		lines.LastLine().Warnf("File too short.")
		Explain(explanation...)
		return
	}

	hline := strings.Repeat("=", 75)
	if line := lines.Lines[0]; line.Text != hline {
		fix := line.Autofix()
		fix.Warnf("Expected a line of exactly 75 \"=\" characters.")
		fix.Explain(explanation...)
		fix.InsertBefore(hline)
		fix.Apply()
		lines.CheckRcsID(0, ``, "")
	} else if 1 < lines.Len() {
		lines.CheckRcsID(1, ``, "")
	}
	for _, line := range lines.Lines {
		ck := LineChecker{line}
		ck.CheckLength(80)
		ck.CheckTrailingWhitespace()
		ck.CheckValidCharacters()
	}
	if lastLine := lines.LastLine(); lastLine.Text != hline {
		fix := lastLine.Autofix()
		fix.Warnf("Expected a line of exactly 75 \"=\" characters.")
		fix.Explain(explanation...)
		fix.InsertAfter(hline)
		fix.Apply()
	}
	ChecklinesTrailingEmptyLines(lines)

	SaveAutofixChanges(lines)
}

func CheckfileMk(filename string) {
	if trace.Tracing {
		defer trace.Call1(filename)()
	}

	mklines := LoadMk(filename, NotEmpty|LogErrors)
	if mklines == nil {
		return
	}

	mklines.Check()
	mklines.SaveAutofixChanges()
}

func (pkglint *Pkglint) Checkfile(filename string) {
	if trace.Tracing {
		defer trace.Call1(filename)()
	}

	basename := path.Base(filename)
	pkgsrcRel := pkglint.Pkgsrc.ToRel(filename)
	depth := strings.Count(pkgsrcRel, "/")

	if depth == 2 && !pkglint.Wip {
		if contains(basename, "README") || contains(basename, "TODO") {
			NewLineWhole(filename).Errorf("Packages in main pkgsrc must not have a %s file.", basename)
			return
		}
	}

	switch {
	case hasPrefix(basename, "work"),
		hasSuffix(basename, "~"),
		hasSuffix(basename, ".orig"),
		hasSuffix(basename, ".rej"),
		contains(basename, "README") && depth == 2,
		contains(basename, "TODO") && depth == 2:
		if pkglint.Opts.Import {
			NewLineWhole(filename).Errorf("Must be cleaned up before committing the package.")
		}
		return
	}

	st, err := os.Lstat(filename)
	if err != nil {
		NewLineWhole(filename).Errorf("Cannot determine file type: %s", err)
		return
	}

	pkglint.checkExecutable(filename, st)
	pkglint.checkMode(filename, st.Mode())
}

// checkMode checks a directory entry based on its filename and its mode
// (regular file, directory, symlink).
func (pkglint *Pkglint) checkMode(filename string, mode os.FileMode) {
	basename := path.Base(filename)
	switch {
	case mode.IsDir():
		switch {
		case basename == "files" || basename == "patches" || isIgnoredFilename(basename):
			// Ok
		case matches(filename, `(?:^|/)files/[^/]*$`):
			// Ok
		case !isEmptyDir(filename):
			NewLineWhole(filename).Warnf("Unknown directory name.")
		}

	case mode&os.ModeSymlink != 0:
		if !hasPrefix(basename, "work") {
			NewLineWhole(filename).Warnf("Unknown symlink name.")
		}

	case !mode.IsRegular():
		NewLineWhole(filename).Errorf("Only files and directories are allowed in pkgsrc.")

	case basename == "ALTERNATIVES":
		if pkglint.Opts.CheckAlternatives {
			CheckfileAlternatives(filename)
		}

	case basename == "buildlink3.mk":
		if pkglint.Opts.CheckBuildlink3 {
			if mklines := LoadMk(filename, NotEmpty|LogErrors); mklines != nil {
				ChecklinesBuildlink3Mk(mklines)
			}
		}

	case hasPrefix(basename, "DESCR"):
		if pkglint.Opts.CheckDescr {
			if lines := Load(filename, NotEmpty|LogErrors); lines != nil {
				ChecklinesDescr(lines)
			}
		}

	case basename == "distinfo":
		if pkglint.Opts.CheckDistinfo {
			if lines := Load(filename, NotEmpty|LogErrors); lines != nil {
				ChecklinesDistinfo(lines)
			}
		}

	case basename == "DEINSTALL" || basename == "INSTALL":
		if pkglint.Opts.CheckInstall {
			CheckfileExtra(filename)
		}

	case hasPrefix(basename, "MESSAGE"):
		if pkglint.Opts.CheckMessage {
			if lines := Load(filename, NotEmpty|LogErrors); lines != nil {
				ChecklinesMessage(lines)
			}
		}

	case basename == "options.mk":
		if pkglint.Opts.CheckOptions {
			if mklines := LoadMk(filename, NotEmpty|LogErrors); mklines != nil {
				ChecklinesOptionsMk(mklines)
			}
		}

	case matches(basename, `^patch-[-A-Za-z0-9_.~+]*[A-Za-z0-9_]$`):
		if pkglint.Opts.CheckPatches {
			if lines := Load(filename, NotEmpty|LogErrors); lines != nil {
				ChecklinesPatch(lines)
			}
		}

	case matches(filename, `(?:^|/)patches/manual[^/]*$`):
		if trace.Tracing {
			trace.Step1("Unchecked file %q.", filename)
		}

	case matches(filename, `(?:^|/)patches/[^/]*$`):
		NewLineWhole(filename).Warnf("Patch files should be named \"patch-\", followed by letters, '-', '_', '.', and digits only.")

	case matches(basename, `^(?:.*\.mk|Makefile.*)$`) && !matches(filename, `files/`) && !matches(filename, `patches/`):
		if pkglint.Opts.CheckMk {
			CheckfileMk(filename)
		}

	case hasPrefix(basename, "PLIST"):
		if pkglint.Opts.CheckPlist {
			if lines := Load(filename, NotEmpty|LogErrors); lines != nil {
				ChecklinesPlist(lines)
			}
		}

	case hasPrefix(basename, "CHANGES-"):
		// This only checks the file but doesn't register the changes globally.
		_ = pkglint.Pkgsrc.loadDocChangesFromFile(filename)

	case matches(filename, `(?:^|/)files/[^/]*$`):
		// Skip

	case basename == "spec":
		if !hasPrefix(pkglint.Pkgsrc.ToRel(filename), "regress/") {
			NewLineWhole(filename).Warnf("Only packages in regress/ may have spec files.")
		}

	default:
		NewLineWhole(filename).Warnf("Unexpected file found.")
		if pkglint.Opts.CheckExtra {
			CheckfileExtra(filename)
		}
	}
}

func (pkglint *Pkglint) checkExecutable(filename string, st os.FileInfo) {
	switch {
	case !st.Mode().IsRegular():
		// Directories and other entries may be executable.

	case st.Mode().Perm()&0111 == 0:
		// Good.

	case isCommitted(filename):
		// Too late to be fixed by the package developer, since
		// CVS remembers the executable bit in the repo file.
		// At this point, it can only be reset by the CVS admins.

	default:
		line := NewLineWhole(filename)
		fix := line.Autofix()
		fix.Warnf("Should not be executable.")
		fix.Explain(
			"No package file should ever be executable.  Even the INSTALL and",
			"DEINSTALL scripts are usually not usable in the form they have in",
			"the package, as the pathnames get adjusted during installation.",
			"So there is no need to have any file executable.")
		fix.Custom(func(showAutofix, autofix bool) {
			fix.Describef(0, "Clearing executable bits")
			if autofix {
				if err := os.Chmod(filename, st.Mode()&^0111); err != nil {
					line.Errorf("Cannot clear executable bits: %s", err)
				}
			}
		})
		fix.Apply()
	}
}

func ChecklinesTrailingEmptyLines(lines Lines) {
	// XXX: Maybe move to LinesChecker if there are enough similar functions.

	max := lines.Len()
	last := max
	for last > 1 && lines.Lines[last-1].Text == "" {
		last--
	}
	if last != max {
		lines.Lines[last].Notef("Trailing empty lines.")
	}
}

// Tool returns the tool definition from the closest scope (file, global), or nil.
// The command can be "sed" or "gsed" or "${SED}".
// If a tool is returned, usable tells whether that tool has been added
// to USE_TOOLS in the current scope.
func (pkglint *Pkglint) Tool(command string, time ToolTime) (tool *Tool, usable bool) {
	varname := ""
	if m, toolVarname := match1(command, `^\$\{(\w+)\}$`); m {
		varname = toolVarname
	}

	tools := pkglint.tools()

	if t := tools.ByName(command); t != nil {
		if tools.Usable(t, time) {
			return t, true
		}
		tool = t
	}

	if t := tools.ByVarname(varname); t != nil {
		if tools.Usable(t, time) {
			return t, true
		}
		if tool == nil {
			tool = t
		}
	}
	return
}

// ToolByVarname looks up the tool by its variable name, e.g. "SED".
//
// The returned tool may come either from the current Makefile or the
// current package. It is not guaranteed to be usable; that must be
// checked by the calling code.
func (pkglint *Pkglint) ToolByVarname(varname string, time ToolTime) *Tool {
	return pkglint.tools().ByVarname(varname)
}

func (pkglint *Pkglint) tools() *Tools {
	if pkglint.Mk != nil {
		return pkglint.Mk.Tools
	} else {
		return pkglint.Pkgsrc.Tools
	}
}
