package schema

import "testing"

func TestRegistryRegisterAndGet(t *testing.T) {
	r := NewRegistry()

	err := r.Register(ModuleSchema{ModuleName: "test", DisplayName: "Test"})
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	s, err := r.Get("test")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if s.ModuleName != "test" {
		t.Errorf("got module %q, want %q", s.ModuleName, "test")
	}
}

func TestRegistryDuplicate(t *testing.T) {
	r := NewRegistry()
	_ = r.Register(ModuleSchema{ModuleName: "test"})
	err := r.Register(ModuleSchema{ModuleName: "test"})
	if err == nil {
		t.Fatal("expected error for duplicate registration")
	}
}

func TestRegistryGetMissing(t *testing.T) {
	r := NewRegistry()
	_, err := r.Get("nonexistent")
	if err == nil {
		t.Fatal("expected error for missing module")
	}
}

func TestRegistryEmptyModuleName(t *testing.T) {
	r := NewRegistry()
	err := r.Register(ModuleSchema{ModuleName: ""})
	if err == nil {
		t.Fatal("expected error for empty module name")
	}
}

func TestRegistryAll(t *testing.T) {
	r := NewRegistry()
	_ = r.Register(ModuleSchema{ModuleName: "a"})
	_ = r.Register(ModuleSchema{ModuleName: "b"})

	schemas := r.All()
	if len(schemas) != 2 {
		t.Fatalf("got %d schemas, want 2", len(schemas))
	}
}
