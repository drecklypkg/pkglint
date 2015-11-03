package main

import (
	"regexp"
)

// Records the state of a block of variable assignments that make up a SUBST
// class (see mk/subst.mk).
type SubstContext struct {
	id        *string
	class     *string
	stage     *string
	message   *string
	files     []string
	sed       []string
	vars      []string
	filterCmd *string
}

func (self *SubstContext) isComplete() bool {
	return self.id != nil && self.class != nil && len(self.files) != 0 && (len(self.sed) != 0 || len(self.vars) != 0 || self.filterCmd != nil)
}
func (self *SubstContext) checkVarassign(line *Line, varname, op, value string) {
	if !GlobalVars.opts.optWarnExtra {
		return
	}

	if varname == "SUBST_CLASSES" {
		classes := regexp.MustCompile(`\s+`).Split(value, -1)
		if len(classes) > 1 {
			line.logWarningF("Please add only one class at a time to SUBST_CLASSES.")
		}
		if self.class != nil {
			line.logWarningF("SUBST_CLASSES should only appear once in a SUBST block.")
		}
		self.id = &classes[0]
		self.class = &classes[0]
		return
	}

	var varbase, varparam string
	if m := regexp.MustCompile(`^(SUBST_(?:STAGE|MESSAGE|FILES|SED|VARS|FILTER_CMD))\.([\-\w_]+)$`).FindStringSubmatch(varname); m != nil {
		varbase, varparam = m[1], m[2]
		if self.id == nil {
			line.logWarningF("SUBST_CLASSES should precede the definition of %q.", varname)
			self.id = &varparam
		}
	} else if self.id != nil {
		line.logWarningF("Foreign variable in SUBST block.")
	}

	if varparam != *self.id {
		if self.isComplete() {
			// XXX: This code sometimes produces weird warnings. See
			// meta-pkgs/xorg/Makefile.common 1.41 for an example.
			self.finish(line)

			// The following assignment prevents an additional warning,
			// but from a technically viewpoint, it is incorrect.
			self.class = &varparam
			self.id = &varparam
		} else {
			line.logWarningF("Variable parameter %q does not match SUBST class %q.", varparam, self.id)
		}
		return
	}

	switch varbase {
	case "SUBST_STAGE":
		if self.stage != nil {
			line.logWarningF("Duplicate definition of %q.", varname)
		}
		self.stage = &value
	case "SUBST_MESSAGE":
		if self.message != nil {
			line.logWarningF("Duplicate definition of %q.", varname)
		}
		self.message = &value
	case "SUBST_FILES":
		if len(self.files) > 0 && op != "+=" {
			line.logWarningF("All but the first SUBST_FILES line should use the \"+=\" operator.")
		}
		self.files = append(self.files, value)
	case "SUBST_SED":
		if len(self.sed) > 0 && op != "+=" {
			line.logWarningF("All but the first SUBST_SED line should use the \"+=\" operator.")
		}
		self.sed = append(self.sed, value)
	case "SUBST_FILTER_CMD":
		if self.filterCmd != nil {
			line.logWarningF("Duplicate definition of %q.", varname)
		}
		self.filterCmd = &value
	case "SUBST_VARS":
		if len(self.vars) > 0 && op != "+=" {
			line.logWarningF("All but the first SUBST_VARS line should use the \"+=\" operator.")
		}
		self.vars = append(self.vars, value)
	default:
		line.logWarningF("Foreign variable in SUBST block.")
	}
}
func (self *SubstContext) finish(line *Line) {
	if self.id == nil || !GlobalVars.opts.optWarnExtra {
		return
	}
	if self.class == nil {
		line.logWarningF("Incomplete SUBST block: SUBST_CLASSES missing.")
	}
	if self.stage == nil {
		line.logWarningF("Incomplete SUBST block: SUBST_STAGE missing.")
	}
	if len(self.files) == 0 {
		line.logWarningF("Incomplete SUBST block: SUBST_FILES missing.")
	}
	if len(self.sed) == 0 && len(self.vars) == 0 && self.filterCmd == nil {
		line.logWarningF("Incomplete SUBST block: SUBST_SED, SUBST_VARS or SUBST_FILTER_CMD missing.")
	}
	self.id = nil
	self.class = nil
	self.stage = nil
	self.message = nil
	self.files = self.files[:0]
	self.sed = self.sed[:0]
	self.vars = self.vars[:0]
	self.filterCmd = nil
}
