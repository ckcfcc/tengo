package tengo

func (s *Script) AddFunction(name string, val func(args ...Object) (Object, error)) {
	if s.builtinIsExist(name) {
		return
	}
	builtinFuncs = append(builtinFuncs, &BuiltinFunction{Name: name, Value: val})
}

func (s *Script) AddFunctions(funcs []*BuiltinFunction) {
	for _, f := range funcs {
		if s.builtinIsExist(f.Name) {
			return
		}
		builtinFuncs = append(builtinFuncs, f)
	}
}

func (s *Script) builtinIsExist(name string) (exist bool) {
	for _, bf := range builtinFuncs {
		if bf.Name == name {
			exist = true
		}
	}

	return
}
