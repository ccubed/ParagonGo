package scripting

var (
	moduleFunctions = map[string]map[string]any{}
)

func AddModlueFunction(namespace string, name string, funcRef any) {
	if _, ok := moduleFunctions[namespace]; !ok {
		moduleFunctions[namespace] = map[string]any{}
	}
	moduleFunctions[namespace][name] = funcRef
}

func setModuleFunctions(vm registrar) {
	vm.Set("modules", moduleFunctions)
}
