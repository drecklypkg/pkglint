package main

import (
	check "gopkg.in/check.v1"
)

func (s *Suite) TestSplitIntoShellwords_LineContinuation(c *check.C) {
	line := NewLine("fname", "1", "dummy", nil)

	words, rest := splitIntoShellwords(line, "if true; then \\")

	c.Check(words, check.DeepEquals, []string{"if", "true", ";", "then"})
	c.Check(rest, equals, "\\")

	words, rest = splitIntoShellwords(line, "pax -s /.*~$$//g")

	c.Check(words, check.DeepEquals, []string{"pax", "-s", "/.*~$$//g"})
	c.Check(rest, equals, "")
}

func (s *Suite) TestChecklineMkShelltext(c *check.C) {
	s.UseCommandLine(c, "-Wall")
	G.mkContext = newMkContext()
	msline := NewMkShellLine(NewLine("fname", "1", "dummy", nil))

	msline.checklineMkShelltext("@# Comment")

	c.Check(s.Output(), equals, "")

	msline.checklineMkShelltext("uname=`uname`; echo $$uname")

	c.Check(s.Output(), equals, ""+
		"WARN: fname:1: Unknown shell command \"uname\".\n"+
		"WARN: fname:1: Please switch to \"set -e\" mode before using a semicolon to separate commands.\n"+
		"WARN: fname:1: Unknown shell command \"echo\".\n"+
		"WARN: fname:1: Unquoted shell variable \"uname\".\n")

	// The following test case goes beyond the limits of the current shell parser.

	// foobar="`echo \"foo   bar\"`"
	msline.checklineMkShelltext("foobar=\"`echo \\\"foo   bar\\\"`\"")

	c.Check(s.Output(), equals, ""+
		"WARN: fname:1: Backslashes should be doubled inside backticks.\n"+
		"WARN: fname:1: Double quotes inside backticks inside double quotes are error prone.\n"+
		"WARN: fname:1: Backslashes should be doubled inside backticks.\n"+
		"WARN: fname:1: Double quotes inside backticks inside double quotes are error prone.\n"+
		"WARN: fname:1: Unknown shell command \"echo\".\n"+
		"ERROR: fname:1: Internal pkglint error: checklineMkShellword state=plain, rest=\"\\\\foo\", shellword=\"\\\\foo\"\n"+
		"ERROR: fname:1: Internal pkglint error: checklineMkShelltext state=continuation rest=\"\\\\\" shellword=\"echo \\\\foo   bar\\\\\"\n")

	G.globalData.tools = map[string]bool{"echo": true}
	G.globalData.predefinedTools = map[string]bool{"echo": true}
	G.mkContext = newMkContext()
	G.globalData.InitVartypes()

	msline.checklineMkShelltext("echo ${PKGNAME:Q}") // VUC_SHW_PLAIN

	c.Check(s.Output(), equals, ""+
		"WARN: fname:1: PKGNAME may not be used in this file.\n"+
		"NOTE: fname:1: The :Q operator isn't necessary for ${PKGNAME} here.\n")

	msline.checklineMkShelltext("echo \"${CFLAGS:Q}\"") // VUC_SHW_DQUOT

	c.Check(s.Output(), equals, ""+
		"WARN: fname:1: Please don't use the :Q operator in double quotes.\n"+
		"WARN: fname:1: CFLAGS may not be used in this file.\n"+
		"WARN: fname:1: Please use ${CFLAGS:M*:Q} instead of ${CFLAGS:Q} and make sure the variable appears outside of any quoting characters.\n")

	msline.checklineMkShelltext("echo '${COMMENT:Q}'") // VUC_SHW_SQUOT

	c.Check(s.Output(), equals, "WARN: fname:1: COMMENT may not be used in this file.\n")
	
	msline.checklineMkShelltext("echo $$@") 

	c.Check(s.Output(), equals, "WARN: fname:1: The $@ shell variable should only be used in double quotes.\n")
	
	msline.checklineMkShelltext("echo \"$$\"") // As seen by make(1); the shell sees: echo $
	
	c.Check(s.Output(), equals, "WARN: fname:1: Unquoted $ or strange shell variable found.\n")
	
	msline.checklineMkShelltext("echo \"\\n\"") // As seen by make(1); the shell sees: echo "\n"
	
	c.Check(s.Output(), equals, "WARN: fname:1: Please use \"\\\\n\" instead of \"\\n\".\n")
}

func (s *Suite) TestChecklineMkShellword(c *check.C) {
	s.UseCommandLine(c, "-Wall")
	G.globalData.InitVartypes()
	line := NewLine("fname", "1", "dummy", nil)

	c.Check(matches("${list}", `^`+reVarnameDirect+`$`), equals, false)

	checklineMkShellword(line, "${${list}}", false)

	c.Check(s.Output(), equals, "")

	checklineMkShellword(line, "\"$@\"", false)

	c.Check(s.Output(), equals, "WARN: fname:1: Please use \"${.TARGET}\" instead of \"$@\".\n")
}

func (s *Suite) TestShelltextContext_CheckCommandStart(c *check.C) {
	s.UseCommandLine(c, "-Wall")
	G.globalData.tools = map[string]bool{"echo": true}
	G.globalData.vartools = map[string]string{"echo": "ECHO"}
	G.globalData.toolsVarRequired = map[string]bool{"echo": true}
	G.mkContext = newMkContext()
	line := NewLine("fname", "3", "dummy", nil)

	checklineMkShellcmd(line, "echo \"hello, world\"")

	c.Check(s.Output(), equals, ""+
		"WARN: fname:3: The \"echo\" tool is used but not added to USE_TOOLS.\n"+
		"WARN: fname:3: Please use \"${ECHO}\" instead of \"echo\".\n")
}

func (s *Suite) TestMkShellLine_checklineMkShelltext(c *check.C) {

	shline := NewMkShellLine(s.DummyLine())

	shline.checklineMkShelltext("for f in *.pl; do ${SED} s,@PREFIX@,${PREFIX}, < $f > $f.tmp && ${MV} $f.tmp $f; done")

	c.Check(s.Output(), equals, "NOTE: fname:1: Please use the SUBST framework instead of ${SED} and ${MV}.\n")

	shline.checklineMkShelltext("install -c manpage.1 ${PREFIX}/man/man1/manpage.1")

	c.Check(s.Output(), equals, "WARN: fname:1: Please use ${PKGMANDIR} instead of \"man\".\n")

	shline.checklineMkShelltext("cp init-script ${PREFIX}/etc/rc.d/service")

	c.Check(s.Output(), equals, "WARN: fname:1: Please use the RCD_SCRIPTS mechanism to install rc.d scripts automatically to ${RCD_SCRIPTS_EXAMPLEDIR}.\n")
}

func (s *Suite) TestMkShellLine_checkCommandUse(c *check.C) {
	G.mkContext = newMkContext()
	G.mkContext.target = "do-install"

	shline := NewMkShellLine(s.DummyLine())

	shline.checkCommandUse("sed")

	c.Check(s.Output(), equals, "WARN: fname:1: The shell command \"sed\" should not be used in the install phase.\n")

	shline.checkCommandUse("cp")

	c.Check(s.Output(), equals, "WARN: fname:1: ${CP} should not be used to install files.\n")
}
