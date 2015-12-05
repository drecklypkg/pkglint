package main

import (
	check "gopkg.in/check.v1"
)

func (s *Suite) TestChecklineMkVartype_SimpleType(c *check.C) {
	s.UseCommandLine(c, "-Wtypes", "-Dunchecked")
	G.globalData.InitVartypes()
	ml := NewMkLine(NewLine("fname", "1", "COMMENT=\tA nice package", nil))

	vartype1 := G.globalData.vartypes["COMMENT"]
	c.Assert(vartype1, check.NotNil)
	c.Check(vartype1.guessed, equals, guNotGuessed)

	vartype := getVariableType(ml.line, "COMMENT")

	c.Assert(vartype, check.NotNil)
	c.Check(vartype.checker.name, equals, "Comment")
	c.Check(vartype.guessed, equals, guNotGuessed)
	c.Check(vartype.kindOfList, equals, lkNone)

	ml.checkVartype("COMMENT", "=", "A nice package", "")

	c.Check(s.Stdout(), equals, "WARN: fname:1: COMMENT should not begin with \"A\".\n")
}

func (s *Suite) TestChecklineMkVartype(c *check.C) {
	G.globalData.InitVartypes()
	ml := NewMkLine(NewLine("fname", "1", "DISTNAME=gcc-${GCC_VERSION}", nil))

	ml.checkVartype("DISTNAME", "=", "gcc-${GCC_VERSION}", "")
}

func (s *Suite) TestChecklineMkVaralign(c *check.C) {
	s.UseCommandLine(c, "-Wspace", "-f")
	lines := s.NewLines("file.mk",
		"VAR=   value",    // Indentation 7, fixed to 8.
		"VAR=    value",   // Indentation 8, fixed to 8.
		"VAR=     value",  // Indentation 9, fixed to 16.
		"VAR= \tvalue",    // Mixed indentation 8, fixed to 8.
		"VAR=   \tvalue",  // Mixed indentation 8, fixed to 8.
		"VAR=    \tvalue", // Mixed indentation 16, fixed to 16.
		"VAR=\tvalue")     // Already aligned with tabs only, left unchanged.

	for _, line := range lines {
		NewMkLine(line).checkVaralign()
	}

	c.Check(lines[0].changed, equals, true)
	c.Check(lines[0].rawLines()[0].String(), equals, "1:VAR=\tvalue\n")
	c.Check(lines[1].changed, equals, true)
	c.Check(lines[1].rawLines()[0].String(), equals, "2:VAR=\tvalue\n")
	c.Check(lines[2].changed, equals, true)
	c.Check(lines[2].rawLines()[0].String(), equals, "3:VAR=\t\tvalue\n")
	c.Check(lines[3].changed, equals, true)
	c.Check(lines[3].rawLines()[0].String(), equals, "4:VAR=\tvalue\n")
	c.Check(lines[4].changed, equals, true)
	c.Check(lines[4].rawLines()[0].String(), equals, "5:VAR=\tvalue\n")
	c.Check(lines[5].changed, equals, true)
	c.Check(lines[5].rawLines()[0].String(), equals, "6:VAR=\t\tvalue\n")
	c.Check(lines[6].changed, equals, false)
	c.Check(lines[6].rawLines()[0].String(), equals, "7:VAR=\tvalue\n")
	c.Check(s.Output(), equals, ""+
		"NOTE: file.mk:1: Alignment of variable values should be done with tabs, not spaces.\n"+
		"NOTE: file.mk:1: Autofix: replacing \"VAR=   \" with \"VAR=\\t\".\n"+
		"NOTE: file.mk:2: Alignment of variable values should be done with tabs, not spaces.\n"+
		"NOTE: file.mk:2: Autofix: replacing \"VAR=    \" with \"VAR=\\t\".\n"+
		"NOTE: file.mk:3: Alignment of variable values should be done with tabs, not spaces.\n"+
		"NOTE: file.mk:3: Autofix: replacing \"VAR=     \" with \"VAR=\\t\\t\".\n"+
		"NOTE: file.mk:4: Alignment of variable values should be done with tabs, not spaces.\n"+
		"NOTE: file.mk:4: Autofix: replacing \"VAR= \\t\" with \"VAR=\\t\".\n"+
		"NOTE: file.mk:5: Alignment of variable values should be done with tabs, not spaces.\n"+
		"NOTE: file.mk:5: Autofix: replacing \"VAR=   \\t\" with \"VAR=\\t\".\n"+
		"NOTE: file.mk:6: Alignment of variable values should be done with tabs, not spaces.\n"+
		"NOTE: file.mk:6: Autofix: replacing \"VAR=    \\t\" with \"VAR=\\t\\t\".\n")
	c.Check(tabLength("VAR=    \t"), equals, 16)
}

func (s *Suite) TestMkLine_CheckAbsolutePathname(c *check.C) {
	ml := NewMkLine(NewLine("Makefile", "1", "# dummy", nil))

	ml.checkAbsolutePathname("bindir=/bin")
	ml.checkAbsolutePathname("bindir=/../lib")

	c.Check(s.Output(), equals, "WARN: Makefile:1: Found absolute pathname: /bin\n")
}
