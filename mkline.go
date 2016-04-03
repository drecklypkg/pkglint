package main

// Checks concerning single lines in Makefiles.

import (
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"
)

type MkLine struct {
	Line *Line

	xtype      uint8
	xmustexist bool
	xop        MkOperator
	xvalign    string
	xs1        string
	xs2        string
	xs3        string
	xvalue     string
	xcomment   string
	mkwords    []string
}

func (mkline *MkLine) Error1(format, arg1 string)      { mkline.Line.Error1(format, arg1) }
func (mkline *MkLine) Warn0(format string)             { mkline.Line.Warn0(format) }
func (mkline *MkLine) Warn1(format, arg1 string)       { mkline.Line.Warn1(format, arg1) }
func (mkline *MkLine) Warn2(format, arg1, arg2 string) { mkline.Line.Warn2(format, arg1, arg2) }
func (mkline *MkLine) Note0(format string)             { mkline.Line.Note0(format) }
func (mkline *MkLine) Note2(format, arg1, arg2 string) { mkline.Line.Note2(format, arg1, arg2) }

func NewMkLine(line *Line) (mkline *MkLine) {
	mkline = &MkLine{Line: line}

	text := line.Text

	if hasPrefix(text, " ") {
		mkline.Warn0("Makefile lines should not start with space characters.")
		Explain3(
			"If you want this line to contain a shell program, use a tab",
			"character for indentation.  Otherwise please remove the leading",
			"white-space.")
	}

	if m, varname, op, valueAlign, value, comment := MatchVarassign(text); m {
		value = strings.Replace(value, "\\#", "#", -1)
		varparam := varnameParam(varname)

		mkline.xtype = 1
		mkline.xs1 = varname
		mkline.xs2 = varnameCanon(varname)
		mkline.xs3 = varparam
		mkline.xop = NewMkOperator(op)
		mkline.xvalign = valueAlign
		mkline.xvalue = value
		mkline.xcomment = comment
		mkline.Tokenize(value)
		mkline.mkwords, _ = splitIntoShellWords(mkline.Line, value)
		return
	}

	if hasPrefix(text, "\t") {
		mkline.xtype = 2
		mkline.xs1 = text[1:]
		mkline.Tokenize(mkline.xs1)
		return
	}

	if index := strings.IndexByte(text, '#'); index != -1 && strings.TrimSpace(text[:index]) == "" {
		mkline.xtype = 3
		mkline.xcomment = text[index+1:]
		return
	}

	if strings.TrimSpace(text) == "" {
		mkline.xtype = 4
		return
	}

	if m, indent, directive, args := matchMkCond(text); m {
		mkline.xtype = 5
		mkline.xs1 = indent
		mkline.xs2 = directive
		mkline.xs3 = args
		return
	}

	if m, directive, includefile := match2(text, reMkInclude); m {
		mkline.xtype = 6
		mkline.xmustexist = directive == "include"
		mkline.xs1 = includefile
		return
	}

	if m, directive, includefile := match2(text, `^\.\s*(s?include)\s+<([^>]+)>\s*(?:#.*)?$`); m {
		mkline.xtype = 7
		mkline.xmustexist = directive == "include"
		mkline.xs1 = includefile
		return
	}

	if m, targets, whitespace, sources := match3(text, `^([^\s:]+(?:\s*[^\s:]+)*)(\s*):\s*([^#]*?)(?:\s*#.*)?$`); m {
		mkline.xtype = 8
		mkline.xs1 = targets
		mkline.xs2 = sources
		if whitespace != "" {
			line.Warn0("Space before colon in dependency line.")
		}
		return
	}

	if matches(text, `^(<<<<<<<|=======|>>>>>>>)`) {
		return
	}

	line.Error0("Unknown Makefile line format.")
	return mkline
}

func (mkline *MkLine) String() string {
	return fmt.Sprintf("%s:%s", mkline.Line.Fname, mkline.Line.linenos())
}
func (mkline *MkLine) IsVarassign() bool   { return mkline.xtype == 1 }
func (mkline *MkLine) Varname() string     { return mkline.xs1 }
func (mkline *MkLine) Varcanon() string    { return mkline.xs2 }
func (mkline *MkLine) Varparam() string    { return mkline.xs3 }
func (mkline *MkLine) Op() MkOperator      { return mkline.xop }
func (mkline *MkLine) ValueAlign() string  { return mkline.xvalign }
func (mkline *MkLine) Value() string       { return mkline.xvalue }
func (mkline *MkLine) Comment() string     { return mkline.xcomment }
func (mkline *MkLine) IsShellcmd() bool    { return mkline.xtype == 2 }
func (mkline *MkLine) Shellcmd() string    { return mkline.xs1 }
func (mkline *MkLine) IsComment() bool     { return mkline.xtype == 3 }
func (mkline *MkLine) IsEmpty() bool       { return mkline.xtype == 4 }
func (mkline *MkLine) IsCond() bool        { return mkline.xtype == 5 }
func (mkline *MkLine) Indent() string      { return mkline.xs1 }
func (mkline *MkLine) Directive() string   { return mkline.xs2 }
func (mkline *MkLine) Args() string        { return mkline.xs3 }
func (mkline *MkLine) IsInclude() bool     { return mkline.xtype == 6 }
func (mkline *MkLine) MustExist() bool     { return mkline.xmustexist }
func (mkline *MkLine) Includefile() string { return mkline.xs1 }
func (mkline *MkLine) IsSysinclude() bool  { return mkline.xtype == 7 }
func (mkline *MkLine) IsDependency() bool  { return mkline.xtype == 8 }
func (mkline *MkLine) Targets() string     { return mkline.xs1 }
func (mkline *MkLine) Sources() string     { return mkline.xs2 }

func (mkline *MkLine) Check() {
	mkline.Line.CheckTrailingWhitespace()
	mkline.Line.CheckValidCharacters(`[\t -~]`)

	switch {
	case mkline.IsVarassign():
		mkline.checkVarassign()

	case mkline.IsShellcmd():
		shellcmd := mkline.Shellcmd()
		mkline.checkText(shellcmd)
		NewShellLine(mkline).CheckShellCommandLine(shellcmd)

	case mkline.IsComment():
		if hasPrefix(mkline.Line.Text, "# url2pkg-marker") {
			mkline.Line.Error0("This comment indicates unfinished work (url2pkg).")
		}

	case mkline.IsInclude():
		mkline.checkInclude()
	}
}

func (mkline *MkLine) checkInclude() {
	if G.opts.Debug {
		defer tracecall0()()
	}

	includefile := mkline.Includefile()
	mustExist := mkline.MustExist()
	if G.opts.Debug {
		traceStep1("includefile=%s", includefile)
	}
	mkline.CheckRelativePath(includefile, mustExist)

	switch {
	case hasSuffix(includefile, "/Makefile"):
		mkline.Line.Error0("Other Makefiles must not be included directly.")
		Explain4(
			"If you want to include portions of another Makefile, extract",
			"the common parts and put them into a Makefile.common.  After",
			"that, both this one and the other package should include the",
			"Makefile.common.")

	case includefile == "../../mk/bsd.prefs.mk":
		if path.Base(mkline.Line.Fname) == "buildlink3.mk" {
			mkline.Note0("For efficiency reasons, please include bsd.fast.prefs.mk instead of bsd.prefs.mk.")
		}
		if G.Pkg != nil {
			G.Pkg.SeenBsdPrefsMk = true
		}

	case includefile == "../../mk/bsd.fast.prefs.mk":
		if G.Pkg != nil {
			G.Pkg.SeenBsdPrefsMk = true
		}

	case hasSuffix(includefile, "/x11-links/buildlink3.mk"):
		mkline.Error1("%s must not be included directly. Include \"../../mk/x11.buildlink3.mk\" instead.", includefile)

	case hasSuffix(includefile, "/jpeg/buildlink3.mk"):
		mkline.Error1("%s must not be included directly. Include \"../../mk/jpeg.buildlink3.mk\" instead.", includefile)

	case hasSuffix(includefile, "/intltool/buildlink3.mk"):
		mkline.Warn0("Please write \"USE_TOOLS+= intltool\" instead of this line.")

	case hasSuffix(includefile, "/builtin.mk"):
		mkline.Line.Error2("%s must not be included directly. Include \"%s/buildlink3.mk\" instead.", includefile, path.Dir(includefile))
	}
}

func (mkline *MkLine) checkCond(indentation *Indentation, forVars map[string]bool) {
	indent, directive, args := mkline.Indent(), mkline.Directive(), mkline.Args()

	switch directive {
	case "endif", "endfor", "elif", "else":
		if indentation.Len() > 1 {
			indentation.Pop()
		} else {
			mkline.Error1("Unmatched .%s.", directive)
		}
	}

	// Check the indentation
	if expected := strings.Repeat(" ", indentation.Depth()); indent != expected {
		if G.opts.WarnSpace && !mkline.Line.AutofixReplace("."+indent, "."+expected) {
			mkline.Line.Notef("This directive should be indented by %d spaces.", indentation.Depth())
		}
	}

	if directive == "if" && matches(args, `^!defined\([\w]+_MK\)$`) {
		indentation.Push(indentation.Depth())

	} else if matches(directive, `^(?:if|ifdef|ifndef|for|elif|else)$`) {
		indentation.Push(indentation.Depth() + 2)
	}

	reDirectivesWithArgs := `^(?:if|ifdef|ifndef|elif|for|undef)$`
	if matches(directive, reDirectivesWithArgs) && args == "" {
		mkline.Error1("\".%s\" requires arguments.", directive)

	} else if !matches(directive, reDirectivesWithArgs) && args != "" {
		mkline.Error1("\".%s\" does not take arguments.", directive)

		if directive == "else" {
			mkline.Note0("If you meant \"else if\", use \".elif\".")
		}

	} else if directive == "if" || directive == "elif" {
		mkline.CheckCond()

	} else if directive == "ifdef" || directive == "ifndef" {
		if matches(args, `\s`) {
			mkline.Error1("The \".%s\" directive can only handle _one_ argument.", directive)
		} else {
			mkline.Line.Warnf("The \".%s\" directive is deprecated. Please use \".if %sdefined(%s)\" instead.",
				directive, ifelseStr(directive == "ifdef", "", "!"), args)
		}

	} else if directive == "for" {
		if m, vars, values := match2(args, `^(\S+(?:\s*\S+)*?)\s+in\s+(.*)$`); m {
			for _, forvar := range splitOnSpace(vars) {
				if !G.Infrastructure && hasPrefix(forvar, "_") {
					mkline.Warn1("Variable names starting with an underscore (%s) are reserved for internal pkgsrc use.", forvar)
				}

				if matches(forvar, `^[_a-z][_a-z0-9]*$`) {
					// Fine.
				} else if matches(forvar, `[A-Z]`) {
					mkline.Warn0(".for variable names should not contain uppercase letters.")
				} else {
					mkline.Error1("Invalid variable name %q.", forvar)
				}

				forVars[forvar] = true
			}

			// Check if any of the value's types is not guessed.
			guessed := true
			for _, value := range splitOnSpace(values) {
				if m, vname := match1(value, `^\$\{(.*)\}`); m {
					vartype := mkline.getVariableType(vname)
					if vartype != nil && !vartype.guessed {
						guessed = false
					}
				}
			}

			forLoopType := &Vartype{lkSpace, CheckvarUnchecked, []AclEntry{{"*", aclpAllRead}}, guessed}
			forLoopContext := &VarUseContext{forLoopType, vucTimeParse, vucQuotFor, vucExtentWord}
			for _, forLoopVar := range mkline.extractUsedVariables(values) {
				mkline.CheckVaruse(&MkVarUse{forLoopVar, nil}, forLoopContext)
			}
		}

	} else if directive == "undef" && args != "" {
		for _, uvar := range splitOnSpace(args) {
			if forVars[uvar] {
				mkline.Note0("Using \".undef\" after a \".for\" loop is unnecessary.")
			}
		}
	}
}

func (mkline *MkLine) checkDependencyRule(allowedTargets map[string]bool) {
	targets := splitOnSpace(mkline.Targets())
	sources := splitOnSpace(mkline.Sources())

	for _, source := range sources {
		if source == ".PHONY" {
			for _, target := range targets {
				allowedTargets[target] = true
			}
		}
	}

	for _, target := range targets {
		if target == ".PHONY" {
			for _, dep := range sources {
				allowedTargets[dep] = true
			}

		} else if target == ".ORDER" {
			// TODO: Check for spelling mistakes.

		} else if !allowedTargets[target] {
			mkline.Warn1("Unusual target %q.", target)
			Explain3(
				"If you want to define your own targets, you can \"declare\"",
				"them by inserting a \".PHONY: my-target\" line before this line.  This",
				"will tell make(1) to not interpret this target's name as a filename.")
		}
	}
}

func (mkline *MkLine) Tokenize(s string) []*MkToken {
	p := NewParser(mkline.Line, s)
	tokens := p.MkTokens()
	if p.Rest() != "" {
		mkline.Error1("Invalid Makefile syntax at %q.", p.Rest())
	}
	return tokens
}

func (mkline *MkLine) checkVarassignDefPermissions(varname string, op MkOperator) {
	if !G.opts.WarnPerm {
		return
	}
	if G.opts.Debug {
		defer tracecall(varname, op)()
	}

	vartype := mkline.getVariableType(varname)
	if vartype == nil {
		if G.opts.Debug {
			traceStep1("No type definition found for %q.", varname)
		}
		return
	}

	perms := vartype.EffectivePermissions(mkline.Line.Fname)
	var needed AclPermissions
	switch op {
	case opAssign, opAssignShell, opAssignEval:
		needed = aclpSet
	case opAssignDefault:
		needed = aclpSetDefault
	case opAssignAppend:
		needed = aclpAppend
	}

	switch {
	case perms.Contains(needed):
		break
	case perms == aclpUnknown:
		if G.opts.Debug {
			traceStep1("Unknown permissions for %q.", varname)
		}
	default:
		alternativeActions := perms & aclpAllWrite
		alternativeFiles := vartype.AllowedFiles(needed)
		switch {
		case alternativeActions != 0 && alternativeFiles != "":
			mkline.Line.Warnf("The variable %s may not be %s (only %s) in this file; it would be ok in %s.",
				varname, needed.HumanString(), alternativeActions.HumanString(), alternativeFiles)
		case alternativeFiles != "":
			mkline.Line.Warnf("The variable %s may not be %s in this file; it would be ok in %s.",
				varname, needed.HumanString(), alternativeFiles)
		case alternativeActions != 0:
			mkline.Line.Warnf("The variable %s may not be %s (only %s) in this file.",
				varname, needed.HumanString(), alternativeActions.HumanString())
		default:
			mkline.Line.Warnf("The variable %s may not be %s by any package.",
				varname, needed.HumanString())
		}
		Explain4(
			"The allowed actions for a variable are determined based on the file",
			"name in which the variable is used or defined.  The exact rules are",
			"hard-coded into pkglint.  If they seem to be incorrect, please ask",
			"on the tech-pkg@NetBSD.org mailing list.")
	}
}

func (mkline *MkLine) CheckVaruse(varuse *MkVarUse, vuc *VarUseContext) {
	if G.opts.Debug {
		defer tracecall(mkline, varuse, vuc)()
	}

	varname := varuse.varname
	vartype := mkline.getVariableType(varname)
	if G.opts.WarnExtra &&
		(vartype == nil || vartype.guessed) &&
		!varIsUsed(varname) &&
		!(G.Mk != nil && G.Mk.forVars[varname]) {
		mkline.Warn1("%s is used but not defined. Spelling mistake?", varname)
	}

	mkline.CheckVarusePermissions(varname, vuc)

	if varname == "LOCALBASE" && !G.Infrastructure {
		mkline.WarnVaruseLocalbase()
	}

	needsQuoting := mkline.variableNeedsQuoting(varname, vartype, vuc)

	if vuc.quoting == vucQuotFor {
		mkline.checkVaruseFor(varname, vartype, needsQuoting)
	}

	if G.opts.WarnQuoting && vuc.quoting != vucQuotUnknown && needsQuoting != nqDontKnow {
		mkline.CheckVaruseShellword(varname, vartype, vuc, varuse.Mod(), needsQuoting)
	}

	if G.globalData.UserDefinedVars[varname] != nil && !G.globalData.SystemBuildDefs[varname] && !G.Mk.buildDefs[varname] {
		mkline.Warn1("The user-defined variable %s is used but not added to BUILD_DEFS.", varname)
		Explain(
			"When a pkgsrc package is built, many things can be configured by the",
			"pkgsrc user in the mk.conf file.  All these configurations should be",
			"recorded in the binary package, so the package can be reliably",
			"rebuilt.  The BUILD_DEFS variable contains a list of all these",
			"user-settable variables, so please add your variable to it, too.")
	}
}

func (mkline *MkLine) CheckVarusePermissions(varname string, vuc *VarUseContext) {
	if !G.opts.WarnPerm {
		return
	}
	if G.opts.Debug {
		defer tracecall(varname, vuc)()
	}

	// This is the type of the variable that is being used. Not to
	// be confused with vuc.vartype, which is the type of the
	// context in which the variable is used (often a ShellCommand
	// or, in an assignment, the type of the left hand side variable).
	vartype := mkline.getVariableType(varname)
	if vartype == nil {
		if G.opts.Debug {
			traceStep1("No type definition found for %q.", varname)
		}
		return
	}

	perms := vartype.EffectivePermissions(mkline.Line.Fname)

	isLoadTime := false // Will the variable be used at load time?

	// Might the variable be used indirectly at load time, for example
	// by assigning it to another variable which then gets evaluated?
	isIndirect := false

	switch {
	case vuc.vartype != nil && vuc.vartype.guessed:
		// Don't warn about unknown variables.

	case vuc.time == vucTimeParse && !perms.Contains(aclpUseLoadtime):
		isLoadTime = true

	case vuc.vartype != nil && vuc.vartype.Union().Contains(aclpUseLoadtime) && !perms.Contains(aclpUseLoadtime):
		isLoadTime = true
		isIndirect = true
	}

	if isLoadTime && !isIndirect {
		mkline.Warn1("%s should not be evaluated at load time.", varname)
		Explain(
			"Many variables, especially lists of something, get their values",
			"incrementally.  Therefore it is generally unsafe to rely on their",
			"value until it is clear that it will never change again.  This",
			"point is reached when the whole package Makefile is loaded and",
			"execution of the shell commands starts, in some cases earlier.",
			"",
			"Additionally, when using the \":=\" operator, each $$ is replaced",
			"with a single $, so variables that have references to shell",
			"variables or regular expressions are modified in a subtle way.")
	}

	if isLoadTime && isIndirect {
		mkline.Warn1("%s should not be evaluated indirectly at load time.", varname)
		Explain4(
			"The variable on the left-hand side may be evaluated at load time,",
			"but the variable on the right-hand side may not.  Because of the",
			"assignment in this line, the variable might be used indirectly",
			"at load time, before it is guaranteed to be properly initialized.")
	}

	if !perms.Contains(aclpUseLoadtime) && !perms.Contains(aclpUse) {
		mkline.Warn1("%s may not be used in this file.", varname)
	}
}

func (mkline *MkLine) WarnVaruseLocalbase() {
	mkline.Warn0("The LOCALBASE variable should not be used by packages.")
	Explain(
		// from jlam via private mail.
		"Currently, LOCALBASE is typically used in these cases:",
		"",
		"(1) To locate a file or directory from another package.",
		"(2) To refer to own files after installation.",
		"",
		"Example for (1):",
		"",
		"	STRLIST=        ${LOCALBASE}/bin/strlist",
		"	do-build:",
		"		cd ${WRKSRC} && ${STRLIST} *.str",
		"",
		"This should better be:",
		"",
		"	EVAL_PREFIX=    STRLIST_PREFIX=strlist",
		"	STRLIST=        ${STRLIST_PREFIX}/bin/strlist",
		"	do-build:",
		"		cd ${WRKSRC} && ${STRLIST} *.str",
		"",
		"Example for (2):",
		"",
		"	CONFIGURE_ENV+= --with-datafiles=${LOCALBASE}/share/pkgbase",
		"",
		"This should better be:",
		"",
		"	CONFIGURE_ENV+= --with-datafiles=${PREFIX}/share/pkgbase")
}

func (mkline *MkLine) checkVaruseFor(varname string, vartype *Vartype, needsQuoting NeedsQuoting) {
	if G.opts.Debug {
		defer tracecall(varname, vartype, needsQuoting)()
	}

	if false && // Too many false positives
		vartype != nil &&
		vartype.kindOfList != lkSpace &&
		needsQuoting != nqDoesntMatter {
		mkline.Warn1("The variable %s should not be used in .for loops.", varname)
		Explain4(
			"The .for loop splits its argument at sequences of white-space, as",
			"opposed to all other places in make(1), which act like the shell.",
			"Therefore only variables that are split at whitespace or don't",
			"contain any special characters should be used here.")
	}
}

func (mkline *MkLine) CheckVaruseShellword(varname string, vartype *Vartype, vuc *VarUseContext, mod string, needsQuoting NeedsQuoting) {
	if G.opts.Debug {
		defer tracecall(varname, vartype, vuc, mod, needsQuoting)()
	}

	// In GNU configure scripts, a few variables need to be
	// passed through the :M* operator before they reach the
	// configure scripts.
	//
	// When doing checks outside a package, the :M* operator is needed for safety.
	needMstar := matches(varname, `^(?:.*_)?(?:CFLAGS||CPPFLAGS|CXXFLAGS|FFLAGS|LDFLAGS|LIBS)$`) &&
		(G.Pkg == nil || G.Pkg.vardef["GNU_CONFIGURE"] != nil)

	strippedMod := mod
	if m, stripped := match1(mod, `(.*?)(?::M\*)?(?::Q)?$`); m {
		strippedMod = stripped
	}

	if mod == ":M*:Q" && !needMstar {
		mkline.Line.Note0("The :M* modifier is not needed here.")

	} else if needsQuoting == nqYes {
		correctMod := strippedMod + ifelseStr(needMstar, ":M*:Q", ":Q")
		if mod != correctMod {
			if vuc.quoting == vucQuotPlain {
				if !mkline.Line.AutofixReplace("${"+varname+mod+"}", "${"+varname+correctMod+"}") {
					mkline.Line.Warnf("Please use ${%s%s} instead of ${%s%s}.", varname, correctMod, varname, mod)
				}
			} else {
				mkline.Line.Warnf("Please use ${%s%s} instead of ${%s%s} and make sure"+
					" the variable appears outside of any quoting characters.", varname, correctMod, varname, mod)
			}
			Explain1(
				"See the pkgsrc guide, section \"quoting guideline\", for details.")

		} else if vuc.quoting != vucQuotPlain {
			mkline.Line.Warnf("Please move ${%s%s} outside of any quoting characters.", varname, mod)
			Explain(
				"The :Q modifier only works reliably when it is used without any",
				"quoting characters.",
				"",
				"Works often, but not always: CFLAGS=\"${CFLAGS:Q}\"",
				"Works almost always: CFLAGS='${CFLAGS:Q}'",
				"Works always: CFLAGS=${CFLAGS:Q}")
		}
	}

	if hasSuffix(mod, ":Q") && (needsQuoting == nqNo || needsQuoting == nqDoesntMatter) {
		bad := "${" + varname + mod + "}"
		good := "${" + varname + strings.TrimSuffix(mod, ":Q") + "}"
		needExplain := false
		if needsQuoting == nqNo && !mkline.Line.AutofixReplace(bad, good) {
			needExplain = true
			mkline.Warn1("The :Q operator should not be used for ${%s} here.", varname)
		}
		if needsQuoting == nqDoesntMatter && !mkline.Line.AutofixReplace(bad, good) {
			needExplain = true
			mkline.Line.Note1("The :Q operator isn't necessary for ${%s} here.", varname)
		}
		if needExplain {
			Explain(
				"Many variables in pkgsrc do not need the :Q operator, since they",
				"are not expected to contain white-space or other special characters.",
				"Examples for these \"safe\" variables are:",
				"",
				"\t* filenames",
				"\t* directory names",
				"\t* user and group names",
				"\t* tool names and tool paths",
				"\t* variable names",
				"\t* PKGNAME")
		}
	}
}

func (mkline *MkLine) checkVarassignPythonVersions(varname, value string) {
	if G.opts.Debug {
		defer tracecall2(varname, value)()
	}

	strversions := splitOnSpace(value)
	intversions := make([]int, len(strversions))
	for i, strversion := range strversions {
		iver, err := strconv.Atoi(strversion)
		if err != nil || !(iver > 0) {
			mkline.Error1("All values for %s must be positive integers.", varname)
			return
		}
		intversions[i] = iver
	}

	for i, ver := range intversions {
		if i > 0 && ver >= intversions[i-1] {
			mkline.Warn1("The values for %s should be in decreasing order.", varname)
			Explain2(
				"If they aren't, it may be possible that needless versions of",
				"packages are installed.")
		}
	}
}

func (mkline *MkLine) checkVarassign() {
	if G.opts.Debug {
		defer tracecall0()()
	}

	varname := mkline.Varname()
	op := mkline.Op()
	value := mkline.Value()
	comment := mkline.Comment()
	varcanon := varnameCanon(varname)

	defineVar(mkline, varname)
	mkline.checkVarassignDefPermissions(varname, op)
	mkline.checkVarassignBsdPrefs()

	mkline.checkText(value)
	mkline.CheckVartype(varname, op, value, comment)

	// If the variable is not used and is untyped, it may be a spelling mistake.
	if op == opAssignEval && varname == strings.ToLower(varname) {
		if G.opts.Debug {
			traceStep1("%s might be unused unless it is an argument to a procedure file.", varname)
		}

	} else if !varIsUsed(varname) {
		if vartypes := G.globalData.vartypes; vartypes[varname] != nil || vartypes[varcanon] != nil {
			// Ok
		} else if deprecated := G.globalData.Deprecated; deprecated[varname] != "" || deprecated[varcanon] != "" {
			// Ok
		} else {
			mkline.Warn1("%s is defined but not used. Spelling mistake?", varname)
		}
	}

	mkline.checkVarassignSpecific()

	if varname == "EVAL_PREFIX" {
		if m, evalVarname := match1(value, `^([\w_]+)=`); m {

			// The variables mentioned in EVAL_PREFIX will later be
			// defined by find-prefix.mk. Therefore, they are marked
			// as known in the current file.
			G.Mk.vardef[evalVarname] = mkline
		}
	}

	if fix := G.globalData.Deprecated[varname]; fix != "" {
		mkline.Warn2("Definition of %s is deprecated. %s", varname, fix)
	} else if fix := G.globalData.Deprecated[varcanon]; fix != "" {
		mkline.Warn2("Definition of %s is deprecated. %s", varname, fix)
	}

	mkline.checkVarassignPlistComment(varname, value)
	mkline.checkVarassignVaruse(varname, op)
}

func (mkline *MkLine) checkVarassignVaruse(varname string, op MkOperator) {
	if G.opts.Debug {
		defer tracecall(varname, op)()
	}

	time := vucTimeRun
	if op == opAssignEval || op == opAssignShell {
		time = vucTimeParse
	}

	vartype := mkline.getVariableType(varname)
	if op == opAssignShell {
		vartype = shellcommandsContextType
	}

	words := mkline.mkwords
	if vartype != nil && vartype.IsShell() {
		words, _ = splitIntoShellTokens(mkline.Line, mkline.Value())
	}

	for _, word := range words {
		if contains(word, "${") {
			p := NewParser(mkline.Line, word)
			mark := p.repl.Mark()
			extent := vucExtentWordpart
			if p.VarUse() != nil && p.EOF() {
				extent = vucExtentWord
			}
			p.repl.Reset(mark)

			quoting := NewShQuote("")
			for _, token := range p.MkTokens() {
				if token.literal != "" {
					quoting.Feed(token.literal)
				} else {
					vuc := &VarUseContext{vartype, time, quoting.State.ToVarUseContext(), extent}
					mkline.CheckVaruse(&token.varuse, vuc)
				}
			}
		}
	}
}

func (mkline *MkLine) checkVarassignSpecific() {
	varname := mkline.Varname()
	value := mkline.Value()

	if contains(value, "/etc/rc.d") {
		mkline.Warn0("Please use the RCD_SCRIPTS mechanism to install rc.d scripts automatically to ${RCD_SCRIPTS_EXAMPLEDIR}.")
	}

	if hasPrefix(varname, "_") && !G.Infrastructure {
		mkline.Warn1("Variable names starting with an underscore (%s) are reserved for internal pkgsrc use.", varname)
	}

	if varname == "PERL5_PACKLIST" && G.Pkg != nil {
		if m, p5pkgname := match1(G.Pkg.EffectivePkgbase, `^p5-(.*)`); m {
			guess := "auto/" + strings.Replace(p5pkgname, "-", "/", -1) + "/.packlist"

			ucvalue, ucguess := strings.ToUpper(value), strings.ToUpper(guess)
			if ucvalue != ucguess && ucvalue != "${PERL5_SITEARCH}/"+ucguess {
				mkline.Warn1("Unusual value for PERL5_PACKLIST -- %q expected.", guess)
			}
		}
	}

	if varname == "CONFIGURE_ARGS" && contains(value, "=${PREFIX}/share/kde") {
		mkline.Note0("Please .include \"../../meta-pkgs/kde3/kde3.mk\" instead of this line.")
		Explain3(
			"That file does many things automatically and consistently that this",
			"package also does.  When using kde3.mk, you can probably also leave",
			"out some explicit dependencies.")
	}

	if varname == "PYTHON_VERSIONS_ACCEPTED" {
		mkline.checkVarassignPythonVersions(varname, value)
	}

	if mkline.Comment() == "# defined" && !hasSuffix(varname, "_MK") && !hasSuffix(varname, "_COMMON") {
		mkline.Note0("Please use \"# empty\", \"# none\" or \"yes\" instead of \"# defined\".")
		Explain(
			"The value #defined says something about the state of the variable,",
			"but not what that _means_.  In some cases a variable that is defined",
			"means \"yes\", in other cases it is an empty list (which is also",
			"only the state of the variable), whose meaning could be described",
			"with \"none\".  It is this meaning that should be described.")
	}

	if m, revvarname := match1(value, `\$\{(PKGNAME|PKGVERSION)[:\}]`); m {
		if varname == "DIST_SUBDIR" || varname == "WRKSRC" {
			mkline.Line.Warnf("%s should not be used in %s, as it includes the PKGREVISION. Please use %s_NOREV instead.", revvarname, varname, revvarname)
		}
	}

	if hasPrefix(varname, "SITES_") {
		mkline.Warn0("SITES_* is deprecated. Please use SITES.* instead.")
	}
}

func (mkline *MkLine) checkVarassignBsdPrefs() {
	if G.opts.WarnExtra && mkline.Op() == opAssignDefault && G.Pkg != nil && !G.Pkg.SeenBsdPrefsMk {
		switch mkline.Varcanon() {
		case "BUILDLINK_PKGSRCDIR.*", "BUILDLINK_DEPMETHOD.*", "BUILDLINK_ABI_DEPENDS.*":
			return
		}

		mkline.Warn0("Please include \"../../mk/bsd.prefs.mk\" before using \"?=\".")
		Explain(
			"The ?= operator is used to provide a default value to a variable.",
			"In pkgsrc, many variables can be set by the pkgsrc user in the",
			"mk.conf file.  This file must be included explicitly.  If a ?=",
			"operator appears before mk.conf has been included, it will not care",
			"about the user's preferences, which can result in unexpected",
			"behavior.",
			"",
			"The easiest way to include the mk.conf file is by including the",
			"bsd.prefs.mk file, which will take care of everything.")
	}
}

func (mkline *MkLine) checkVarassignPlistComment(varname, value string) {
	if false && // This is currently neither correct nor helpful
		contains(value, "@comment") && !matches(value, `="@comment "`) {
		mkline.Warn1("Please don't use @comment in %s.", varname)
		Explain(
			"If you are defining a PLIST conditional here, use one of the",
			"following patterns instead:",
			"",
			"1. The direct way, without intermediate variable",
			"",
			"\tPLIST_SUBST+=\tMY_VAR=\"@comment \"",
			"",
			"2. The indirect way, with a separate variable",
			"",
			"\tPLIST_VARS+=\tMY_VAR",
			"\t.if ...",
			"\tMY_VAR?=\tyes",
			"\t.endif")
	}

	// Mark the variable as PLIST condition. This is later used in checkfile_PLIST.
	if G.Pkg != nil {
		if m, plistVarname := match1(value, `(.+)=.*@comment.*`); m {
			G.Pkg.plistSubstCond[plistVarname] = true
		}
	}
}

const reVarnamePlural = `^(?:` +
	`.*[Ss]` +
	`|.*LIST` +
	`|.*_AWK` +
	`|.*_ENV` +
	`|.*_OVERRIDE` +
	`|.*_PREREQ` +
	`|.*_REQD` +
	`|.*_SED` +
	`|.*_SKIP` +
	`|.*_SRC` +
	`|.*_SUBST` +
	`|.*_TARGET` +
	`|.*_TMPL` +
	`|BROKEN_EXCEPT_ON_PLATFORM` +
	`|BROKEN_ON_PLATFORM` +
	`|BUILDLINK_DEPMETHOD` +
	`|BUILDLINK_LDADD` +
	`|BUILDLINK_TRANSFORM` +
	`|COMMENT` +
	`|CRYPTO` +
	`|DEINSTALL_TEMPLATE` +
	`|EVAL_PREFIX` +
	`|EXTRACT_ONLY` +
	`|FETCH_MESSAGE` +
	`|FIX_RPATH` +
	`|GENERATE_PLIST` +
	`|INSTALL_TEMPLATE` +
	`|INTERACTIVE_STAGE` +
	`|LICENSE` +
	`|MASTER_SITE_.*` +
	`|MASTER_SORT_REGEX` +
	`|NOT_FOR_COMPILER` +
	`|NOT_FOR_PLATFORM` +
	`|ONLY_FOR_COMPILER` +
	`|ONLY_FOR_PLATFORM` +
	`|PERL5_PACKLIST` +
	`|PLIST_CAT` +
	`|PLIST_PRE` +
	`|PKG_FAIL_REASON` +
	`|PKG_SKIP_REASON` +
	`|PREPEND_PATH` +
	`|PYTHON_VERSIONS_INCOMPATIBLE` +
	`|REPLACE_INTERPRETER` +
	`|REPLACE_PERL` +
	`|REPLACE_RUBY` +
	`|RESTRICTED` +
	`|SITES_.+` +
	`|TOOLS_ALIASES\..+` +
	`|TOOLS_BROKEN` +
	`|TOOLS_CREATE` +
	`|TOOLS_GNU_MISSING` +
	`|TOOLS_NOOP` +
	`)$`

func (mkline *MkLine) CheckVartype(varname string, op MkOperator, value, comment string) {
	if G.opts.Debug {
		defer tracecall(varname, op, value, comment)()
	}

	if !G.opts.WarnTypes {
		return
	}

	varbase := varnameBase(varname)
	vartype := mkline.getVariableType(varname)

	if op == opAssignAppend {
		if vartype != nil {
			if !vartype.MayBeAppendedTo() {
				mkline.Warn0("The \"+=\" operator should only be used with lists.")
			}
		} else if !hasPrefix(varbase, "_") && !matches(varbase, reVarnamePlural) {
			mkline.Warn1("As %s is modified using \"+=\", its name should indicate plural.", varname)
		}
	}

	switch {
	case vartype == nil:
		// Cannot check anything if the type is not known.
		if G.opts.Debug {
			traceStep1("Unchecked variable assignment for %s.", varname)
		}

	case op == opAssignShell:
		if G.opts.Debug {
			traceStep1("Unchecked use of !=: %q", value)
		}

	case vartype.kindOfList == lkNone:
		mkline.CheckVartypePrimitive(varname, vartype.checker, op, value, comment, vartype.IsConsideredList(), vartype.guessed)

	case vartype.kindOfList == lkSpace:
		for _, word := range splitOnSpace(value) {
			mkline.CheckVartypePrimitive(varname, vartype.checker, op, word, comment, true, vartype.guessed)
		}

	case vartype.kindOfList == lkShell:
		words, _ := splitIntoShellWords(mkline.Line, value)
		for _, word := range words {
			mkline.CheckVartypePrimitive(varname, vartype.checker, op, word, comment, true, vartype.guessed)
		}
	}
}

// For some variables (like `BuildlinkDepth`), `op` influences the valid values.
// The `comment` parameter comes from a variable assignment, when a part of the line is commented out.
func (mkline *MkLine) CheckVartypePrimitive(varname string, checker *VarChecker, op MkOperator, value, comment string, isList bool, guessed bool) {
	if G.opts.Debug {
		defer tracecall(varname, checker.name, op, value, comment, isList, guessed)()
	}

	ctx := &VartypeCheck{mkline, mkline.Line, varname, op, value, "", comment, isList, guessed}
	ctx.valueNovar = mkline.withoutMakeVariables(value, isList)

	checker.checker(ctx)
}

func (mkline *MkLine) withoutMakeVariables(value string, qModifierAllowed bool) string {
	valueNovar := value
	for {
		var m []string
		if m, valueNovar = replaceFirst(valueNovar, `\$\{([^{}]*)\}`, ""); m != nil {
			varuse := m[1]
			if !qModifierAllowed && hasSuffix(varuse, ":Q") {
				mkline.Warn0("The :Q operator should only be used in lists and shell commands.")
			}
		} else {
			return valueNovar
		}
	}
}

func (mkline *MkLine) checkText(text string) {
	if G.opts.Debug {
		defer tracecall1(text)()
	}

	if contains(text, "${WRKSRC}/..") {
		mkline.Warn0("Building the package should take place entirely inside ${WRKSRC}, not \"${WRKSRC}/..\".")
		Explain(
			"WRKSRC should be defined so that there is no need to do anything",
			"outside of this directory.",
			"",
			"Example:",
			"",
			"\tWRKSRC=\t${WRKDIR}",
			"\tCONFIGURE_DIRS=\t${WRKSRC}/lib ${WRKSRC}/src",
			"\tBUILD_DIRS=\t${WRKSRC}/lib ${WRKSRC}/src ${WRKSRC}/cmd",
			"",
			"See the pkgsrc guide, section \"Directories used during the build",
			"process\" for more information.")
	}

	// Note: A simple -R is not detected, as the rate of false positives is too high.
	if m, flag := match1(text, `\b(-Wl,--rpath,|-Wl,-rpath-link,|-Wl,-rpath,|-Wl,-R)\b`); m {
		mkline.Warn1("Please use ${COMPILER_RPATH_FLAG} instead of %q.", flag)
	}

	rest := text
	for {
		m, r := replaceFirst(rest, `(?:^|[^$])\$\{([-A-Z0-9a-z_]+)(\.[\-0-9A-Z_a-z]+)?(?::[^\}]+)?\}`, "")
		if m == nil {
			break
		}
		rest = r

		varbase, varext := m[1], m[2]
		varname := varbase + varext
		varcanon := varnameCanon(varname)
		instead := G.globalData.Deprecated[varname]
		if instead == "" {
			instead = G.globalData.Deprecated[varcanon]
		}
		if instead != "" {
			mkline.Warn2("Use of %q is deprecated. %s", varname, instead)
		}
	}
}

func (mkline *MkLine) CheckCond() {
	if G.opts.Debug {
		defer tracecall0()()
	}

	p := NewParser(mkline.Line, mkline.Args())
	cond := p.MkCond()
	if !p.EOF() {
		mkline.Warn1("Invalid conditional %q.", mkline.Args())
		return
	}

	cond.Visit("empty", func(node *Tree) {
		varuse := node.args[0].(MkVarUse)
		varname := varuse.varname
		if matches(varname, `^\$.*:[MN]`) {
			mkline.Warn0("The empty() function takes a variable name as parameter, not a variable expression.")
			Explain(
				"Instead of empty(${VARNAME:Mpattern}), you should write either",
				"of the following:",
				"",
				"\tempty(VARNAME:Mpattern)",
				"\t${VARNAME:Mpattern} == \"\"",
				"",
				"Instead of !empty(${VARNAME:Mpattern}), you should write either",
				"of the following:",
				"",
				"\t!empty(VARNAME:Mpattern)",
				"\t${VARNAME:Mpattern}")
		}
		for _, modifier := range varuse.modifiers {
			if modifier[0] == 'M' || modifier[0] == 'N' {
				mkline.CheckVartype(varname, opUseMatch, modifier[1:], "")
			}
		}
	})

	cond.Visit("compareVarStr", func(node *Tree) {
		varuse := node.args[0].(MkVarUse)
		varname := varuse.varname
		varmods := varuse.modifiers
		value := node.args[2].(string)
		if len(varmods) == 0 {
			mkline.CheckVartype(varname, opUse, value, "")
		} else if len(varmods) == 1 && matches(varmods[0], `^[MN]`) && value != "" {
			mkline.CheckVartype(varname, opUseMatch, value, "")
		}
	})
}

func (mkline *MkLine) CheckValidCharactersInValue(reValid string) {
	rest := regcomp(reValid).ReplaceAllString(mkline.Value(), "")
	if rest != "" {
		uni := ""
		for _, c := range rest {
			uni += fmt.Sprintf(" %U", c)
		}
		mkline.Warn2("%s contains invalid characters (%s).", mkline.Varname(), uni[1:])
	}
}

func (mkline *MkLine) explainRelativeDirs() {
	Explain3(
		"Directories in the form \"../../category/package\" make it easier to",
		"move a package around in pkgsrc, for example from pkgsrc-wip to the",
		"main pkgsrc repository.")
}

func (mkline *MkLine) CheckRelativePkgdir(pkgdir string) {
	mkline.CheckRelativePath(pkgdir, true)
	pkgdir = resolveVarsInRelativePath(pkgdir, false)

	if m, otherpkgpath := match1(pkgdir, `^(?:\./)?\.\./\.\./([^/]+/[^/]+)$`); m {
		if !fileExists(G.globalData.Pkgsrcdir + "/" + otherpkgpath + "/Makefile") {
			mkline.Error1("There is no package in %q.", otherpkgpath)
		}

	} else if !containsVarRef(pkgdir) {
		mkline.Warn1("%q is not a valid relative package directory.", pkgdir)
		Explain3(
			"A relative pathname always starts with \"../../\", followed",
			"by a category, a slash and a the directory name of the package.",
			"For example, \"../../misc/screen\" is a valid relative pathname.")
	}
}

func (mkline *MkLine) CheckRelativePath(path string, mustExist bool) {
	if !G.Wip && contains(path, "/wip/") {
		mkline.Line.Error0("A main pkgsrc package must not depend on a pkgsrc-wip package.")
	}

	resolvedPath := resolveVarsInRelativePath(path, true)
	if containsVarRef(resolvedPath) {
		return
	}

	abs := resolvedPath
	if !hasPrefix(abs, "/") {
		abs = G.CurrentDir + "/" + abs
	}
	if _, err := os.Stat(abs); err != nil {
		if mustExist {
			mkline.Error1("%q does not exist.", resolvedPath)
		}
		return
	}

	if hasPrefix(path, "../") &&
		!matches(path, `^\.\./\.\./[^/]+/[^/]`) &&
		!(G.CurPkgsrcdir == ".." && hasPrefix(path, "../mk/")) && // For category Makefiles.
		!hasPrefix(path, "../../mk/") {
		mkline.Warn1("Invalid relative path %q.", path)
	}
}

func matchMkCond(text string) (m bool, indent, directive, args string) {
	i, n := 0, len(text)
	if i < n && text[i] == '.' {
		i++
	} else {
		return
	}

	indentStart := i
	for i < n && (text[i] == ' ' || text[i] == '\t') {
		i++
	}
	indentEnd := i

	directiveStart := i
	for i < n && 'a' <= text[i] && text[i] <= 'z' {
		i++
	}
	directiveEnd := i
	directive = text[directiveStart:directiveEnd]
	switch directive {
	case "if", "ifdef", "ifndef", "else", "elif", "endif", "for", "endfor", "undef":
		break
	default:
		return
	}

	for i < n && (text[i] == ' ' || text[i] == '\t') {
		i++
	}

	argsStart := i
	for i < n && text[i] != '#' {
		i++
	}
	for i > argsStart && (text[i-1] == ' ' || text[i-1] == '\t') {
		i--
	}
	argsEnd := i

	m = true
	indent = text[indentStart:indentEnd]
	args = text[argsStart:argsEnd]
	return
}

type NeedsQuoting uint8

const (
	nqNo NeedsQuoting = iota
	nqYes
	nqDoesntMatter
	nqDontKnow
)

func (nq NeedsQuoting) String() string {
	return [...]string{"no", "yes", "doesn't matter", "don't known"}[nq]
}

func (mkline *MkLine) variableNeedsQuoting(varname string, vartype *Vartype, vuc *VarUseContext) (needsQuoting NeedsQuoting) {
	if G.opts.Debug {
		defer tracecall(varname, vartype, vuc, "=>", &needsQuoting)()
	}

	if vartype == nil || vuc.vartype == nil {
		return nqDontKnow
	}

	isPlainWord := vartype.checker.IsEnum()
	switch vartype.checker {
	case CheckvarBuildlinkDepmethod,
		CheckvarCategory,
		CheckvarDistSuffix,
		CheckvarEmulPlatform,
		CheckvarFileMode,
		CheckvarFilename,
		CheckvarIdentifier,
		CheckvarInteger,
		CheckvarMachineGnuPlatform,
		CheckvarMachinePlatform,
		CheckvarOption,
		CheckvarPathname,
		CheckvarPerl5Packlist,
		CheckvarPkgName,
		CheckvarPkgOptionsVar,
		CheckvarPkgPath,
		CheckvarPkgRevision,
		CheckvarPrefixPathname,
		CheckvarPythonDependency,
		CheckvarRelativePkgDir,
		CheckvarRelativePkgPath,
		CheckvarStage,
		CheckvarUserGroupName,
		CheckvarVersion,
		CheckvarWrkdirSubdirectory,
		CheckvarYesNo,
		CheckvarYesNoIndirectly:
		isPlainWord = true
	}
	if isPlainWord {
		if vartype.kindOfList == lkNone {
			return nqDoesntMatter
		}
		if vartype.kindOfList == lkShell && vuc.extent != vucExtentWordpart {
			return nqNo
		}
	}

	// In .for loops, the :Q operator is always misplaced, since
	// the items are broken up at white-space, not as shell words
	// like in all other parts of make(1).
	if vuc.quoting == vucQuotFor {
		return nqNo
	}

	// Determine whether the context expects a list of shell words or not.
	wantList := vuc.vartype.IsConsideredList()
	haveList := vartype.IsConsideredList()

	if G.opts.Debug {
		traceStep("variableNeedsQuoting: varname=%q, context=%v, type=%v, wantList=%v, haveList=%v",
			varname, vuc, vartype, wantList, haveList)
	}

	// A shell word may appear as part of a shell word, for example COMPILER_RPATH_FLAG.
	if vuc.extent == vucExtentWordpart && vuc.quoting == vucQuotPlain {
		if vartype.kindOfList == lkNone && vartype.checker == CheckvarShellWord {
			return nqNo
		}
	}

	// Both of these can be correct, depending on the situation:
	// 1. echo ${PERL5:Q}
	// 2. xargs ${PERL5}
	if vuc.extent == vucExtentWord && vuc.quoting == vucQuotPlain {
		if wantList && haveList {
			return nqDontKnow
		}
	}

	// Assuming the tool definitions don't include very special characters,
	// so they can safely be used inside any quotes.
	if G.globalData.Tools.byVarname[varname] != nil {
		switch vuc.quoting {
		case vucQuotPlain:
			if vuc.extent != vucExtentWordpart {
				return nqNo
			}
		case vucQuotBackt:
			return nqNo
		case vucQuotDquot, vucQuotSquot:
			return nqDoesntMatter
		}
	}

	// Variables that appear as parts of shell words generally need
	// to be quoted. An exception is in the case of backticks,
	// because the whole backticks expression is parsed as a single
	// shell word by pkglint.
	if vuc.extent == vucExtentWordpart && vuc.vartype != nil && vuc.vartype.IsShell() && vuc.quoting != vucQuotBackt {
		return nqYes
	}

	// Assigning lists to lists does not require any quoting, though
	// there may be cases like "CONFIGURE_ARGS+= -libs ${LDFLAGS:Q}"
	// where quoting is necessary.
	if wantList && haveList && vuc.extent != vucExtentWordpart {
		return nqDoesntMatter
	}

	if G.opts.Debug {
		trace("", "MkLine.variableNeedsQuoting", "wantList", wantList, "haveList", haveList)
	}
	if wantList != haveList {
		if vuc.vartype != nil && vartype != nil {
			if vuc.vartype.checker == CheckvarFetchURL && vartype.checker == CheckvarURL {
				return nqNo
			}
		}
		return nqYes
	}

	if G.opts.Debug {
		traceStep1("Don't know whether :Q is needed for %q", varname)
	}
	return nqDontKnow
}

// Returns the type of the variable (maybe guessed based on the variable name),
// or nil if the type cannot even be guessed.
func (mkline *MkLine) getVariableType(varname string) *Vartype {
	if G.opts.Debug {
		defer tracecall1(varname)()
	}

	if vartype := G.globalData.vartypes[varname]; vartype != nil {
		return vartype
	}
	if vartype := G.globalData.vartypes[varnameCanon(varname)]; vartype != nil {
		return vartype
	}

	if G.globalData.Tools.byVarname[varname] != nil {
		return &Vartype{lkNone, CheckvarShellCommand, []AclEntry{{"*", aclpUse}}, false}
	}

	if m, toolvarname := match1(varname, `^TOOLS_(.*)`); m && G.globalData.Tools.byVarname[toolvarname] != nil {
		return &Vartype{lkNone, CheckvarPathname, []AclEntry{{"*", aclpUse}}, false}
	}

	allowAll := []AclEntry{{"*", aclpAll}}
	allowRuntime := []AclEntry{{"*", aclpAllRuntime}}

	// Guess the datatype of the variable based on naming conventions.
	varbase := varnameBase(varname)
	var gtype *Vartype
	switch {
	case hasSuffix(varbase, "DIRS"):
		gtype = &Vartype{lkShell, CheckvarPathmask, allowRuntime, true}
	case hasSuffix(varbase, "DIR"), hasSuffix(varname, "_HOME"):
		gtype = &Vartype{lkNone, CheckvarPathname, allowRuntime, true}
	case hasSuffix(varbase, "FILES"):
		gtype = &Vartype{lkShell, CheckvarPathmask, allowRuntime, true}
	case hasSuffix(varbase, "FILE"):
		gtype = &Vartype{lkNone, CheckvarPathname, allowRuntime, true}
	case hasSuffix(varbase, "PATH"):
		gtype = &Vartype{lkNone, CheckvarPathlist, allowRuntime, true}
	case hasSuffix(varbase, "PATHS"):
		gtype = &Vartype{lkShell, CheckvarPathname, allowRuntime, true}
	case hasSuffix(varbase, "_USER"):
		gtype = &Vartype{lkNone, CheckvarUserGroupName, allowAll, true}
	case hasSuffix(varbase, "_GROUP"):
		gtype = &Vartype{lkNone, CheckvarUserGroupName, allowAll, true}
	case hasSuffix(varbase, "_ENV"):
		gtype = &Vartype{lkShell, CheckvarShellWord, allowRuntime, true}
	case hasSuffix(varbase, "_CMD"):
		gtype = &Vartype{lkNone, CheckvarShellCommand, allowRuntime, true}
	case hasSuffix(varbase, "_ARGS"):
		gtype = &Vartype{lkShell, CheckvarShellWord, allowRuntime, true}
	case hasSuffix(varbase, "_CFLAGS"), hasSuffix(varname, "_CPPFLAGS"), hasSuffix(varname, "_CXXFLAGS"):
		gtype = &Vartype{lkShell, CheckvarCFlag, allowRuntime, true}
	case hasSuffix(varname, "_LDFLAGS"):
		gtype = &Vartype{lkShell, CheckvarLdFlag, allowRuntime, true}
	case hasSuffix(varbase, "_MK"):
		gtype = &Vartype{lkNone, CheckvarUnchecked, allowAll, true}
	case hasPrefix(varbase, "PLIST."):
		gtype = &Vartype{lkNone, CheckvarYes, allowAll, true}
	}

	if G.opts.Debug {
		if gtype != nil {
			traceStep2("The guessed type of %q is %q.", varname, gtype.String())
		} else {
			traceStep1("No type definition found for %q.", varname)
		}
	}
	return gtype
}

// TODO: merge with determineUsedVariables
func (mkline *MkLine) extractUsedVariables(text string) []string {
	re := regcomp(`^(?:[^\$]+|\$[\$*<>?@]|\$\{([.0-9A-Z_a-z]+)(?::(?:[^\${}]|\$[^{])+)?\})`)
	rest := text
	var result []string
	for {
		m := re.FindStringSubmatchIndex(rest)
		if m == nil {
			break
		}
		varname := rest[negToZero(m[2]):negToZero(m[3])]
		rest = rest[:m[0]] + rest[m[1]:]
		if varname != "" {
			result = append(result, varname)
		}
	}

	if G.opts.Debug && rest != "" {
		traceStep1("extractUsedVariables: rest=%q", rest)
	}
	return result
}

func (mkline *MkLine) determineUsedVariables() (varnames []string) {
	rest := mkline.Line.Text

	if strings.HasPrefix(rest, "#") {
		return
	}

	for {
		p1 := strings.Index(rest, "${")
		p2 := strings.Index(rest, "$(")
		p3 := strings.Index(rest, "defined(")
		p4 := strings.Index(rest, "empty(")
		if p1 == -1 && p2 == -1 && p3 == -1 && p4 == -1 {
			return
		}
		min := -1
		if min == -1 || (p1 != -1 && p1 < min) {
			min = p1
		}
		if min == -1 || (p2 != -1 && p2 < min) {
			min = p2
		}
		if min == -1 || (p3 != -1 && p3 < min) {
			min = p3
		}
		if min == -1 || (p4 != -1 && p4 < min) {
			min = p4
		}
		rest = rest[min:]

		m := regcomp(`(?:\$\{|\$\(|defined\(|empty\()([0-9+.A-Z_a-z]+)[:})]`).FindStringSubmatchIndex(rest)
		if m == nil {
			return
		}
		varname := rest[m[2]:m[3]]
		varnames = append(varnames, varname)
		rest = rest[:m[0]] + rest[m[1]:]
	}
	return
}

// VarUseContext defines the context in which a variable is defined
// or used. Whether that is allowed depends on:
//
// * The variable’s data type, as defined in vardefs.go.
// * When used on the right-hand side of an assigment, the variable can
//   represent a list of words, a single word or even only part of a
//   word. This distinction decides upon the correct use of the :Q
//   operator.
// * When used in preprocessing statements like .if or .for, the other
//   operands of that statement should fit to the variable and are
//   checked against the variable type. For example, comparing OPSYS to
//   x86_64 doesn’t make sense.
type VarUseContext struct {
	vartype *Vartype
	time    vucTime
	quoting vucQuoting
	extent  vucExtent
}

type vucTime uint8

const (
	vucTimeUnknown vucTime = iota

	// When Makefiles are loaded, the operators := and != are evaluated,
	// as well as the conditionals .if, .elif and .for.
	// During loading, not all variables are available yet.
	// Variable values are still subject to change, especially lists.
	vucTimeParse

	// All files have been read, all variables can be referenced.
	// Variable values don’t change anymore.
	vucTimeRun
)

func (t vucTime) String() string { return [...]string{"unknown", "parse", "run"}[t] }

// The quoting context in which the variable is used.
// Depending on this context, the modifiers :Q or :M can be allowed or not.
type vucQuoting uint8

const (
	vucQuotUnknown vucQuoting = iota
	vucQuotPlain              // Example: echo LOCALBASE=${LOCALBASE}
	vucQuotDquot              // Example: echo "The version is ${PKGVERSION}."
	vucQuotSquot              // Example: echo 'The version is ${PKGVERSION}.'
	vucQuotBackt              // Example: echo \`sed 1q ${WRKSRC}/README\`

	// The .for loop in Makefiles. This is the only place where
	// variables are split on whitespace. Everywhere else (:Q, :M)
	// they are split like in the shell.
	//
	// Example: .for f in ${EXAMPLE_FILES}
	vucQuotFor
)

func (q vucQuoting) String() string {
	return [...]string{"unknown", "plain", "dquot", "squot", "backt", "mk-for"}[q]
}

type vucExtent uint8

const (
	vucExtentUnknown  vucExtent = iota
	vucExtentWord               // Example: echo ${LOCALBASE}
	vucExtentWordpart           // Example: echo LOCALBASE=${LOCALBASE}
)

func (e vucExtent) String() string {
	return [...]string{"unknown", "word", "wordpart"}[e]
}

func (vuc *VarUseContext) String() string {
	typename := "no-type"
	if vuc.vartype != nil {
		typename = vuc.vartype.String()
	}
	return fmt.Sprintf("(%s time:%s quoting:%s extent:%s)", typename, vuc.time, vuc.quoting, vuc.extent)
}

type Indentation struct {
	data []int
}

func (ind *Indentation) Len() int        { return len(ind.data) }
func (ind *Indentation) Depth() int      { return ind.data[len(ind.data)-1] }
func (ind *Indentation) Pop()            { ind.data = ind.data[:len(ind.data)-1] }
func (ind *Indentation) Push(indent int) { ind.data = append(ind.data, indent) }
