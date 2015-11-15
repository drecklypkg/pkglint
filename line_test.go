package main

import (
	check "gopkg.in/check.v1"
)

func (s *Suite) TestLineAppendPrepend(c *check.C) {
	line := NewLine("fname", "1", "dummy", []*RawLine{{1, "original\n"}})

	c.Check(line.changed, equals, false)
	c.Check(line.rawLines(), check.DeepEquals, []*RawLine{{1, "original\n"}})

	line.replaceRegex(`(.)(.*)(.)`, "$3$2$1")

	c.Check(line.changed, equals, true)
	c.Check(line.rawLines(), check.DeepEquals, []*RawLine{{1, "lriginao\n"}})

	line.changed = false
	line.replace("i", "u")

	c.Check(line.changed, equals, true)
	c.Check(line.rawLines(), check.DeepEquals, []*RawLine{{1, "lruginao\n"}})
	c.Check(line.raw[0].textnl, equals, "lruginao\n")

	line.replace("lruginao", "middle")
	
	c.Check(line.rawLines(), check.DeepEquals, []*RawLine{{1, "middle\n"}})
	c.Check(line.raw[0].textnl, equals, "middle\n")
	
	line.appendBefore("before")
	line.appendBefore("between before and middle")
	line.prependBefore("beginning")
	line.appendAfter("after")
	line.appendAfter("end")
	line.prependAfter("between middle and after")

	c.Check(line.rawLines(), check.DeepEquals, []*RawLine{
		{0, "beginning\n"},
		{0, "before\n"},
		{0, "between before and middle\n"},
		{1, "middle\n"},
		{0, "between middle and after\n"},
		{0, "after\n"},
		{0, "end\n"}})

	line.delete()

	c.Check(line.rawLines(), check.DeepEquals, []*RawLine{
		{0, "beginning\n"},
		{0, "before\n"},
		{0, "between before and middle\n"},
		{0, "between middle and after\n"},
		{0, "after\n"},
		{0, "end\n"}})
}
