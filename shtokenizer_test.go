package main

import (
	check "gopkg.in/check.v1"
)

// @Beta
func (s *Suite) Test_ShTokenizer_ShAtom(c *check.C) {
	checkRest := func(s string, expected ...*ShAtom) string {
		p := NewShTokenizer(dummyLine, s)
		q := shqPlain
		for _, exp := range expected {
			c.Check(p.ShAtom(q), deepEquals, exp)
			q = exp.Quoting
		}
		return p.Rest()
	}
	check := func(s string, expected ...*ShAtom) {
		rest := checkRest(s, expected...)
		c.Check(rest, equals, "")
	}

	token := func(typ ShAtomType, text string, quoting ShQuoting) *ShAtom {
		return &ShAtom{typ, text, quoting, nil}
	}
	word := func(s string) *ShAtom { return token(shtWord, s, shqPlain) }
	dquot := func(s string) *ShAtom { return token(shtWord, s, shqDquot) }
	squot := func(s string) *ShAtom { return token(shtWord, s, shqSquot) }
	backt := func(s string) *ShAtom { return token(shtWord, s, shqBackt) }
	varuse := func(varname string, modifiers ...string) *ShAtom {
		text := "${" + varname
		for _, modifier := range modifiers {
			text += ":" + modifier
		}
		text += "}"
		varuse := &MkVarUse{varname: varname, modifiers: modifiers}
		return &ShAtom{shtVaruse, text, shqPlain, varuse}
	}
	q := func(q ShQuoting, token *ShAtom) *ShAtom {
		return &ShAtom{token.Type, token.Text, q, token.Data}
	}
	whitespace := func(s string) *ShAtom { return token(shtSpace, s, shqPlain) }
	space := token(shtSpace, " ", shqPlain)
	semicolon := token(shtSemicolon, ";", shqPlain)
	pipe := token(shtPipe, "|", shqPlain)

	check("" /* none */)

	check("$$var",
		word("$$var"))

	check("$$var$$var",
		word("$$var$$var"))

	check("$$var;;",
		word("$$var"),
		token(shtCaseSeparator, ";;", shqPlain))

	check("'single-quoted'",
		q(shqSquot, word("'")),
		q(shqSquot, word("single-quoted")),
		q(shqPlain, word("'")))

	rest := checkRest("\"" /* none */)
	c.Check(rest, equals, "\"")

	check("$${file%.c}.o",
		word("$${file%.c}.o"))

	check("hello",
		word("hello"))

	check("hello, world",
		word("hello,"),
		space,
		word("world"))

	check("\"",
		dquot("\""))

	check("`",
		backt("`"))

	check("`cat fname`",
		backt("`"),
		backt("cat"),
		token(shtSpace, " ", shqBackt),
		backt("fname"),
		word("`"))

	check("hello, \"world\"",
		word("hello,"),
		space,
		dquot("\""),
		dquot("world"),
		word("\""))

	check("set -e;",
		word("set"),
		space,
		word("-e"),
		semicolon)

	check("cd ${WRKSRC}/doc/man/man3; PAGES=\"`ls -1 | ${SED} -e 's,3qt$$,3,'`\";",
		word("cd"),
		space,
		varuse("WRKSRC"),
		word("/doc/man/man3"),
		semicolon,
		space,
		word("PAGES="),
		dquot("\""),
		q(shqDquotBackt, word("`")),
		q(shqDquotBackt, word("ls")),
		q(shqDquotBackt, space),
		q(shqDquotBackt, word("-1")),
		q(shqDquotBackt, space),
		token(shtPipe, "|", shqDquotBackt),
		q(shqDquotBackt, space),
		q(shqDquotBackt, varuse("SED")),
		q(shqDquotBackt, space),
		q(shqDquotBackt, word("-e")),
		q(shqDquotBackt, space),
		q(shqDquotBacktSquot, word("'")),
		q(shqDquotBacktSquot, word("s,3qt$$,3,")),
		q(shqDquotBackt, word("'")),
		q(shqDquot, word("`")),
		q(shqPlain, word("\"")),
		semicolon)

	check("ls -1 | ${SED} -e 's,3qt$$,3,'",
		word("ls"),
		space,
		word("-1"),
		space,
		pipe,
		space,
		varuse("SED"),
		space,
		word("-e"),
		space,
		squot("'"),
		squot("s,3qt$$,3,"),
		word("'"))

	check("(for PAGE in $$PAGES; do ",
		&ShAtom{shtParenOpen, "(", shqPlain, nil},
		word("for"),
		space,
		word("PAGE"),
		space,
		word("in"),
		space,
		word("$$PAGES"),
		semicolon,
		space,
		word("do"),
		space)

	check("    ${ECHO} installing ${DESTDIR}${QTPREFIX}/man/man3/$${PAGE}; ",
		whitespace("    "),
		varuse("ECHO"),
		space,
		word("installing"),
		space,
		varuse("DESTDIR"),
		varuse("QTPREFIX"),
		word("/man/man3/$${PAGE}"),
		semicolon,
		space)

	check("    set - X `head -1 $${PAGE}qt`; ",
		whitespace("    "),
		word("set"),
		space,
		word("-"),
		space,
		word("X"),
		space,
		backt("`"),
		backt("head"),
		q(shqBackt, space),
		backt("-1"),
		q(shqBackt, space),
		backt("$${PAGE}qt"),
		word("`"),
		semicolon,
		space)

	check("`\"one word\"`",
		backt("`"),
		q(shqBacktDquot, word("\"")),
		q(shqBacktDquot, word("one word")),
		q(shqBackt, word("\"")),
		word("`"))

	check("$$var \"$$var\" '$$var' `$$var`",
		word("$$var"),
		space,
		dquot("\""),
		dquot("$$var"),
		word("\""),
		space,
		squot("'"),
		squot("$$var"),
		word("'"),
		space,
		backt("`"),
		backt("$$var"),
		word("`"))

	check("\"`'echo;echo'`\"",
		q(shqDquot, word("\"")),
		q(shqDquotBackt, word("`")),
		q(shqDquotBacktSquot, word("'")),
		q(shqDquotBacktSquot, word("echo;echo")),
		q(shqDquotBackt, word("'")),
		q(shqDquot, word("`")),
		q(shqPlain, word("\"")))

	check("cat<file",
		word("cat"),
		token(shtRedirect, "<", shqPlain),
		word("file"))

	check("-e \"s,\\$$sysconfdir/jabberd,\\$$sysconfdir,g\"",
		word("-e"),
		space,
		dquot("\""),
		dquot("s,\\$$sysconfdir/jabberd,\\$$sysconfdir,g"),
		word("\""))

	check("echo $$,$$/",
		word("echo"),
		space,
		word("$$,$$/"))

	rest = checkRest("COMMENT=\t\\Make $$$$ fast\"",
		word("COMMENT="),
		whitespace("\t"),
		word("\\Make"),
		space,
		word("$$$$"),
		space,
		word("fast"))
	c.Check(rest, equals, "\"")

	check("var=`echo;echo|echo&echo||echo&&echo>echo`",
		q(shqPlain, word("var=")),
		q(shqBackt, word("`")),
		q(shqBackt, word("echo")),
		q(shqBackt, semicolon),
		q(shqBackt, word("echo")),
		q(shqBackt, token(shtPipe, "|", shqBackt)),
		q(shqBackt, word("echo")),
		q(shqBackt, token(shtBackground, "&", shqBackt)),
		q(shqBackt, word("echo")),
		q(shqBackt, token(shtOr, "||", shqBackt)),
		q(shqBackt, word("echo")),
		q(shqBackt, token(shtAnd, "&&", shqBackt)),
		q(shqBackt, word("echo")),
		q(shqBackt, token(shtRedirect, ">", shqBackt)),
		q(shqBackt, word("echo")),
		q(shqPlain, word("`")))

	check("# comment",
		token(shtComment, "# comment", shqPlain))
	check("no#comment",
		word("no#comment"))
	check("`# comment`continue",
		token(shtWord, "`", shqBackt),
		token(shtComment, "# comment", shqBackt),
		token(shtWord, "`", shqPlain),
		token(shtWord, "continue", shqPlain))
	check("`no#comment`continue",
		token(shtWord, "`", shqBackt),
		token(shtWord, "no#comment", shqBackt),
		token(shtWord, "`", shqPlain),
		token(shtWord, "continue", shqPlain))

	check("var=`tr 'A-Z' 'a-z'`",
		token(shtWord, "var=", shqPlain),
		token(shtWord, "`", shqBackt),
		token(shtWord, "tr", shqBackt),
		token(shtSpace, " ", shqBackt),
		token(shtWord, "'", shqBacktSquot),
		token(shtWord, "A-Z", shqBacktSquot),
		token(shtWord, "'", shqBackt),
		token(shtSpace, " ", shqBackt),
		token(shtWord, "'", shqBacktSquot),
		token(shtWord, "a-z", shqBacktSquot),
		token(shtWord, "'", shqBackt),
		token(shtWord, "`", shqPlain))

	check("var=\"`echo \"\\`echo foo\\`\"`\"",
		token(shtWord, "var=", shqPlain),
		token(shtWord, "\"", shqDquot),
		token(shtWord, "`", shqDquotBackt),
		token(shtWord, "echo", shqDquotBackt),
		token(shtSpace, " ", shqDquotBackt),
		token(shtWord, "\"", shqDquotBacktDquot),
		token(shtWord, "\\`echo foo\\`", shqDquotBacktDquot), // One token, since it doesn’t influence parsing.
		token(shtWord, "\"", shqDquotBackt),
		token(shtWord, "`", shqDquot),
		token(shtWord, "\"", shqPlain))
}

func (s *Suite) Test_Shtokenizer_ShAtom_Quoting(c *check.C) {
	checkQuotingChange := func(input, expectedOutput string) {
		p := NewShTokenizer(dummyLine, input)
		q := shqPlain
		result := ""
		for {
			token := p.ShAtom(q)
			if token == nil {
				break
			}
			result += token.Text
			if token.Quoting != q {
				q = token.Quoting
				result += "[" + q.String() + "]"
			}
		}
		c.Check(result, equals, expectedOutput)
		c.Check(p.Rest(), equals, "")
	}

	checkQuotingChange("hello, world", "hello, world")
	checkQuotingChange("hello, \"world\"", "hello, \"[d]world\"[plain]")
	checkQuotingChange("1 \"\" 2 '' 3 `` 4", "1 \"[d]\"[plain] 2 '[s]'[plain] 3 `[b]`[plain] 4")
	checkQuotingChange("\"\"", "\"[d]\"[plain]")
	checkQuotingChange("''", "'[s]'[plain]")
	checkQuotingChange("``", "`[b]`[plain]")
	checkQuotingChange("x\"x`x`x\"x'x\"x'", "x\"[d]x`[db]x`[d]x\"[plain]x'[s]x\"x'[plain]")
	checkQuotingChange("x\"x`x'x'x`x\"", "x\"[d]x`[db]x'[dbs]x'[db]x`[d]x\"[plain]")
	checkQuotingChange("x\\\"x\\'x\\`x\\\\", "x\\\"x\\'x\\`x\\\\")
	checkQuotingChange("x\"x\\\"x\\'x\\`x\\\\", "x\"[d]x\\\"x\\'x\\`x\\\\")
	checkQuotingChange("x'x\\\"x\\'x\\`x\\\\", "x'[s]x\\\"x\\'[plain]x\\`x\\\\")
	checkQuotingChange("x`x\\\"x\\'x\\`x\\\\", "x`[b]x\\\"x\\'x\\`x\\\\")
}

func (s *Suite) Test_ShTokenizer_ShToken(c *check.C) {
	check := func(s string, expected ...*ShToken) {
		p := NewShTokenizer(dummyLine, s)
		for _, exp := range expected {
			c.Check(p.ShToken(), deepEquals, exp)
		}
		c.Check(p.Rest(), equals, "")
	}

	check("",
		nil)

	check("echo",
		NewShToken("echo",
			NewShAtom(shtWord, "echo", shqPlain)))

	check("`cat file`",
		NewShToken("`cat file`",
			NewShAtom(shtWord, "`", shqBackt),
			NewShAtom(shtWord, "cat", shqBackt),
			NewShAtom(shtSpace, " ", shqBackt),
			NewShAtom(shtWord, "file", shqBackt),
			NewShAtom(shtWord, "`", shqPlain)))

	check("PAGES=\"`ls -1 | ${SED} -e 's,3qt$$,3,'`\"",
		NewShToken("PAGES=\"`ls -1 | ${SED} -e 's,3qt$$,3,'`\"",
			NewShAtom(shtWord, "PAGES=", shqPlain),
			NewShAtom(shtWord, "\"", shqDquot),
			NewShAtom(shtWord, "`", shqDquotBackt),
			NewShAtom(shtWord, "ls", shqDquotBackt),
			NewShAtom(shtSpace, " ", shqDquotBackt),
			NewShAtom(shtWord, "-1", shqDquotBackt),
			NewShAtom(shtSpace, " ", shqDquotBackt),
			NewShAtom(shtPipe, "|", shqDquotBackt),
			NewShAtom(shtSpace, " ", shqDquotBackt),
			NewShAtomVaruse("${SED}", shqDquotBackt, "SED"),
			NewShAtom(shtSpace, " ", shqDquotBackt),
			NewShAtom(shtWord, "-e", shqDquotBackt),
			NewShAtom(shtSpace, " ", shqDquotBackt),
			NewShAtom(shtWord, "'", shqDquotBacktSquot),
			NewShAtom(shtWord, "s,3qt$$,3,", shqDquotBacktSquot),
			NewShAtom(shtWord, "'", shqDquotBackt),
			NewShAtom(shtWord, "`", shqDquot),
			NewShAtom(shtWord, "\"", shqPlain)))

	check("echo hello, world",
		NewShToken("echo",
			NewShAtom(shtWord, "echo", shqPlain)),
		NewShToken("hello,",
			NewShAtom(shtWord, "hello,", shqPlain)),
		NewShToken("world",
			NewShAtom(shtWord, "world", shqPlain)))
}
