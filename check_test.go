package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	check "gopkg.in/check.v1"
)

var equals = check.Equals
var deepEquals = check.DeepEquals

type Suite struct {
	stdout bytes.Buffer
	stderr bytes.Buffer
	tmpdir string
}

func (s *Suite) Stdout() string {
	defer s.stdout.Reset()
	return s.stdout.String()
}

func (s *Suite) Stderr() string {
	defer s.stderr.Reset()
	return s.stderr.String()
}

// Returns and consumes the output from both stdout and stderr.
func (s *Suite) Output() string {
	return s.Stdout() + s.Stderr()
}

func (s *Suite) OutputCleanTmpdir() string {
	return strings.Replace(s.Output(), s.tmpdir+"/", "", -1)
}

func (s *Suite) NewLines(fname string, lines ...string) []*Line {
	result := make([]*Line, len(lines))
	for i, line := range lines {
		result[i] = NewLine(fname, i+1, line, []*RawLine{{i + 1, line + "\n"}})
	}
	return result
}

func (s *Suite) NewMkLines(fname string, lines ...string) *MkLines {
	return NewMkLines(s.NewLines(fname, lines...))
}

func (s *Suite) UseCommandLine(c *check.C, args ...string) {
	exitcode := new(Pkglint).ParseCommandLine(append([]string{"pkglint"}, args...))
	if exitcode != nil && *exitcode != 0 {
		c.Fatalf("Cannot parse command line: %#v", args)
	}
}

func (s *Suite) RegisterTool(toolname, varname string, varRequired bool) {
	if G.globalData.tools == nil {
		G.globalData.tools = make(map[string]bool)
		G.globalData.vartools = make(map[string]string)
		G.globalData.toolsVarRequired = make(map[string]bool)
	}
	G.globalData.tools[toolname] = true
	G.globalData.vartools[toolname] = varname
	if varRequired {
		G.globalData.toolsVarRequired[toolname] = true
	}
}

func (s *Suite) CreateTmpFile(c *check.C, relFname, content string) (absFname string) {
	if s.tmpdir == "" {
		s.tmpdir = filepath.ToSlash(c.MkDir())
	}
	absFname = s.tmpdir + "/" + relFname
	err := os.MkdirAll(path.Dir(absFname), 0777)
	c.Assert(err, check.IsNil)

	err = ioutil.WriteFile(absFname, []byte(content), 0666)
	c.Check(err, check.IsNil)
	return
}

func (s *Suite) ExpectFatalError(action func()) {
	if r := recover(); r != nil {
		if _, ok := r.(pkglintFatal); ok {
			action()
			return
		}
		panic(r)
	}
}

func (s *Suite) SetUpTest(c *check.C) {
	G = new(GlobalVars)
	G.logOut, G.logErr, G.traceOut = &s.stdout, &s.stderr, &s.stdout
}

func (s *Suite) TearDownTest(c *check.C) {
	G = nil
	if out := s.Output(); out != "" {
		fmt.Fprintf(os.Stderr, "Unchecked output in %q; check with: c.Check(s.Output(), equals, %q)", c.TestName(), out)
	}
	s.tmpdir = ""
}

var _ = check.Suite(new(Suite))

func Test(t *testing.T) { check.TestingT(t) }
