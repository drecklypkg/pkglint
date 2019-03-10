package pkglint

// RedundantScope checks for redundant variable definitions and for variables
// that are accidentally overwritten. It tries to be as correct as possible
// by not flagging anything that is defined conditionally.
//
// There may be some edge cases though like defining PKGNAME, then evaluating
// it using :=, then defining it again. This pattern is so error-prone that
// it should not appear in pkgsrc at all, thus pkglint doesn't even expect it.
// (Well, except for the PKGNAME case, but that's deep in the infrastructure
// and only affects the "nb13" extension.)
//
// TODO: This scope is not only used for detecting redundancies. It also
// provides information about whether the variables are constant or depend on
// other variables. Therefore the name may change soon.
type RedundantScope struct {
	vars        map[string]*redundantScopeVarinfo
	includePath includePath
	OnRedundant func(old, new MkLine)
	OnOverwrite func(old, new MkLine)
}
type redundantScopeVarinfo struct {
	vari         *Var
	includePaths []includePath
	lastAction   uint8 // 0 = none, 1 = read, 2 = write
}

func NewRedundantScope() *RedundantScope {
	return &RedundantScope{vars: make(map[string]*redundantScopeVarinfo)}
}

func (s *RedundantScope) Handle(mkline MkLine, ind *Indentation) {
	s.updateIncludePath(mkline)

	switch {
	case mkline.IsVarassign():
		s.handleVarassign(mkline, ind)
	}

	s.handleVarUse(mkline)
}

func (s *RedundantScope) updateIncludePath(mkline MkLine) {
	if mkline.firstLine == 1 {
		s.includePath.push(mkline.Location.Filename)
	} else {
		s.includePath.popUntil(mkline.Location.Filename)
	}
}

func (s *RedundantScope) handleVarassign(mkline MkLine, ind *Indentation) {
	varname := mkline.Varname()
	first := s.vars[varname] == nil
	info := s.get(varname)

	defer func() {
		info.vari.Write(mkline, ind.Depth("") > 0, ind.Varnames()...)
		info.lastAction = 2
		s.access(varname)
	}()

	// In the very first assignment, no redundancy can occur.
	if first {
		return
	}

	// TODO: Just being conditional is only half the truth.
	//  To be precise, the "conditional path" must differ between
	//  this variable assignment and the/any? previous one.
	//  See Test_RedundantScope__overwrite_inside_conditional.
	//  Anyway, too few warnings are better than wrong warnings.
	if info.vari.Conditional() || ind.Depth("") > 0 {
		return
	}

	// When the variable has been read after the previous write,
	// it is not redundant.
	if info.lastAction == 1 {
		return
	}

	op := mkline.Op()
	value := mkline.Value()

	// FIXME: Skip the whole redundancy check if the value is not known to be constant.
	if op == opAssign && info.vari.Value() == value {
		op = /* effectively */ opAssignDefault
	}

	prevWrites := info.vari.WriteLocations()
	if len(prevWrites) > 0 {
		switch op {
		// TODO: What about opAssignEval?
		case opAssign:
			if s.includePath.includesAny(info.includePaths) {
				// This is the usual pattern of including a file and
				// then overwriting some of them. Although technically
				// this overwrites the previous definition, it is not
				// worth a warning since this is used a lot and
				// intentionally.

				// FIXME: ind.IsConditional is not precise enough since it
				//  only looks at the variables. There may be conditions entirely
				//  without variables, such as exists(/usr).
			} else if !ind.IsConditional() {
				s.OnOverwrite(prevWrites[len(prevWrites)-1], mkline)
			}

		case opAssignDefault:
			if s.includePath.includedByAny(info.includePaths) {
				// A variable has been defined before including this file
				// containing the default assignment. This is common and fine.
				// Except when the value is the same as the default value.
				if info.vari.Constant() && info.vari.ConstantValue() == mkline.Value() {
					s.OnRedundant(mkline, prevWrites[len(prevWrites)-1])
				}

			} else if s.includePath.includesOrEqualsAll(info.includePaths) {
				// After including one or more files, the variable is either
				// overwritten or defaulted with the same value as its
				// guaranteed current value. All previous accesses to the
				// variable were either in this file or in an included file.
				s.OnRedundant(prevWrites[len(prevWrites)-1], mkline)
			}
		}
	}
}

func (s *RedundantScope) handleVarUse(mkline MkLine) {
	switch {
	case mkline.IsVarassign(), mkline.IsCommentedVarassign():
		for _, varname := range mkline.DetermineUsedVariables() {
			info := s.get(varname)
			info.vari.Read(mkline)
			info.lastAction = 1
			s.access(varname)
		}

	case mkline.IsDirective():
		// TODO: Handle varuse for conditions and loops.
		break

	case mkline.IsInclude(), mkline.IsSysinclude():
		// TODO: Handle VarUse for includes, which may reference variables.
		break

	case mkline.IsDependency():
		// TODO: Handle VarUse for this case.
	}
}

// access returns the info for the given variable, creating it if necessary.
func (s *RedundantScope) get(varname string) *redundantScopeVarinfo {
	info := s.vars[varname]
	if info == nil {
		v := NewVar(varname)
		info = &redundantScopeVarinfo{v, nil, 0}
		s.vars[varname] = info
	}
	return info
}

// access records the current file location, to be used in later inclusion checks.
func (s *RedundantScope) access(varname string) {
	info := s.vars[varname]
	info.includePaths = append(info.includePaths, s.includePath.copy())
}

// includePath remembers the whole sequence of included files,
// such as Makefile includes ../../a/b/buildlink3.mk includes ../../c/d/buildlink3.mk.
//
// This information is used by the RedundantScope to decide whether
// one of two variable assignments is redundant. Two assignments can
// only be redundant if one location includes the other.
type includePath struct {
	files []string
}

func (p *includePath) push(filename string) {
	p.files = append(p.files, filename)
}

func (p *includePath) popUntil(filename string) {
	for p.files[len(p.files)-1] != filename {
		p.files = p.files[:len(p.files)-1]
	}
}

func (p *includePath) includes(other includePath) bool {
	for i, filename := range p.files {
		if i < len(other.files) && other.files[i] == filename {
			continue
		}
		return false
	}
	return len(p.files) < len(other.files)
}

func (p *includePath) includesAny(others []includePath) bool {
	for _, other := range others {
		if p.includes(other) {
			return true
		}
	}
	return false
}

func (p *includePath) includedByAny(others []includePath) bool {
	for _, other := range others {
		if other.includes(*p) {
			return true
		}
	}
	return false
}

func (p *includePath) includesOrEqualsAll(others []includePath) bool {
	for _, other := range others {
		if !(p.includes(other) || p.equals(other)) {
			return false
		}
	}
	return true
}

func (p *includePath) equals(other includePath) bool {
	if len(p.files) != len(other.files) {
		return false
	}
	for i, filename := range p.files {
		if other.files[i] != filename {
			return false
		}
	}
	return true
}

func (p *includePath) copy() includePath {
	return includePath{append([]string(nil), p.files...)}
}
