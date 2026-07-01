package schema

import "fmt"

type Registry struct {
	schemas map[string]ModuleSchema
}

func NewRegistry() *Registry {
	return &Registry{schemas: make(map[string]ModuleSchema)}
}

func (r *Registry) Register(s ModuleSchema) error {
	if s.ModuleName == "" {
		return fmt.Errorf("schema module_name is required")
	}
	if _, exists := r.schemas[s.ModuleName]; exists {
		return fmt.Errorf("schema already registered for module %q", s.ModuleName)
	}
	r.schemas[s.ModuleName] = s
	return nil
}

func (r *Registry) Get(module string) (ModuleSchema, error) {
	s, exists := r.schemas[module]
	if !exists {
		return ModuleSchema{}, fmt.Errorf("no schema registered for module %q", module)
	}
	return s, nil
}

func (r *Registry) All() []ModuleSchema {
	result := make([]ModuleSchema, 0, len(r.schemas))
	for _, s := range r.schemas {
		result = append(result, s)
	}
	return result
}
