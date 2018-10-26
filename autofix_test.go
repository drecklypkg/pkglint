package main

import (
	"gopkg.in/check.v1"
	"os"
	"runtime"
	"strings"
)

func (s *Suite) Test_Autofix_ReplaceAfter__autofix(c *check.C) {
	t := s.Init(c)

	t.SetupCommandLine("--autofix", "--source")
	mklines := t.SetupFileMkLines("Makefile",
		"# line 1 \\",
		"continuation 1 \\",
		"continuation 2")

	fix := mklines.lines.Lines[0].Autofix()
	fix.Warnf("N should be replaced with V.")
	fix.ReplaceAfter("", "n", "v")
	fix.Apply()

	t.CheckOutputLines(
		"AUTOFIX: ~/Makefile:1: Replacing \"n\" with \"v\".",
		"-\t# line 1 \\",
		"+\t# live 1 \\",
		">\tcontinuation 1 \\",
		">\tcontinuation 2")
}

func (s *Suite) Test_Autofix_ReplaceRegex__show_autofix(c *check.C) {
	t := s.Init(c)

	t.SetupCommandLine("--show-autofix")
	lines := t.SetupFileLines("Makefile",
		"line1",
		"line2",
		"line3")

	fix := lines.Lines[1].Autofix()
	fix.Warnf("Something's wrong here.")
	fix.ReplaceRegex(`.`, "X", -1)
	fix.Apply()
	SaveAutofixChanges(lines)

	c.Check(lines.Lines[1].raw[0].textnl, equals, "XXXXX\n")
	t.CheckFileLines("Makefile",
		"line1",
		"line2",
		"line3")
	t.CheckOutputLines(
		"WARN: ~/Makefile:2: Something's wrong here.",
		"AUTOFIX: ~/Makefile:2: Replacing \"l\" with \"X\".",
		"AUTOFIX: ~/Makefile:2: Replacing \"i\" with \"X\".",
		"AUTOFIX: ~/Makefile:2: Replacing \"n\" with \"X\".",
		"AUTOFIX: ~/Makefile:2: Replacing \"e\" with \"X\".",
		"AUTOFIX: ~/Makefile:2: Replacing \"2\" with \"X\".")
}

func (s *Suite) Test_Autofix_ReplaceRegex__autofix(c *check.C) {
	t := s.Init(c)

	t.SetupCommandLine("--autofix", "--source")
	lines := t.SetupFileLines("Makefile",
		"line1",
		"line2",
		"line3")

	fix := lines.Lines[1].Autofix()
	fix.Warnf("Something's wrong here.")
	fix.ReplaceRegex(`.`, "X", 3)
	fix.Apply()

	t.CheckOutputLines(
		"AUTOFIX: ~/Makefile:2: Replacing \"l\" with \"X\".",
		"AUTOFIX: ~/Makefile:2: Replacing \"i\" with \"X\".",
		"AUTOFIX: ~/Makefile:2: Replacing \"n\" with \"X\".",
		"-\tline2",
		"+\tXXXe2")

	fix.Warnf("Use Y instead of X.")
	fix.Replace("X", "Y")
	fix.Apply()

	t.CheckOutputLines(
		"",
		"AUTOFIX: ~/Makefile:2: Replacing \"X\" with \"Y\".",
		"-\tline2",
		"+\tYXXe2")

	SaveAutofixChanges(lines)

	t.CheckFileLines("Makefile",
		"line1",
		"YXXe2",
		"line3")
}

func (s *Suite) Test_Autofix_ReplaceRegex__show_autofix_and_source(c *check.C) {
	t := s.Init(c)

	t.SetupCommandLine("--show-autofix", "--source")
	lines := t.SetupFileLines("Makefile",
		"line1",
		"line2",
		"line3")

	fix := lines.Lines[1].Autofix()
	fix.Warnf("Something's wrong here.")
	fix.ReplaceRegex(`.`, "X", -1)
	fix.Apply()

	fix.Warnf("Use Y instead of X.")
	fix.Replace("X", "Y")
	fix.Apply()

	SaveAutofixChanges(lines)

	t.CheckOutputLines(
		"WARN: ~/Makefile:2: Something's wrong here.",
		"AUTOFIX: ~/Makefile:2: Replacing \"l\" with \"X\".",
		"AUTOFIX: ~/Makefile:2: Replacing \"i\" with \"X\".",
		"AUTOFIX: ~/Makefile:2: Replacing \"n\" with \"X\".",
		"AUTOFIX: ~/Makefile:2: Replacing \"e\" with \"X\".",
		"AUTOFIX: ~/Makefile:2: Replacing \"2\" with \"X\".",
		"-\tline2",
		"+\tXXXXX",
		"",
		"WARN: ~/Makefile:2: Use Y instead of X.",
		"AUTOFIX: ~/Makefile:2: Replacing \"X\" with \"Y\".",
		"-\tline2",
		"+\tYXXXX")
}

func (s *Suite) Test_SaveAutofixChanges(c *check.C) {
	t := s.Init(c)

	t.SetupCommandLine("--autofix")
	t.CreateFileLines("category/basename/Makefile",
		"line1 := value1",
		"line2 := value2",
		"line3 := value3")
	pkg := NewPackage(t.File("category/basename"))
	G.Pkg = pkg
	mklines := pkg.loadPackageMakefile()
	G.Pkg = nil

	fix := mklines.mklines[1].Autofix()
	fix.Warnf("Something's wrong here.")
	fix.ReplaceRegex(`...`, "XXX", -1)
	fix.Apply()

	fix = mklines.mklines[2].Autofix()
	fix.Warnf("Something's wrong here.")
	fix.ReplaceRegex(`...`, "XXX", 1)
	fix.Apply()

	SaveAutofixChanges(mklines.lines)

	t.CheckOutputLines(
		"AUTOFIX: ~/category/basename/Makefile:2: Replacing \"lin\" with \"XXX\".",
		"AUTOFIX: ~/category/basename/Makefile:2: Replacing \"e2 \" with \"XXX\".",
		"AUTOFIX: ~/category/basename/Makefile:2: Replacing \":= \" with \"XXX\".",
		"AUTOFIX: ~/category/basename/Makefile:2: Replacing \"val\" with \"XXX\".",
		"AUTOFIX: ~/category/basename/Makefile:2: Replacing \"ue2\" with \"XXX\".",
		"AUTOFIX: ~/category/basename/Makefile:3: Replacing \"lin\" with \"XXX\".")
	t.CheckFileLines("category/basename/Makefile",
		"line1 := value1",
		"XXXXXXXXXXXXXXX",
		"XXXe3 := value3")
}

func (s *Suite) Test_SaveAutofixChanges__no_changes_necessary(c *check.C) {
	t := s.Init(c)

	t.SetupCommandLine("--autofix")
	lines := t.SetupFileLines("DESCR",
		"Line 1",
		"Line 2")

	fix := lines.Lines[0].Autofix()
	fix.Warnf("Dummy warning.")
	fix.Replace("X", "Y")
	fix.Apply()

	// Since nothing has been effectively changed,
	// nothing needs to be saved.
	SaveAutofixChanges(lines)

	// And therefore, no AUTOFIX action must appear in the log.
	t.CheckOutputEmpty()
}

func (s *Suite) Test_Autofix__multiple_modifications(c *check.C) {
	t := s.Init(c)

	t.SetupCommandLine("--show-autofix", "--explain")

	line := t.NewLine("fname", 1, "original")

	c.Check(line.autofix, check.IsNil)
	c.Check(line.raw, check.DeepEquals, t.NewRawLines(1, "original\n"))

	{
		fix := line.Autofix()
		fix.Warnf(SilentMagicDiagnostic)
		fix.ReplaceRegex(`(.)(.*)(.)`, "lriginao", 1) // XXX: the replacement should be "$3$2$1"
		fix.Apply()
	}

	c.Check(line.autofix, check.NotNil)
	c.Check(line.raw, check.DeepEquals, t.NewRawLines(1, "original\n", "lriginao\n"))
	t.CheckOutputLines(
		"AUTOFIX: fname:1: Replacing \"original\" with \"lriginao\".")

	{
		fix := line.Autofix()
		fix.Warnf(SilentMagicDiagnostic)
		fix.Replace("i", "u")
		fix.Apply()
	}

	c.Check(line.autofix, check.NotNil)
	c.Check(line.raw, check.DeepEquals, t.NewRawLines(1, "original\n", "lruginao\n"))
	c.Check(line.raw[0].textnl, equals, "lruginao\n")
	t.CheckOutputLines(
		"AUTOFIX: fname:1: Replacing \"i\" with \"u\".")

	{
		fix := line.Autofix()
		fix.Warnf(SilentMagicDiagnostic)
		fix.Replace("lruginao", "middle")
		fix.Apply()
	}

	c.Check(line.autofix, check.NotNil)
	c.Check(line.raw, check.DeepEquals, t.NewRawLines(1, "original\n", "middle\n"))
	c.Check(line.raw[0].textnl, equals, "middle\n")
	t.CheckOutputLines(
		"AUTOFIX: fname:1: Replacing \"lruginao\" with \"middle\".")

	{
		fix := line.Autofix()
		fix.Warnf(SilentMagicDiagnostic)
		fix.InsertBefore("before")
		fix.Apply()

		fix.Warnf(SilentMagicDiagnostic)
		fix.InsertBefore("between before and middle")
		fix.Apply()

		fix.Warnf(SilentMagicDiagnostic)
		fix.InsertAfter("between middle and after")
		fix.Apply()

		fix.Notef("This diagnostic is necessary for the following explanation.")
		fix.Explain(
			"When inserting multiple lines, Apply must be called in-between.",
			"Otherwise the changes are not described to the human reader.")
		fix.InsertAfter("after")
		fix.Apply()
	}

	c.Check(line.autofix.linesBefore, check.DeepEquals, []string{
		"before\n",
		"between before and middle\n"})
	c.Check(line.autofix.lines[0].textnl, equals, "middle\n")
	c.Check(line.autofix.linesAfter, deepEquals, []string{
		"between middle and after\n",
		"after\n"})
	t.CheckOutputLines(
		"AUTOFIX: fname:1: Inserting a line \"before\" before this line.",
		"AUTOFIX: fname:1: Inserting a line \"between before and middle\" before this line.",
		"AUTOFIX: fname:1: Inserting a line \"between middle and after\" after this line.",
		"NOTE: fname:1: This diagnostic is necessary for the following explanation.",
		"AUTOFIX: fname:1: Inserting a line \"after\" after this line.",
		"",
		"\tWhen inserting multiple lines, Apply must be called in-between.",
		"\tOtherwise the changes are not described to the human reader.",
		"")

	{
		fix := line.Autofix()
		fix.Warnf(SilentMagicDiagnostic)
		fix.Delete()
		fix.Apply()
	}

	c.Check(line.autofix.linesBefore, check.DeepEquals, []string{
		"before\n",
		"between before and middle\n"})
	c.Check(line.autofix.lines[0].textnl, equals, "")
	c.Check(line.autofix.linesAfter, deepEquals, []string{
		"between middle and after\n",
		"after\n"})
	t.CheckOutputLines(
		"AUTOFIX: fname:1: Deleting this line.")
}

func (s *Suite) Test_Autofix__show_autofix_and_source(c *check.C) {
	t := s.Init(c)

	t.SetupCommandLine("--show-autofix", "--source")
	mklines := t.SetupFileMkLines("Makefile",
		MkRcsID,
		"# before \\",
		"The old song \\",
		"after")
	line := mklines.lines.Lines[1]

	{
		fix := line.Autofix()
		fix.Warnf("Using \"old\" is deprecated.")
		fix.Replace("old", "new")
		fix.Apply()
	}

	t.CheckOutputLines(
		"WARN: ~/Makefile:2--4: Using \"old\" is deprecated.",
		"AUTOFIX: ~/Makefile:3: Replacing \"old\" with \"new\".",
		">\t# before \\",
		"-\tThe old song \\",
		"+\tThe new song \\",
		">\tafter")
}

func (s *Suite) Test_Autofix_InsertBefore(c *check.C) {
	t := s.Init(c)

	t.SetupCommandLine("--show-autofix", "--source")
	line := t.NewLine("Makefile", 30, "original")

	fix := line.Autofix()
	fix.Warnf("Dummy.")
	fix.InsertBefore("inserted")
	fix.Apply()

	t.CheckOutputLines(
		"WARN: Makefile:30: Dummy.",
		"AUTOFIX: Makefile:30: Inserting a line \"inserted\" before this line.",
		"+\tinserted",
		">\toriginal")
}

func (s *Suite) Test_Autofix_Delete(c *check.C) {
	t := s.Init(c)

	t.SetupCommandLine("--show-autofix", "--source")
	line := t.NewLine("Makefile", 30, "to be deleted")

	fix := line.Autofix()
	fix.Warnf("Dummy.")
	fix.Delete()
	fix.Apply()

	t.CheckOutputLines(
		"WARN: Makefile:30: Dummy.",
		"AUTOFIX: Makefile:30: Deleting this line.",
		"-\tto be deleted")
}

func (s *Suite) Test_Autofix_Delete__combined_with_insert(c *check.C) {
	t := s.Init(c)

	t.SetupCommandLine("--show-autofix", "--source")
	line := t.NewLine("Makefile", 30, "to be deleted")

	fix := line.Autofix()
	fix.Warnf("This line should be replaced completely.")
	fix.Delete()
	fix.InsertAfter("below")
	fix.InsertBefore("above")
	fix.Apply()

	t.CheckOutputLines(
		"WARN: Makefile:30: This line should be replaced completely.",
		"AUTOFIX: Makefile:30: Deleting this line.",
		"AUTOFIX: Makefile:30: Inserting a line \"below\" after this line.",
		"AUTOFIX: Makefile:30: Inserting a line \"above\" before this line.",
		"+\tabove",
		"-\tto be deleted",
		"+\tbelow")
}

// Demonstrates that the --show-autofix option only shows those diagnostics
// that would be fixed.
func (s *Suite) Test_Autofix__suppress_unfixable_warnings(c *check.C) {
	t := s.Init(c)

	t.SetupCommandLine("--show-autofix", "--source")
	lines := t.NewLines("Makefile",
		"line1",
		"line2",
		"line3")

	lines.Lines[0].Warnf("This warning is not shown since it is not part of a fix.")

	fix := lines.Lines[1].Autofix()
	fix.Warnf("Something's wrong here.")
	fix.ReplaceRegex(`.`, "X", -1)
	fix.Apply()

	fix.Warnf("Since XXX marks are usually not fixed, use TODO instead to draw attention.")
	fix.Replace("XXX", "TODO")
	fix.Apply()

	lines.Lines[2].Warnf("Neither is this warning shown.")

	t.CheckOutputLines(
		"WARN: Makefile:2: Something's wrong here.",
		"AUTOFIX: Makefile:2: Replacing \"l\" with \"X\".",
		"AUTOFIX: Makefile:2: Replacing \"i\" with \"X\".",
		"AUTOFIX: Makefile:2: Replacing \"n\" with \"X\".",
		"AUTOFIX: Makefile:2: Replacing \"e\" with \"X\".",
		"AUTOFIX: Makefile:2: Replacing \"2\" with \"X\".",
		"-\tline2",
		"+\tXXXXX",
		"",
		"WARN: Makefile:2: Since XXX marks are usually not fixed, use TODO instead to draw attention.",
		"AUTOFIX: Makefile:2: Replacing \"XXX\" with \"TODO\".",
		"-\tline2",
		"+\tTODOXX")
}

// If an Autofix doesn't do anything it must not log any diagnostics.
func (s *Suite) Test_Autofix__noop_replace(c *check.C) {
	t := s.Init(c)

	line := t.NewLine("Makefile", 14, "Original text")

	fix := line.Autofix()
	fix.Warnf("All-uppercase words should not be used at all.")
	fix.ReplaceRegex(`\b[A-Z]{3,}\b`, "---censored---", -1)
	fix.Apply()

	// No output since there was no all-uppercase word in the text.
	t.CheckOutputEmpty()
}

// When using Autofix.CustomFix, it is tricky to get all the details right.
// For best results, see the existing examples and the documentation.
func (s *Suite) Test_Autofix_Custom(c *check.C) {
	t := s.Init(c)

	lines := t.NewLines("Makefile",
		"line1",
		"line2",
		"line3")

	doFix := func(line Line) {
		fix := line.Autofix()
		fix.Warnf("Please write in ALL-UPPERCASE.")
		fix.Custom(func(printAutofix, autofix bool) {
			fix.Describef(int(line.firstLine), "Converting to uppercase")
			if printAutofix || autofix {
				line.Text = strings.ToUpper(line.Text)
			}
		})
		fix.Apply()
	}

	doFix(lines.Lines[0])

	t.CheckOutputLines(
		"WARN: Makefile:1: Please write in ALL-UPPERCASE.")

	t.SetupCommandLine("--show-autofix")

	doFix(lines.Lines[1])

	t.CheckOutputLines(
		"WARN: Makefile:2: Please write in ALL-UPPERCASE.",
		"AUTOFIX: Makefile:2: Converting to uppercase")
	c.Check(lines.Lines[1].Text, equals, "LINE2")

	t.SetupCommandLine("--autofix")

	doFix(lines.Lines[2])

	t.CheckOutputLines(
		"AUTOFIX: Makefile:3: Converting to uppercase")
	c.Check(lines.Lines[2].Text, equals, "LINE3")
}

func (s *Suite) Test_Autofix_Explain(c *check.C) {
	t := s.Init(c)

	line := t.NewLine("Makefile", 74, "line1")

	fix := line.Autofix()
	fix.Warnf("Please write row instead of line.")
	fix.Replace("line", "row")
	fix.Explain("Explanation")
	fix.Apply()

	t.CheckOutputLines(
		"WARN: Makefile:74: Please write row instead of line.")
	c.Check(G.explanationsAvailable, equals, true)
}

// Since the diagnostic doesn't contain the string "few", nothing happens.
func (s *Suite) Test_Autofix__skip(c *check.C) {
	t := s.Init(c)

	t.SetupCommandLine("--only", "few", "--autofix")

	mklines := t.SetupFileMkLines("fname",
		"VAR=\t111 222 333 444 555 \\",
		"666")
	lines := mklines.lines

	fix := lines.Lines[0].Autofix()
	fix.Warnf("Many.")
	fix.Explain(
		"Explanation.")
	fix.Replace("111", "___")
	fix.ReplaceAfter(" ", "222", "___")
	fix.ReplaceRegex(`\d+`, "___", 1)
	fix.InsertBefore("before")
	fix.InsertAfter("after")
	fix.Delete()
	fix.Custom(func(printAutofix, autofix bool) {})
	fix.Realign(mklines.mklines[0], 32)
	fix.Apply()

	SaveAutofixChanges(lines)

	t.CheckOutputEmpty()
	t.CheckFileLines("fname",
		"VAR=\t111 222 333 444 555 \\",
		"666")
	c.Check(lines.Lines[0].raw[0].textnl, equals, "VAR=\t111 222 333 444 555 \\\n")
	c.Check(lines.Lines[0].raw[1].textnl, equals, "666\n")
}

func (s *Suite) Test_Autofix_Apply__panic(c *check.C) {
	t := s.Init(c)

	line := t.NewLine("fileName", 123, "text")

	t.ExpectFatal(
		func() {
			fix := line.Autofix()
			fix.Apply()
		},
		"FATAL: Pkglint internal error: Each autofix must have a log level and a diagnostic.")

	t.ExpectFatal(
		func() {
			fix := line.Autofix()
			fix.Replace("from", "to")
			fix.Apply()
		},
		"FATAL: Pkglint internal error: Autofix: The diagnostic must be given before the action.")

	t.ExpectFatal(
		func() {
			fix := line.Autofix()
			fix.Warnf("Warning without period")
			fix.Apply()
		},
		"FATAL: Pkglint internal error: Autofix: format \"Warning without period\" must end with a period.")
}

func (s *Suite) Test_Autofix_Apply__file_removed(c *check.C) {
	t := s.Init(c)

	t.SetupCommandLine("--autofix")
	lines := t.SetupFileLines("subdir/file.txt",
		"line 1")
	_ = os.RemoveAll(t.File("subdir"))

	fix := lines.Lines[0].Autofix()
	fix.Warnf("Should start with an uppercase letter.")
	fix.Replace("line", "Line")
	fix.Apply()

	SaveAutofixChanges(lines)

	c.Check(t.Output(), check.Matches, ""+
		"AUTOFIX: ~/subdir/file.txt:1: Replacing \"line\" with \"Line\".\n"+
		"ERROR: ~/subdir/file.txt.pkglint.tmp: Cannot write: .*\n")
}

func (s *Suite) Test_Autofix_Apply__file_busy_Windows(c *check.C) {
	t := s.Init(c)

	if runtime.GOOS != "windows" {
		return
	}

	t.SetupCommandLine("--autofix")
	lines := t.SetupFileLines("subdir/file.txt",
		"line 1")

	// As long as the file is kept open, it cannot be overwritten or deleted.
	openFile, err := os.OpenFile(t.File("subdir/file.txt"), 0, 0666)
	defer openFile.Close()
	c.Check(err, check.IsNil)

	fix := lines.Lines[0].Autofix()
	fix.Warnf("Should start with an uppercase letter.")
	fix.Replace("line", "Line")
	fix.Apply()

	SaveAutofixChanges(lines)

	c.Check(t.Output(), check.Matches, ""+
		"AUTOFIX: ~/subdir/file.txt:1: Replacing \"line\" with \"Line\".\n"+
		"ERROR: ~/subdir/file.txt.pkglint.tmp: Cannot overwrite with auto-fixed content: .*\n")
}

// This test tests the highly unlikely situation in which a file is loaded
// by pkglint, and just before writing the autofixed content back, another
// process takes the file and replaces it with a directory of the same name.
//
// 100% code coverage sometimes requires creativity. :)
func (s *Suite) Test_Autofix_Apply__file_converted_to_directory(c *check.C) {
	t := s.Init(c)

	t.SetupCommandLine("--autofix")
	lines := t.SetupFileLines("file.txt",
		"line 1")

	c.Check(os.RemoveAll(t.File("file.txt")), check.IsNil)
	c.Check(os.MkdirAll(t.File("file.txt"), 0777), check.IsNil)

	fix := lines.Lines[0].Autofix()
	fix.Warnf("Should start with an uppercase letter.")
	fix.Replace("line", "Line")
	fix.Apply()

	SaveAutofixChanges(lines)

	c.Check(t.Output(), check.Matches, ""+
		"AUTOFIX: ~/file.txt:1: Replacing \"line\" with \"Line\".\n"+
		"ERROR: ~/file.txt.pkglint.tmp: Cannot overwrite with auto-fixed content: .*\n")
}
