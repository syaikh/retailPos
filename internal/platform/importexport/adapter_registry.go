package importexport

import "fmt"

type AdapterRegistry struct {
	adapters map[string]Adapter
}

func NewAdapterRegistry() *AdapterRegistry {
	return &AdapterRegistry{adapters: make(map[string]Adapter)}
}

func (r *AdapterRegistry) Register(a Adapter) error {
	if a.ModuleName() == "" {
		return fmt.Errorf("adapter module_name is required")
	}
	if _, exists := r.adapters[a.ModuleName()]; exists {
		return fmt.Errorf("adapter already registered for module %q", a.ModuleName())
	}
	r.adapters[a.ModuleName()] = a
	return nil
}

func (r *AdapterRegistry) Get(module string) (Adapter, error) {
	a, exists := r.adapters[module]
	if !exists {
		return nil, fmt.Errorf("no adapter registered for module %q", module)
	}
	return a, nil
}

func (r *AdapterRegistry) Modules() []string {
	modules := make([]string, 0, len(r.adapters))
	for m := range r.adapters {
		modules = append(modules, m)
	}
	return modules
}
