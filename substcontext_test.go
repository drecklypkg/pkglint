package pkglint

import (
	"gopkg.in/check.v1"
)

func (s *Suite) Test_SubstContext__incomplete(c *check.C) {
	t := s.Init(c)

	t.SetUpCommandLine("-Wextra")
	ctx := NewSubstContext()

	ctx.Varassign(t.NewMkLine("Makefile", 10, "PKGNAME=pkgname-1.0"))

	t.CheckEquals(ctx.id, "")

	ctx.Varassign(t.NewMkLine("Makefile", 11, "SUBST_CLASSES+=interp"))

	t.CheckEquals(ctx.id, "interp")

	ctx.Varassign(t.NewMkLine("Makefile", 12, "SUBST_FILES.interp=Makefile"))

	t.CheckEquals(ctx.IsComplete(), false)

	ctx.Varassign(t.NewMkLine("Makefile", 13, "SUBST_SED.interp=s,@PREFIX@,${PREFIX},g"))

	t.CheckEquals(ctx.IsComplete(), false)

	ctx.Finish(t.NewMkLine("Makefile", 14, ""))

	t.CheckOutputLines(
		"NOTE: Makefile:13: The substitution command \"s,@PREFIX@,${PREFIX},g\" "+
			"can be replaced with \"SUBST_VARS.interp= PREFIX\".",
		"WARN: Makefile:14: Incomplete SUBST block: SUBST_STAGE.interp missing.")
}

func (s *Suite) Test_SubstContext__complete(c *check.C) {
	t := s.Init(c)

	t.SetUpCommandLine("-Wextra")
	ctx := NewSubstContext()

	ctx.Varassign(t.NewMkLine("Makefile", 10, "PKGNAME=pkgname-1.0"))
	ctx.Varassign(t.NewMkLine("Makefile", 11, "SUBST_CLASSES+=p"))
	ctx.Varassign(t.NewMkLine("Makefile", 12, "SUBST_FILES.p=Makefile"))
	ctx.Varassign(t.NewMkLine("Makefile", 13, "SUBST_SED.p=s,@PREFIX@,${PREFIX},g"))

	t.CheckEquals(ctx.IsComplete(), false)

	ctx.Varassign(t.NewMkLine("Makefile", 14, "SUBST_STAGE.p=post-configure"))

	t.CheckEquals(ctx.IsComplete(), true)

	ctx.Finish(t.NewMkLine("Makefile", 15, ""))

	t.CheckOutputLines(
		"NOTE: Makefile:13: The substitution command \"s,@PREFIX@,${PREFIX},g\" " +
			"can be replaced with \"SUBST_VARS.p= PREFIX\".")
}

func (s *Suite) Test_SubstContext__OPSYSVARS(c *check.C) {
	t := s.Init(c)

	G.Opts.WarnExtra = true
	ctx := NewSubstContext()

	// SUBST_CLASSES is added to OPSYSVARS in mk/bsd.pkg.mk.
	ctx.Varassign(t.NewMkLine("Makefile", 11, "SUBST_CLASSES.SunOS+=prefix"))
	ctx.Varassign(t.NewMkLine("Makefile", 12, "SUBST_CLASSES.NetBSD+=prefix"))
	ctx.Varassign(t.NewMkLine("Makefile", 13, "SUBST_FILES.prefix=Makefile"))
	ctx.Varassign(t.NewMkLine("Makefile", 14, "SUBST_SED.prefix=s,@PREFIX@,${PREFIX},g"))
	ctx.Varassign(t.NewMkLine("Makefile", 15, "SUBST_STAGE.prefix=post-configure"))

	t.CheckEquals(ctx.IsComplete(), true)

	ctx.Finish(t.NewMkLine("Makefile", 15, ""))

	t.CheckOutputLines(
		"NOTE: Makefile:14: The substitution command \"s,@PREFIX@,${PREFIX},g\" " +
			"can be replaced with \"SUBST_VARS.prefix= PREFIX\".")
}

func (s *Suite) Test_SubstContext__no_class(c *check.C) {
	t := s.Init(c)

	t.SetUpCommandLine("-Wextra")
	ctx := NewSubstContext()

	ctx.Varassign(t.NewMkLine("Makefile", 10, "UNRELATED=anything"))
	ctx.Varassign(t.NewMkLine("Makefile", 11, "SUBST_FILES.repl+=Makefile.in"))
	ctx.Varassign(t.NewMkLine("Makefile", 12, "SUBST_SED.repl+=-e s,from,to,g"))
	ctx.Finish(t.NewMkLine("Makefile", 13, ""))

	t.CheckOutputLines(
		"WARN: Makefile:11: SUBST_CLASSES should come before the definition of \"SUBST_FILES.repl\".",
		"WARN: Makefile:13: Incomplete SUBST block: SUBST_STAGE.repl missing.")
}

func (s *Suite) Test_SubstContext__multiple_classes_in_one_line(c *check.C) {
	t := s.Init(c)

	t.SetUpCommandLine("-Wextra")
	mklines := t.NewMkLines("filename.mk",
		"SUBST_CLASSES+=         one two",
		"SUBST_STAGE.one=        post-configure",
		"SUBST_FILES.one=        one.txt",
		"SUBST_SED.one=          s,one,1,g",
		"SUBST_STAGE.two=        post-configure",
		"SUBST_FILES.two=        two.txt",
		"")
	ctx := NewSubstContext()

	mklines.ForEach(ctx.Process)

	t.CheckOutputLines(
		"WARN: filename.mk:1: Please add only one class at a time to SUBST_CLASSES.",
		"WARN: filename.mk:7: Incomplete SUBST block: SUBST_SED.two, SUBST_VARS.two or SUBST_FILTER_CMD.two missing.")
}

func (s *Suite) Test_SubstContext__multiple_classes_in_one_block(c *check.C) {
	t := s.Init(c)

	t.SetUpCommandLine("-Wextra")

	mklines := t.NewMkLines("filename.mk",
		"SUBST_CLASSES+=         one",
		"SUBST_STAGE.one=        post-configure",
		"SUBST_STAGE.one=        post-configure",
		"SUBST_FILES.one=        one.txt",
		"SUBST_CLASSES+=         two", // The block "one" is not finished yet.
		"SUBST_SED.one=          s,one,1,g",
		"SUBST_STAGE.two=        post-configure",
		"SUBST_FILES.two=        two.txt",
		"SUBST_SED.two=          s,two,2,g",
		"")
	ctx := NewSubstContext()

	mklines.ForEach(ctx.Process)

	t.CheckOutputLines(
		"WARN: filename.mk:3: Duplicate definition of \"SUBST_STAGE.one\".",
		"WARN: filename.mk:5: Incomplete SUBST block: SUBST_SED.one, SUBST_VARS.one or SUBST_FILTER_CMD.one missing.",
		"WARN: filename.mk:5: Subst block \"one\" should be finished before adding the next class to SUBST_CLASSES.",
		"WARN: filename.mk:6: Variable \"SUBST_SED.one\" does not match SUBST class \"two\".")
}

func (s *Suite) Test_SubstContext__files_missing(c *check.C) {
	t := s.Init(c)

	mklines := t.NewMkLines("filename.mk",
		"SUBST_CLASSES+=         one",
		"SUBST_STAGE.one=        pre-configure",
		"SUBST_CLASSES+=         two",
		"SUBST_STAGE.two=        pre-configure",
		"SUBST_FILES.two=        two.txt",
		"SUBST_SED.two=          s,two,2,g",
		"")
	ctx := NewSubstContext()

	mklines.ForEach(ctx.Process)

	t.CheckOutputLines(
		"WARN: filename.mk:3: Incomplete SUBST block: SUBST_FILES.one missing.",
		"WARN: filename.mk:3: Incomplete SUBST block: "+
			"SUBST_SED.one, SUBST_VARS.one or SUBST_FILTER_CMD.one missing.",
		"WARN: filename.mk:3: Subst block \"one\" should be finished "+
			"before adding the next class to SUBST_CLASSES.")
}

func (s *Suite) Test_SubstContext__directives(c *check.C) {
	t := s.Init(c)

	t.SetUpCommandLine("-Wextra")

	mklines := t.NewMkLines("filename.mk",
		"SUBST_CLASSES+=         os",
		"SUBST_STAGE.os=         post-configure",
		"SUBST_MESSAGE.os=       Guessing operating system",
		"SUBST_FILES.os=         guess-os.h",
		".if ${OPSYS} == NetBSD",
		"SUBST_FILTER_CMD.os=    ${SED} -e s,@OPSYS@,NetBSD,",
		".elif ${OPSYS} == Darwin",
		"SUBST_SED.os=           -e s,@OPSYS@,Darwin1,",
		"SUBST_SED.os=           -e s,@OPSYS@,Darwin2,",
		".elif ${OPSYS} == Linux",
		"SUBST_SED.os=           -e s,@OPSYS@,Linux,",
		".else",
		"SUBST_VARS.os=           OPSYS",
		".endif",
		"")
	ctx := NewSubstContext()

	mklines.ForEach(ctx.Process)

	// All the other lines are correctly determined as being alternatives
	// to each other. And since every branch contains some transformation
	// (SED, VARS, FILTER_CMD), everything is fine.
	t.CheckOutputLines(
		"WARN: filename.mk:9: All but the first \"SUBST_SED.os\" lines " +
			"should use the \"+=\" operator.")
}

func (s *Suite) Test_SubstContext__directives_around_everything_then(c *check.C) {
	t := s.Init(c)

	t.SetUpCommandLine("-Wextra")

	mklines := t.NewMkLines("filename.mk",
		"SUBST_CLASSES+=         os",
		".if ${OPSYS} == NetBSD",
		"SUBST_VARS.os=          OPSYS",
		"SUBST_SED.os=           -e s,@OPSYS@,NetBSD,",
		"SUBST_STAGE.os=         post-configure",
		"SUBST_MESSAGE.os=       Guessing operating system",
		"SUBST_FILES.os=         guess-os.h",
		".endif",
		"")
	ctx := NewSubstContext()

	mklines.ForEach(ctx.Process)

	// TODO: Warn about missing SUBST_STAGE.
	t.CheckOutputLines(
		"WARN: filename.mk:9: Incomplete SUBST block: SUBST_FILES.os missing.",
		"WARN: filename.mk:9: Incomplete SUBST block: "+
			"SUBST_SED.os, SUBST_VARS.os or SUBST_FILTER_CMD.os missing.")
}

func (s *Suite) Test_SubstContext__directives_around_everything_else(c *check.C) {
	t := s.Init(c)

	t.SetUpCommandLine("-Wextra")

	mklines := t.NewMkLines("filename.mk",
		"SUBST_CLASSES+=         os",
		".if ${OPSYS} == NetBSD",
		".else",
		"SUBST_VARS.os=          OPSYS",
		"SUBST_SED.os=           -e s,@OPSYS@,NetBSD,",
		"SUBST_STAGE.os=         post-configure",
		"SUBST_MESSAGE.os=       Guessing operating system",
		"SUBST_FILES.os=         guess-os.h",
		".endif",
		"")
	ctx := NewSubstContext()

	mklines.ForEach(ctx.Process)

	t.CheckOutputLines(
		"WARN: filename.mk:10: Incomplete SUBST block: SUBST_FILES.os missing.",
		"WARN: filename.mk:10: Incomplete SUBST block: "+
			"SUBST_SED.os, SUBST_VARS.os or SUBST_FILTER_CMD.os missing.")
}

func (s *Suite) Test_SubstContext__empty_directive(c *check.C) {
	t := s.Init(c)

	t.SetUpCommandLine("-Wextra")

	mklines := t.NewMkLines("filename.mk",
		"SUBST_CLASSES+=         os",
		"SUBST_VARS.os=          OPSYS",
		"SUBST_SED.os=           -e s,@OPSYS@,NetBSD,",
		"SUBST_STAGE.os=         post-configure",
		"SUBST_MESSAGE.os=       Guessing operating system",
		"SUBST_FILES.os=         guess-os.h",
		".if ${OPSYS} == NetBSD",
		".else",
		".endif",
		"")
	ctx := NewSubstContext()

	mklines.ForEach(ctx.Process)

	t.CheckOutputEmpty()
}

func (s *Suite) Test_SubstContext__missing_transformation_in_one_branch(c *check.C) {
	t := s.Init(c)

	t.SetUpCommandLine("-Wextra")

	mklines := t.NewMkLines("filename.mk",
		"SUBST_CLASSES+=         os",
		"SUBST_STAGE.os=         post-configure",
		"SUBST_MESSAGE.os=       Guessing operating system",
		"SUBST_FILES.os=         guess-os.h",
		".if ${OPSYS} == NetBSD",
		"SUBST_FILES.os=         -e s,@OpSYS@,NetBSD,", // A simple typo, this should be SUBST_SED.
		".elif ${OPSYS} == Darwin",
		"SUBST_SED.os=           -e s,@OPSYS@,Darwin1,",
		"SUBST_SED.os=           -e s,@OPSYS@,Darwin2,",
		".else",
		"SUBST_VARS.os=           OPSYS",
		".endif",
		"")
	ctx := NewSubstContext()

	mklines.ForEach(ctx.Process)

	t.CheckOutputLines(
		"WARN: filename.mk:6: All but the first \"SUBST_FILES.os\" lines should use the \"+=\" operator.",
		"WARN: filename.mk:9: All but the first \"SUBST_SED.os\" lines should use the \"+=\" operator.",
		"WARN: filename.mk:13: Incomplete SUBST block: SUBST_SED.os, SUBST_VARS.os or SUBST_FILTER_CMD.os missing.")
}

func (s *Suite) Test_SubstContext__nested_conditionals(c *check.C) {
	t := s.Init(c)

	t.SetUpCommandLine("-Wextra")

	mklines := t.NewMkLines("filename.mk",
		"SUBST_CLASSES+=         os",
		"SUBST_STAGE.os=         post-configure",
		"SUBST_MESSAGE.os=       Guessing operating system",
		".if ${OPSYS} == NetBSD",
		"SUBST_FILES.os=         guess-netbsd.h",
		".  if ${ARCH} == i386",
		"SUBST_FILTER_CMD.os=    ${SED} -e s,@OPSYS,NetBSD-i386,",
		".  elif ${ARCH} == x86_64",
		"SUBST_VARS.os=          OPSYS",
		".  else",
		"SUBST_SED.os=           -e s,@OPSYS,NetBSD-unknown",
		".  endif",
		".else",
		// This branch omits SUBST_FILES.
		"SUBST_SED.os=           -e s,@OPSYS@,unknown,",
		".endif",
		"")
	ctx := NewSubstContext()

	mklines.ForEach(ctx.Process)

	t.CheckOutputLines(
		"WARN: filename.mk:16: Incomplete SUBST block: SUBST_FILES.os missing.")
}

func (s *Suite) Test_SubstContext__pre_patch(c *check.C) {
	t := s.Init(c)

	t.SetUpCommandLine("-Wextra", "--show-autofix")
	t.SetUpVartypes()

	mklines := t.NewMkLines("os.mk",
		MkCvsID,
		"",
		"SUBST_CLASSES+=\tos",
		"SUBST_STAGE.os=\tpre-patch",
		"SUBST_FILES.os=\tguess-os.h",
		"SUBST_SED.os=\t-e s,@OPSYS@,Darwin,")

	mklines.Check()

	t.CheckOutputLines(
		"WARN: os.mk:4: Substitutions should not happen in the patch phase.",
		"AUTOFIX: os.mk:4: Replacing \"pre-patch\" with \"post-extract\".")
}

func (s *Suite) Test_SubstContext__post_patch(c *check.C) {
	t := s.Init(c)

	t.SetUpCommandLine("-Wextra", "--show-autofix")
	t.SetUpVartypes()

	mklines := t.NewMkLines("os.mk",
		MkCvsID,
		"",
		"SUBST_CLASSES+=\tos",
		"SUBST_STAGE.os=\tpost-patch",
		"SUBST_FILES.os=\tguess-os.h",
		"SUBST_SED.os=\t-e s,@OPSYS@,Darwin,")

	mklines.Check()

	t.CheckOutputLines(
		"WARN: os.mk:4: Substitutions should not happen in the patch phase.",
		"AUTOFIX: os.mk:4: Replacing \"post-patch\" with \"pre-configure\".")
}

func (s *Suite) Test_SubstContext__with_NO_CONFIGURE(c *check.C) {
	t := s.Init(c)

	pkg := t.SetUpPackage("category/package",
		"SUBST_CLASSES+=\t\tpre",
		"SUBST_STAGE.pre=\tpre-configure",
		"SUBST_FILES.pre=\tguess-os.h",
		"SUBST_SED.pre=\t\t-e s,@OPSYS@,Darwin,",
		"",
		"SUBST_CLASSES+=\t\tpost",
		"SUBST_STAGE.post=\tpost-configure",
		"SUBST_FILES.post=\tguess-os.h",
		"SUBST_SED.post=\t\t-e s,@OPSYS@,Darwin,",
		"",
		"SUBST_CLASSES+=\te",
		"SUBST_STAGE.e=\tpost-extract",
		"SUBST_FILES.e=\tguess-os.h",
		"SUBST_SED.e=\t-e s,@OPSYS@,Darwin,",
		"",
		"NO_CONFIGURE=\tyes")
	t.FinishSetUp()

	G.Check(pkg)

	t.CheckOutputLines(
		"WARN: ~/category/package/Makefile:21: SUBST_STAGE pre-configure has no effect "+
			"when NO_CONFIGURE is set (in line 35).",
		"WARN: ~/category/package/Makefile:26: SUBST_STAGE post-configure has no effect "+
			"when NO_CONFIGURE is set (in line 35).")
}

func (s *Suite) Test_SubstContext__without_NO_CONFIGURE(c *check.C) {
	t := s.Init(c)

	pkg := t.SetUpPackage("category/package",
		"SUBST_CLASSES+=\t\tpre",
		"SUBST_STAGE.pre=\tpre-configure",
		"SUBST_FILES.pre=\tguess-os.h",
		"SUBST_SED.pre=\t\t-e s,@OPSYS@,Darwin,")
	t.FinishSetUp()

	G.Check(pkg)

	t.CheckOutputEmpty()
}

func (s *Suite) Test_SubstContext__adjacent(c *check.C) {
	t := s.Init(c)

	t.SetUpCommandLine("-Wextra")
	t.SetUpVartypes()

	mklines := t.NewMkLines("os.mk",
		MkCvsID,
		"",
		"SUBST_CLASSES+=\t1",
		"SUBST_STAGE.1=\tpre-configure",
		"SUBST_FILES.1=\tfile1",
		"SUBST_SED.1=\t-e s,subst1,repl1,",
		"SUBST_CLASSES+=\t2",
		"SUBST_SED.1+=\t-e s,subst1b,repl1b,", // Misplaced
		"SUBST_STAGE.2=\tpre-configure",
		"SUBST_FILES.2=\tfile2",
		"SUBST_SED.2=\t-e s,subst2,repl2,")

	mklines.Check()

	t.CheckOutputLines(
		"WARN: os.mk:8: Variable \"SUBST_SED.1\" does not match SUBST class \"2\".")
}

func (s *Suite) Test_SubstContext__do_patch(c *check.C) {
	t := s.Init(c)

	t.SetUpVartypes()

	mklines := t.NewMkLines("os.mk",
		MkCvsID,
		"",
		"SUBST_CLASSES+=\tos",
		"SUBST_STAGE.os=\tdo-patch",
		"SUBST_FILES.os=\tguess-os.h",
		"SUBST_SED.os=\t-e s,@OPSYS@,Darwin,")

	mklines.Check()

	// No warning, since there is nothing to fix automatically.
	// This case doesn't occur in practice anyway.
	t.CheckOutputEmpty()
}

// Variables mentioned in SUBST_VARS are not considered "foreign"
// in the block and may be mixed with the other SUBST variables.
func (s *Suite) Test_SubstContext__SUBST_VARS_defined_in_block(c *check.C) {
	t := s.Init(c)

	t.SetUpVartypes()

	mklines := t.NewMkLines("os.mk",
		MkCvsID,
		"",
		"SUBST_CLASSES+=\tos",
		"SUBST_STAGE.os=\tpre-configure",
		"SUBST_FILES.os=\tguess-os.h",
		"SUBST_VARS.os=\tTODAY1",
		"TODAY1!=\tdate",
		"TODAY2!=\tdate")

	mklines.Check()

	t.CheckOutputLines(
		"WARN: os.mk:8: TODAY2 is defined but not used.",
		"WARN: os.mk:8: Foreign variable \"TODAY2\" in SUBST block.")
}

// Variables mentioned in SUBST_VARS may appear in the same paragraph,
// or alternatively anywhere else in the file.
func (s *Suite) Test_SubstContext__SUBST_VARS_in_next_paragraph(c *check.C) {
	t := s.Init(c)

	t.SetUpVartypes()

	mklines := t.NewMkLines("os.mk",
		MkCvsID,
		"",
		"SUBST_CLASSES+=\tos",
		"SUBST_STAGE.os=\tpre-configure",
		"SUBST_FILES.os=\tguess-os.h",
		"SUBST_VARS.os=\tTODAY1",
		"",
		"TODAY1!=\tdate",
		"TODAY2!=\tdate")

	mklines.Check()

	t.CheckOutputLines(
		"WARN: os.mk:9: TODAY2 is defined but not used.")
}

func (s *Suite) Test_SubstContext__multiple_SUBST_VARS(c *check.C) {
	t := s.Init(c)

	t.SetUpCommandLine("-Wextra")
	t.SetUpVartypes()

	mklines := t.NewMkLines("os.mk",
		MkCvsID,
		"",
		"SUBST_CLASSES+=\tos",
		"SUBST_STAGE.os=\tpre-configure",
		"SUBST_FILES.os=\tguess-os.h",
		"SUBST_VARS.os=\tPREFIX VARBASE")

	mklines.Check()

	t.CheckOutputEmpty()
}

// As of May 2019, pkglint does not check the order of the variables in
// a SUBST block. Enforcing this order, or at least suggesting it, would
// make pkgsrc packages more uniform, which is a good idea, but not urgent.
func (s *Suite) Test_SubstContext__unusual_variable_order(c *check.C) {
	t := s.Init(c)

	t.SetUpVartypes()

	mklines := t.NewMkLines("subst.mk",
		MkCvsID,
		"",
		"SUBST_CLASSES+=\t\tid",
		"SUBST_SED.id=\t\t-e /deleteme/d",
		"SUBST_FILES.id=\t\tfile",
		"SUBST_MESSAGE.id=\tMessage",
		"SUBST_STAGE.id=\t\tpre-configure")

	mklines.Check()

	t.CheckOutputEmpty()
}

// Since the SUBST_CLASSES definition starts the SUBST block, all
// directives above it are ignored by the SUBST context.
func (s *Suite) Test_SubstContext_Directive__before_SUBST_CLASSES(c *check.C) {
	t := s.Init(c)

	t.SetUpVartypes()
	t.DisableTracing() // Just for branch coverage.

	mklines := t.NewMkLines("os.mk",
		MkCvsID,
		"",
		".if 0",
		".endif",
		"SUBST_CLASSES+=\tos",
		".elif 0") // Just for branch coverage.

	mklines.Check()

	t.CheckOutputLines(
		"WARN: os.mk:EOF: Incomplete SUBST block: SUBST_STAGE.os missing.",
		"WARN: os.mk:EOF: Incomplete SUBST block: SUBST_FILES.os missing.",
		"WARN: os.mk:EOF: Incomplete SUBST block: "+
			"SUBST_SED.os, SUBST_VARS.os or SUBST_FILTER_CMD.os missing.")
}

func (s *Suite) Test_SubstContext_suggestSubstVars(c *check.C) {
	t := s.Init(c)

	t.SetUpVartypes()
	t.SetUpTool("sh", "SH", AtRunTime)

	mklines := t.NewMkLines("subst.mk",
		MkCvsID,
		"",
		"SUBST_CLASSES+=\t\ttest",
		"SUBST_STAGE.test=\tpre-configure",
		"SUBST_FILES.test=\tfilename",
		"SUBST_SED.test+=\t-e s,@SH@,${SH},g",            // Can be replaced.
		"SUBST_SED.test+=\t-e s,@SH@,${SH:Q},g",          // Can be replaced, with or without the :Q modifier.
		"SUBST_SED.test+=\t-e s,@SH@,${SH:T},g",          // Cannot be replaced because of the :T modifier.
		"SUBST_SED.test+=\t-e s,@SH@,${SH},",             // Can be replaced, even without the g option.
		"SUBST_SED.test+=\t-e 's,@SH@,${SH},'",           // Can be replaced, whether in single quotes or not.
		"SUBST_SED.test+=\t-e \"s,@SH@,${SH},\"",         // Can be replaced, whether in double quotes or not.
		"SUBST_SED.test+=\t-e s,'@SH@','${SH}',",         // Can be replaced, even when the quoting changes midways.
		"SUBST_SED.test+=\ts,'@SH@','${SH}',",            // Can be replaced manually, even when the -e is missing.
		"SUBST_SED.test+=\t-e s,@SH@,${PKGNAME},",        // Cannot be replaced since the variable name differs.
		"SUBST_SED.test+=\t-e s,@SH@,'\"'${SH:Q}'\"',g",  // Cannot be replaced since the double quotes are added.
		"SUBST_SED.test+=\t-e s",                         // Just to get 100% code coverage.
		"SUBST_SED.test+=\t-e s,@SH@,${SH:Q}",            // Just to get 100% code coverage.
		"SUBST_SED.test+=\t-e s,@SH@,${SH:Q}, # comment", // Just a note; not fixed because of the comment.
		"SUBST_SED.test+=\t-n s,@SH@,${SH:Q},",           // Just a note; not fixed because of the -n.
		"# end")

	mklines.Check()

	t.CheckOutputLines(
		"WARN: subst.mk:6: Please use ${SH:Q} instead of ${SH}.",
		"NOTE: subst.mk:6: The substitution command \"s,@SH@,${SH},g\" "+
			"can be replaced with \"SUBST_VARS.test= SH\".",
		"NOTE: subst.mk:7: The substitution command \"s,@SH@,${SH:Q},g\" "+
			"can be replaced with \"SUBST_VARS.test+= SH\".",
		"WARN: subst.mk:8: Please use ${SH:T:Q} instead of ${SH:T}.",
		"WARN: subst.mk:9: Please use ${SH:Q} instead of ${SH}.",
		"NOTE: subst.mk:9: The substitution command \"s,@SH@,${SH},\" "+
			"can be replaced with \"SUBST_VARS.test+= SH\".",
		"NOTE: subst.mk:10: The substitution command \"'s,@SH@,${SH},'\" "+
			"can be replaced with \"SUBST_VARS.test+= SH\".",
		"NOTE: subst.mk:11: The substitution command \"\\\"s,@SH@,${SH},\\\"\" "+
			"can be replaced with \"SUBST_VARS.test+= SH\".",
		"NOTE: subst.mk:12: The substitution command \"s,'@SH@','${SH}',\" "+
			"can be replaced with \"SUBST_VARS.test+= SH\".",
		"NOTE: subst.mk:13: Please always use \"-e\" in sed commands, "+
			"even if there is only one substitution.",
		"NOTE: subst.mk:13: The substitution command \"s,'@SH@','${SH}',\" "+
			"can be replaced with \"SUBST_VARS.test+= SH\".",
		"NOTE: subst.mk:18: The substitution command \"s,@SH@,${SH:Q},\" "+
			"can be replaced with \"SUBST_VARS.test+= SH\".",
		"NOTE: subst.mk:19: Please always use \"-e\" in sed commands, "+
			"even if there is only one substitution.",
		"NOTE: subst.mk:19: The substitution command \"s,@SH@,${SH:Q},\" "+
			"can be replaced with \"SUBST_VARS.test+= SH\".")

	t.SetUpCommandLine("--show-autofix")

	mklines.Check()

	t.CheckOutputLines(
		"NOTE: subst.mk:6: The substitution command \"s,@SH@,${SH},g\" "+
			"can be replaced with \"SUBST_VARS.test= SH\".",
		"AUTOFIX: subst.mk:6: Replacing \"SUBST_SED.test+=\\t-e s,@SH@,${SH},g\" "+
			"with \"SUBST_VARS.test=\\tSH\".",
		"NOTE: subst.mk:7: The substitution command \"s,@SH@,${SH:Q},g\" "+
			"can be replaced with \"SUBST_VARS.test+= SH\".",
		"AUTOFIX: subst.mk:7: Replacing \"SUBST_SED.test+=\\t-e s,@SH@,${SH:Q},g\" "+
			"with \"SUBST_VARS.test+=\\tSH\".",
		"NOTE: subst.mk:9: The substitution command \"s,@SH@,${SH},\" "+
			"can be replaced with \"SUBST_VARS.test+= SH\".",
		"AUTOFIX: subst.mk:9: Replacing \"SUBST_SED.test+=\\t-e s,@SH@,${SH},\" "+
			"with \"SUBST_VARS.test+=\\tSH\".",
		"NOTE: subst.mk:10: The substitution command \"'s,@SH@,${SH},'\" "+
			"can be replaced with \"SUBST_VARS.test+= SH\".",
		"AUTOFIX: subst.mk:10: Replacing \"SUBST_SED.test+=\\t-e 's,@SH@,${SH},'\" "+
			"with \"SUBST_VARS.test+=\\tSH\".",
		"NOTE: subst.mk:11: The substitution command \"\\\"s,@SH@,${SH},\\\"\" "+
			"can be replaced with \"SUBST_VARS.test+= SH\".",
		"AUTOFIX: subst.mk:11: Replacing \"SUBST_SED.test+=\\t-e \\\"s,@SH@,${SH},\\\"\" "+
			"with \"SUBST_VARS.test+=\\tSH\".",
		"NOTE: subst.mk:12: The substitution command \"s,'@SH@','${SH}',\" "+
			"can be replaced with \"SUBST_VARS.test+= SH\".",
		"AUTOFIX: subst.mk:12: Replacing \"SUBST_SED.test+=\\t-e s,'@SH@','${SH}',\" "+
			"with \"SUBST_VARS.test+=\\tSH\".")
}

// If the SUBST_CLASS identifier ends with a plus, the generated code must
// use the correct assignment operator and be nicely formatted.
func (s *Suite) Test_SubstContext_suggestSubstVars__plus(c *check.C) {
	t := s.Init(c)

	t.SetUpVartypes()
	t.SetUpTool("sh", "SH", AtRunTime)

	mklines := t.NewMkLines("subst.mk",
		MkCvsID,
		"",
		"SUBST_CLASSES+=\t\tgtk+",
		"SUBST_STAGE.gtk+ =\tpre-configure",
		"SUBST_FILES.gtk+ =\tfilename",
		"SUBST_SED.gtk+ +=\t-e s,@SH@,${SH:Q},g",
		"SUBST_SED.gtk+ +=\t-e s,@SH@,${SH:Q},g")

	mklines.Check()

	t.CheckOutputLines(
		"NOTE: subst.mk:6: The substitution command \"s,@SH@,${SH:Q},g\" "+
			"can be replaced with \"SUBST_VARS.gtk+ = SH\".",
		"NOTE: subst.mk:7: The substitution command \"s,@SH@,${SH:Q},g\" "+
			"can be replaced with \"SUBST_VARS.gtk+ += SH\".")

	t.SetUpCommandLine("--show-autofix")

	mklines.Check()

	t.CheckOutputLines(
		"NOTE: subst.mk:6: The substitution command \"s,@SH@,${SH:Q},g\" "+
			"can be replaced with \"SUBST_VARS.gtk+ = SH\".",
		"AUTOFIX: subst.mk:6: Replacing \"SUBST_SED.gtk+ +=\\t-e s,@SH@,${SH:Q},g\" "+
			"with \"SUBST_VARS.gtk+ =\\tSH\".",
		"NOTE: subst.mk:7: The substitution command \"s,@SH@,${SH:Q},g\" "+
			"can be replaced with \"SUBST_VARS.gtk+ += SH\".",
		"AUTOFIX: subst.mk:7: Replacing \"SUBST_SED.gtk+ +=\\t-e s,@SH@,${SH:Q},g\" "+
			"with \"SUBST_VARS.gtk+ +=\\tSH\".")
}

// The last of the SUBST_SED variables is 15 characters wide. When SUBST_SED
// is replaced with SUBST_VARS, this becomes 16 characters and therefore
// requires the whole paragraph to be indented by one more tab.
func (s *Suite) Test_SubstContext_suggestSubstVars__autofix_realign_paragraph(c *check.C) {
	t := s.Init(c)

	t.SetUpVartypes()
	t.Chdir(".")

	mklines := t.SetUpFileMkLines("subst.mk",
		MkCvsID,
		"",
		"SUBST_CLASSES+=\t\tpfx",
		"SUBST_STAGE.pfx=\tpre-configure",
		"SUBST_FILES.pfx=\tfilename",
		"SUBST_SED.pfx=\t\t-e s,@PREFIX@,${PREFIX},g",
		"SUBST_SED.pfx+=\t\t-e s,@PREFIX@,${PREFIX},g")

	mklines.Check()

	t.CheckOutputLines(
		"NOTE: subst.mk:6: The substitution command \"s,@PREFIX@,${PREFIX},g\" "+
			"can be replaced with \"SUBST_VARS.pfx= PREFIX\".",
		"NOTE: subst.mk:7: The substitution command \"s,@PREFIX@,${PREFIX},g\" "+
			"can be replaced with \"SUBST_VARS.pfx+= PREFIX\".")

	t.SetUpCommandLine("--autofix")

	mklines.Check()

	t.CheckOutputLines(
		"AUTOFIX: subst.mk:6: Replacing \"SUBST_SED.pfx=\\t\\t-e s,@PREFIX@,${PREFIX},g\" "+
			"with \"SUBST_VARS.pfx=\\t\\tPREFIX\".",
		"AUTOFIX: subst.mk:7: Replacing \"SUBST_SED.pfx+=\\t\\t-e s,@PREFIX@,${PREFIX},g\" "+
			"with \"SUBST_VARS.pfx+=\\tPREFIX\".")

	t.CheckFileLinesDetab("subst.mk",
		MkCvsID,
		"",
		"SUBST_CLASSES+=         pfx",
		"SUBST_STAGE.pfx=        pre-configure",
		"SUBST_FILES.pfx=        filename",
		"SUBST_VARS.pfx=         PREFIX",
		"SUBST_VARS.pfx+=        PREFIX")
}

func (s *Suite) Test_SubstContext_suggestSubstVars__autofix_plus_sed(c *check.C) {
	t := s.Init(c)

	t.SetUpVartypes()
	t.Chdir(".")

	mklines := t.SetUpFileMkLines("subst.mk",
		MkCvsID,
		"",
		"SUBST_CLASSES+=\t\tpfx",
		"SUBST_STAGE.pfx=\tpre-configure",
		"SUBST_FILES.pfx=\tfilename",
		"SUBST_SED.pfx=\t\t-e s,@PREFIX@,${PREFIX},g",
		"SUBST_SED.pfx+=\t\t-e s,@PREFIX@,other,g")

	mklines.Check()

	t.CheckOutputLines(
		"NOTE: subst.mk:6: The substitution command \"s,@PREFIX@,${PREFIX},g\" " +
			"can be replaced with \"SUBST_VARS.pfx= PREFIX\".")

	t.SetUpCommandLine("-Wall", "--autofix")

	mklines.Check()

	t.CheckOutputLines(
		"AUTOFIX: subst.mk:6: Replacing \"SUBST_SED.pfx=\\t\\t-e s,@PREFIX@,${PREFIX},g\" " +
			"with \"SUBST_VARS.pfx=\\t\\tPREFIX\".")

	t.CheckFileLinesDetab("subst.mk",
		MkCvsID,
		"",
		"SUBST_CLASSES+=         pfx",
		"SUBST_STAGE.pfx=        pre-configure",
		"SUBST_FILES.pfx=        filename",
		"SUBST_VARS.pfx=         PREFIX",
		// TODO: If this subst class is used nowhere else, pkglint could
		//  replace this += with a simple =.
		"SUBST_SED.pfx+=         -e s,@PREFIX@,other,g")
}

func (s *Suite) Test_SubstContext_suggestSubstVars__autofix_plus_vars(c *check.C) {
	t := s.Init(c)

	t.SetUpCommandLine("-Wall", "--autofix")
	t.SetUpVartypes()
	t.Chdir(".")

	mklines := t.SetUpFileMkLines("subst.mk",
		MkCvsID,
		"",
		"SUBST_CLASSES+=\tid",
		"SUBST_STAGE.id=\tpre-configure",
		"SUBST_FILES.id=\tfilename",
		"SUBST_SED.id=\t-e s,@PREFIX@,${PREFIX},g",
		"SUBST_VARS.id=\tPKGMANDIR")

	mklines.Check()

	t.CheckOutputLines(
		"AUTOFIX: subst.mk:6: Replacing \"SUBST_SED.id=\\t-e s,@PREFIX@,${PREFIX},g\" " +
			"with \"SUBST_VARS.id=\\tPREFIX\".")

	t.CheckFileLinesDetab("subst.mk",
		MkCvsID,
		"",
		"SUBST_CLASSES+= id",
		"SUBST_STAGE.id= pre-configure",
		"SUBST_FILES.id= filename",
		"SUBST_VARS.id=  PREFIX",
		// FIXME: This must be += instead of = since the previous line already uses =.
		//  Luckily the check for redundant assignments catches this already.
		"SUBST_VARS.id=  PKGMANDIR")
}

func (s *Suite) Test_SubstContext_suggestSubstVars__autofix_indentation(c *check.C) {
	t := s.Init(c)

	t.SetUpCommandLine("-Wall", "--autofix")
	t.SetUpVartypes()
	t.Chdir(".")

	mklines := t.SetUpFileMkLines("subst.mk",
		MkCvsID,
		"",
		"SUBST_CLASSES+=\t\t\tfix-paths",
		"SUBST_STAGE.fix-paths=\t\tpre-configure",
		"SUBST_MESSAGE.fix-paths=\tMessage",
		"SUBST_FILES.fix-paths=\t\tfilename",
		"SUBST_SED.fix-paths=\t\t-e s,@PREFIX@,${PREFIX},g")

	mklines.Check()

	t.CheckOutputLines(
		"AUTOFIX: subst.mk:7: Replacing \"SUBST_SED.fix-paths=\\t\\t-e s,@PREFIX@,${PREFIX},g\" " +
			"with \"SUBST_VARS.fix-paths=\\t\\tPREFIX\".")

	t.CheckFileLinesDetab("subst.mk",
		MkCvsID,
		"",
		"SUBST_CLASSES+=                 fix-paths",
		"SUBST_STAGE.fix-paths=          pre-configure",
		"SUBST_MESSAGE.fix-paths=        Message",
		"SUBST_FILES.fix-paths=          filename",
		"SUBST_VARS.fix-paths=           PREFIX")
}

func (s *Suite) Test_SubstContext_extractVarname(c *check.C) {
	t := s.Init(c)

	test := func(input, expected string) {
		t.CheckEquals((*SubstContext).extractVarname(nil, input), expected)
	}

	// A simple variable name.
	test("s,@VAR@,${VAR},", "VAR")

	// A parameterized variable name.
	test("s,@VAR.param@,${VAR.param},", "VAR.param")

	// Only substitution commands can be replaced with SUBST_VARS.
	test("/pattern/d", "")

	// An incomplete substitution command.
	test("s", "")

	// Wrong placeholder character, only @ works.
	test("s,!VAR!,${VAR},", "")

	// The placeholder must have exactly 1 @ on each side.
	test("s,@@VAR@@,${VAR},", "")

	// Malformed because the comma is the separator.
	test("s,@VAR,VAR@,${VAR},", "")

	// The replacement pattern is not a simple variable name enclosed in @.
	test("s,@VAR!VAR@,${VAR},", "")

	// The replacement may only contain the :Q modifier.
	test("s,@VAR@,${VAR:Mpattern},", "")

	// The :Q modifier is allowed in the replacement.
	test("s,@VAR@,${VAR:Q},", "VAR")

	// The replacement may contain the :Q modifier only once.
	test("s,@VAR@,${VAR:Q:Q},", "")

	// The replacement must be a plain variable expression, without prefix.
	test("s,@VAR@,prefix${VAR},", "")

	// The replacement must be a plain variable expression, without suffix.
	test("s,@VAR@,${VAR}suffix,", "")
}
