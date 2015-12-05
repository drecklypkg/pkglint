package main

import (
	check "gopkg.in/check.v1"
)

func (s *Suite) TestChecklinesDistinfo(c *check.C) {
	s.CreateTmpFile(c, "patches/patch-aa", ""+
		"$"+"NetBSD$ line is ignored\n"+
		"patch contents\n")
	G.currentDir = s.tmpdir

	checklinesDistinfo(s.NewLines("distinfo",
		"should be the RCS ID",
		"should be empty",
		"MD5 (distfile.tar.gz) = 12345678901234567890123456789012",
		"SHA1 (distfile.tar.gz) = 1234567890123456789012345678901234567890",
		"SHA1 (patch-aa) = 6b98dd609f85a9eb9c4c1e4e7055a6aaa62b7cc7"))

	c.Check(s.Output(), equals, ""+
		"ERROR: distinfo:1: Expected \"$"+"NetBSD$\".\n"+
		"NOTE: distinfo:2: Empty line expected.\n"+
		"ERROR: distinfo:5: Expected SHA1, RMD160, SHA512, Size checksums for \"distfile.tar.gz\", got MD5, SHA1.\n")
}

func (s *Suite) TestChecklinesDistinfo_GlobalHashMismatch(c *check.C) {
	otherLine := NewLine("other/distinfo", "7", "dummy", nil)
	G.ipcDistinfo = make(map[string]*Hash)
	G.ipcDistinfo["SHA512:pkgname-1.0.tar.gz"] = &Hash{"asdfasdf", otherLine}

	checklinesDistinfo(s.NewLines("distinfo",
		"$"+"NetBSD$",
		"",
		"SHA512 (pkgname-1.0.tar.gz) = 12341234"))

	c.Check(s.Output(), equals, ""+
		"ERROR: distinfo:3: The hash SHA512 for pkgname-1.0.tar.gz is 12341234, ...\n"+
		"ERROR: other/distinfo:7: ... which differs from asdfasdf.\n"+
		"ERROR: distinfo:EOF: Expected SHA1, RMD160, SHA512, Size checksums for \"pkgname-1.0.tar.gz\", got SHA512.\n")
}

func (s *Suite) TestChecklinesDistinfo_UncommittedPatch(c *check.C) {
	s.CreateTmpFile(c, "patches/patch-aa", ""+
		"$"+"NetBSD$\n"+
		"\n"+
		"--- oldfile\n"+
		"+++ newfile\n"+
		"@@ -1,1 +1,1 @@\n"+
		"-old\n"+
		"+new\n")
	s.CreateTmpFile(c, "CVS/Entries",
		"/distinfo/...\n")
	G.currentDir = s.tmpdir

	checklinesDistinfo(s.NewLines(s.tmpdir+"/distinfo",
		"$"+"NetBSD$",
		"",
		"SHA1 (patch-aa) = 5ad1fb9b3c328fff5caa1a23e8f330e707dd50c0"))

	c.Check(s.Output(), equals, "WARN: "+s.tmpdir+"/distinfo:3: patches/patch-aa is registered in distinfo but not added to CVS.\n")
}

func (s *Suite) TestChecklinesDistinfo_UnrecordedPatches(c *check.C) {
	s.CreateTmpFile(c, "patches/CVS/Entries", "")
	s.CreateTmpFile(c, "patches/patch-aa", "")
	s.CreateTmpFile(c, "patches/patch-src-Makefile", "")
	G.currentDir = s.tmpdir

	checklinesDistinfo(s.NewLines(s.tmpdir+"/distinfo",
		"$"+"NetBSD$",
		"",
		"SHA1 (distfile.tar.gz) = ...",
		"RMD160 (distfile.tar.gz) = ...",
		"SHA512 (distfile.tar.gz) = ...",
		"Size (distfile.tar.gz) = 1024 bytes"))

	c.Check(s.Output(), equals, sprintf(""+
		"ERROR: %[1]s: patch \"patches/patch-aa\" is not recorded. Run \"%s makepatchsum\".\n"+
		"ERROR: %[1]s: patch \"patches/patch-src-Makefile\" is not recorded. Run \"%s makepatchsum\".\n",
		s.tmpdir+"/distinfo", confMake))
}
