package intqa

import (
	"bytes"
	"fmt"
	"gopkg.in/check.v1"
	"io/ioutil"
	"testing"
)

type Suite struct {
	c       *check.C
	ck      *TestNameChecker
	summary string
}

func Test(t *testing.T) {
	check.Suite(&Suite{})
	check.TestingT(t)
}

func (s *Suite) Init(c *check.C) *TestNameChecker {
	errorf := func(format string, args ...interface{}) {
		s.summary = fmt.Sprintf(format, args...)
	}

	s.c = c
	s.ck = NewTestNameChecker(errorf)
	s.ck.Enable(EAll)
	s.ck.out = ioutil.Discard
	return s.ck
}

func (s *Suite) TearDownTest(c *check.C) {
	s.c = c
	s.CheckErrors(nil...)
	s.CheckSummary("")
}

func (s *Suite) CheckErrors(errors ...string) {
	s.c.Check(s.ck.errors, check.DeepEquals, errors)
	s.ck.errors = nil
}

func (s *Suite) CheckSummary(summary string) {
	s.c.Check(s.summary, check.Equals, summary)
	s.summary = ""
}

func (s *Suite) Test_TestNameChecker_Enable(c *check.C) {
	ck := s.Init(c)

	ck.Enable(ENone) // overwrite initialization from Suite.Init

	c.Check(ck.errorsMask, check.Equals, uint64(0))

	ck.Enable(EAll)

	c.Check(ck.errorsMask, check.Equals, ^uint64(0))

	ck.Enable(ENone, EMissingTest)

	c.Check(ck.errorsMask, check.Equals, uint64(4))

	ck.Enable(EAll, -EMissingTest)

	c.Check(ck.errorsMask, check.Equals, ^uint64(0)^4)
}

func (s *Suite) Test_TestNameChecker_Check(c *check.C) {
	ck := s.Init(c)

	ck.Check()

	s.CheckErrors(
		"Missing unit test \"Test_NewTestNameChecker\" for \"NewTestNameChecker\".",
		"Missing unit test \"Test_TestNameChecker_IgnoreFiles\" for \"TestNameChecker.IgnoreFiles\".",
		"Missing unit test \"Test_TestNameChecker_load\" for \"TestNameChecker.load\".",
		"Missing unit test \"Test_TestNameChecker_loadDecl\" for \"TestNameChecker.loadDecl\".",
		"Missing unit test \"Test_TestNameChecker_addCode\" for \"TestNameChecker.addCode\".",
		"Missing unit test \"Test_TestNameChecker_addTestee\" for \"TestNameChecker.addTestee\".",
		"Missing unit test \"Test_TestNameChecker_relate\" for \"TestNameChecker.relate\".",
		"Missing unit test \"Test_TestNameChecker_checkTests\" for \"TestNameChecker.checkTests\".",
		"Missing unit test \"Test_TestNameChecker_checkTestees\" for \"TestNameChecker.checkTestees\".",
		"Missing unit test \"Test_TestNameChecker_isIgnored\" for \"TestNameChecker.isIgnored\".",
		"Missing unit test \"Test_TestNameChecker_addError\" for \"TestNameChecker.addError\".",
		"Missing unit test \"Test_Suite_Init\" for \"Suite.Init\".",
		"Missing unit test \"Test_Suite_TearDownTest\" for \"Suite.TearDownTest\".",
		"Missing unit test \"Test_Suite_CheckErrors\" for \"Suite.CheckErrors\".",
		"Missing unit test \"Test_Suite_CheckSummary\" for \"Suite.CheckSummary\".",
		"Missing unit test \"Test_Value_Method\" for \"Value.Method\".")
	s.CheckSummary("16 errors.")
}

func (s *Suite) Test_TestNameChecker_addTest(c *check.C) {
	ck := s.Init(c)

	ck.addTest(code{"filename.go", "Type", "Method", 0})

	s.CheckErrors(
		"Test \"Type.Method\" must start with \"Test_\".")
}

func (s *Suite) Test_TestNameChecker_addTest__empty_description(c *check.C) {
	ck := s.Init(c)

	ck.addTest(code{"filename.go", "Suite", "Test_Method__", 0})

	s.CheckErrors(
		"Test \"Suite.Test_Method__\" must not have a nonempty description.")
}

func (s *Suite) Test_TestNameChecker_nextOrder(c *check.C) {
	ck := s.Init(c)

	c.Check(ck.nextOrder(), check.Equals, 0)
	c.Check(ck.nextOrder(), check.Equals, 1)
	c.Check(ck.nextOrder(), check.Equals, 2)
}

func (s *Suite) Test_TestNameChecker_checkTestFile__global(c *check.C) {
	ck := s.Init(c)

	ck.checkTestFile(&test{
		code{"demo_test.go", "Suite", "Test__Global", 0},
		"",
		"",
		&testee{code{"other.go", "", "Global", 0}}})

	s.CheckErrors(
		"Test \"Suite.Test__Global\" for \"Global\" " +
			"must be in other_test.go instead of demo_test.go.")
}

func (s *Suite) Test_TestNameChecker_checkTestTestee__global(c *check.C) {
	ck := s.Init(c)

	ck.checkTestTestee(&test{
		code{"demo_test.go", "Suite", "Test__Global", 0},
		"",
		"",
		nil})

	s.CheckErrors(
		nil...)
}

func (s *Suite) Test_TestNameChecker_checkTestTestee__no_testee(c *check.C) {
	ck := s.Init(c)

	ck.checkTestTestee(&test{
		code{"demo_test.go", "Suite", "Test_Missing", 0},
		"Missing",
		"",
		nil})

	s.CheckErrors(
		"Missing testee \"Missing\" for test \"Suite.Test_Missing\".")
}

func (s *Suite) Test_TestNameChecker_checkTestTestee__testee_exists(c *check.C) {
	ck := s.Init(c)

	ck.checkTestTestee(&test{
		code{"demo_test.go", "Suite", "Test_Missing", 0},
		"Missing",
		"",
		&testee{}})

	s.CheckErrors(
		nil...)
}

func (s *Suite) Test_TestNameChecker_checkTestDescr__camel_case(c *check.C) {
	ck := s.Init(c)

	ck.checkTestDescr(&test{
		code{"demo_test.go", "Suite", "Test_Missing__CamelCase", 0},
		"Missing",
		"CamelCase",
		&testee{}})

	s.CheckErrors(
		"Suite.Test_Missing__CamelCase: Test description \"CamelCase\" " +
			"must not use CamelCase in the first word.")
}

func (s *Suite) Test_TestNameChecker_checkTesteeTest(c *check.C) {
	ck := s.Init(c)

	ck.checkTesteeTest(
		&testee{code{"demo.go", "Type", "", 0}},
		nil)
	ck.checkTesteeTest(
		&testee{code{"demo.go", "", "Func", 0}},
		nil)
	ck.checkTesteeTest(
		&testee{code{"demo.go", "Type", "Method", 0}},
		nil)

	s.CheckErrors(
		"Missing unit test \"Test_Func\" for \"Func\".",
		"Missing unit test \"Test_Type_Method\" for \"Type.Method\".")
}

func (s *Suite) Test_TestNameChecker_checkOrder(c *check.C) {
	ck := s.Init(c)

	ck.addTestee(code{"f.go", "T", "", 10})
	ck.addTestee(code{"f.go", "T", "M1", 11})
	ck.addTestee(code{"f.go", "T", "M2", 12})
	ck.addTestee(code{"f.go", "T", "M3", 13})
	ck.addTest(code{"f_test.go", "S", "Test_T_M1", 100})    // maxTestee = 11
	ck.addTest(code{"f_test.go", "S", "Test_T_M2", 101})    // maxTestee = 12
	ck.addTest(code{"f_test.go", "S", "Test_T", 102})       // testee 10 < maxTestee 12: insert before first [.testee > testee 10] == T_M1
	ck.addTest(code{"f_test.go", "S", "Test_T_M3", 103})    // maxTestee = 13
	ck.addTest(code{"f_test.go", "S", "Test_T__1", 104})    // testee < maxTestee: insert before first [testee > 10]
	ck.addTest(code{"f_test.go", "S", "Test_T__2", 105})    // testee < maxTestee: insert before first [testee > 10]
	ck.addTest(code{"f_test.go", "S", "Test_T_M2__1", 106}) // testee < maxTestee: insert before first [testee > 12] == T_M3
	ck.relate()

	ck.checkOrder()

	s.CheckErrors(
		"Test \"S.Test_T\" must be ordered before \"S.Test_T_M1\".",
		"Test \"S.Test_T__1\" must be ordered before \"S.Test_T_M1\".",
		"Test \"S.Test_T__2\" must be ordered before \"S.Test_T_M1\".",
		"Test \"S.Test_T_M2__1\" must be ordered before \"S.Test_T_M3\".")
}

func (s *Suite) Test_TestNameChecker_print__empty(c *check.C) {
	var out bytes.Buffer
	ck := s.Init(c)
	ck.out = &out

	ck.print()

	c.Check(out.String(), check.Equals, "")
}

func (s *Suite) Test_TestNameChecker_print__errors(c *check.C) {
	var out bytes.Buffer
	ck := s.Init(c)
	ck.out = &out

	ck.addError(EName, "1")
	ck.print()

	c.Check(out.String(), check.Equals, "1\n")
	s.CheckErrors("1")
	s.CheckSummary("1 error.")
}

func (s *Suite) Test_code_fullName(c *check.C) {
	_ = s.Init(c)

	test := func(typeName, funcName, fullName string) {
		code := code{"filename", typeName, funcName, 0}
		c.Check(code.fullName(), check.Equals, fullName)
	}

	test("Type", "", "Type")
	test("", "Func", "Func")
	test("Type", "Method", "Type.Method")
}

func (s *Suite) Test_code_isType(c *check.C) {
	_ = s.Init(c)

	test := func(typeName, funcName string, isType bool) {
		code := code{"filename", typeName, funcName, 0}
		c.Check(code.isType(), check.Equals, isType)
	}

	test("Type", "", true)
	test("", "Func", false)
	test("Type", "Method", false)
}

func (s *Suite) Test_code_isMethod(c *check.C) {
	_ = s.Init(c)

	test := func(typeName, funcName string, isMethod bool) {
		code := code{"filename", typeName, funcName, 0}
		c.Check(code.isMethod(), check.Equals, isMethod)
	}

	test("Type", "", false)
	test("", "Func", false)
	test("Type", "Method", true)
}

func (s *Suite) Test_code_isTest(c *check.C) {
	_ = s.Init(c)

	test := func(filename, typeName, funcName string, isTest bool) {
		code := code{filename, typeName, funcName, 0}
		c.Check(code.isTest(), check.Equals, isTest)
	}

	test("f.go", "Type", "", false)
	test("f.go", "", "Func", false)
	test("f.go", "Type", "Method", false)
	test("f.go", "Type", "Test", false)
	test("f.go", "Type", "Test_Type_Method", false)
	test("f.go", "", "Test_Type_Method", false)
	test("f_test.go", "Type", "Test", true)
	test("f_test.go", "Type", "Test_Type_Method", true)
	test("f_test.go", "", "Test_Type_Method", true)
}

func (s *Suite) Test_plural(c *check.C) {
	_ = s.Init(c)

	c.Check(plural(0, "singular", "plural"), check.Equals, "")
	c.Check(plural(1, "singular", "plural"), check.Equals, "1 singular")
	c.Check(plural(2, "singular", "plural"), check.Equals, "2 plural")
	c.Check(plural(1000, "singular", "plural"), check.Equals, "1000 plural")
}

func (s *Suite) Test_isCamelCase(c *check.C) {
	_ = s.Init(c)

	c.Check(isCamelCase(""), check.Equals, false)
	c.Check(isCamelCase("Word"), check.Equals, false)
	c.Check(isCamelCase("Ada_Case"), check.Equals, false)
	c.Check(isCamelCase("snake_case"), check.Equals, false)
	c.Check(isCamelCase("CamelCase"), check.Equals, true)

	// After the first underscore of the description, any CamelCase
	// is ignored because there is no danger of confusing the method
	// name with the description.
	c.Check(isCamelCase("Word_CamelCase"), check.Equals, false)
}

func (s *Suite) Test_join(c *check.C) {
	_ = s.Init(c)

	c.Check(join("", " and ", ""), check.Equals, "")
	c.Check(join("one", " and ", ""), check.Equals, "one")
	c.Check(join("", " and ", "two"), check.Equals, "two")
	c.Check(join("one", " and ", "two"), check.Equals, "one and two")
}

type Value struct{}

// Method has no star on the receiver,
// for code coverage of TestNameChecker.loadDecl.
func (Value) Method() {}
