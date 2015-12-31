package main

import (
	check "gopkg.in/check.v1"
)

func (s *Suite) TestCheckdirCategory_TotallyBroken(c *check.C) {
	G.globalData.InitVartypes()
	s.CreateTmpFile(c, "archivers/Makefile", ""+
		"# $\n"+
		"SUBDIR+=pkg1\n"+
		"SUBDIR+=\u0020aaaaa\n"+
		"SUBDIR-=unknown #doesn’t work\n"+
		"\n"+
		".include \"../mk/category.mk\"\n")

	G.CurrentDir = s.tmpdir + "/archivers"
	CheckdirCategory()

	c.Check(s.OutputCleanTmpdir(), equals, ""+
		"ERROR: ~/archivers/Makefile:1: Expected \"# $"+"NetBSD$\".\n"+
		"WARN: ~/archivers/Makefile:4: SUBDIR- is defined but not used. Spelling mistake?\n"+
		"ERROR: ~/archivers/Makefile:6: \"../mk/category.mk\" does not exist.\n"+
		"ERROR: ~/archivers/Makefile:2: COMMENT= line expected.\n"+
		"WARN: ~/archivers/Makefile:2: Indentation should be a single tab character.\n"+
		"WARN: ~/archivers/Makefile:3: Indentation should be a single tab character.\n"+
		"WARN: ~/archivers/Makefile:3: \"aaaaa\" should come before \"pkg1\".\n"+
		"ERROR: ~/archivers/Makefile:4: SUBDIR+= line or empty line expected.\n"+
		"ERROR: ~/archivers/Makefile:2: \"pkg1\" exists in the Makefile, but not in the file system.\n"+
		"ERROR: ~/archivers/Makefile:3: \"aaaaa\" exists in the Makefile, but not in the file system.\n"+
		"WARN: ~/archivers/Makefile:4: This line should contain the following text: .include \"../mk/misc/category.mk\"\n"+
		"ERROR: ~/archivers/Makefile:4: The file should end here.\n")
}
