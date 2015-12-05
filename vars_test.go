package main

import (
	check "gopkg.in/check.v1"
)

func (s *Suite) TestVariableNeedsQuoting(c *check.C) {
	line := NewLine("fname", "1", "dummy", nil)
	G.globalData.InitVartypes()
	pkgnameType := G.globalData.vartypes["PKGNAME"]

	// In Makefile: PKGNAME := ${UNKNOWN}
	vuc := &VarUseContext{vucTimeParse, pkgnameType, vucQuotUnknown, vucExtentUnknown}
	nq := variableNeedsQuoting(line, "UNKNOWN", vuc)

	c.Check(nq, equals, nqDontKnow)
}
