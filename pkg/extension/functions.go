package extension

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/quvly/Qasmith/internal/limits"
	"github.com/quvly/Qasmith/pkg/ir"
)

// TypeDescriptor identifies an opaque type by name and registration identity.
type TypeDescriptor struct {
	name     Type
	validate func(StaticContext, any) error
}

type opaqueValue struct {
	descriptor *TypeDescriptor
	payload    any
}

// OpaqueType packs and unpacks a typed, immutable host snapshot.
type OpaqueType[T any] struct {
	descriptor *TypeDescriptor
	snapshot   func(T) (T, error)
}

// NewOpaqueType defines a named type. Snapshot must validate and isolate any
// mutable storage on both packing and unpacking.
func NewOpaqueType[T any](name Type, snapshot func(T) (T, error)) (*OpaqueType[T], error) {
	return NewCheckedOpaqueType(name, snapshot, nil)
}

// NewCheckedOpaqueType adds a budget check when a host returns a value to Smith.
func NewCheckedOpaqueType[T any](name Type, snapshot func(T) (T, error), validate func(StaticContext, T) error) (*OpaqueType[T], error) {
	switch name {
	case Int, Float, Bool, Qubit, Void, MeasuredResult, "QReg":
		return nil, fmt.Errorf("reserved type %s", name)
	}
	if !namePattern.MatchString(string(name)) || snapshot == nil {
		return nil, fmt.Errorf("invalid opaque type %q or snapshot callback", name)
	}
	d := &TypeDescriptor{name: name}
	if validate != nil {
		d.validate = func(ctx StaticContext, value any) error {
			payload, ok := value.(T)
			if !ok {
				return fmt.Errorf("invalid %s payload", name)
			}
			return validate(ctx, payload)
		}
	}
	return &OpaqueType[T]{d, snapshot}, nil
}

func (t *OpaqueType[T]) Descriptor() *TypeDescriptor { return t.descriptor }
func (t *OpaqueType[T]) Name() Type                  { return t.descriptor.name }

func (t *OpaqueType[T]) Pack(payload T) (Value, error) {
	snapshot, err := t.snapshot(payload)
	if err != nil {
		return Value{}, err
	}
	return Value{Type: t.Name(), opaque: &opaqueValue{t.descriptor, snapshot}}, nil
}

func (t *OpaqueType[T]) Unpack(value Value) (T, error) {
	var empty T
	if !t.descriptor.Accepts(value) {
		return empty, fmt.Errorf("invalid or foreign %s value", t.Name())
	}
	snapshot, ok := value.opaque.payload.(T)
	if !ok {
		return empty, fmt.Errorf("invalid %s payload", t.Name())
	}
	return t.snapshot(snapshot)
}

func (t *TypeDescriptor) Accepts(value Value) bool {
	return t != nil && value.Type == t.name && value.opaque != nil && value.opaque.descriptor == t
}

// ValidateResult checks registration identity and any declared result budgets.
func (t *TypeDescriptor) ValidateResult(ctx StaticContext, value Value) error {
	if !t.Accepts(value) {
		return fmt.Errorf("invalid or foreign %s value", value.Type)
	}
	if t.validate != nil {
		return t.validate(ctx, value.opaque.payload)
	}
	return nil
}

// StaticLimits is the public type of a trusted callback's budget snapshot.
type StaticLimits = limits.Limits

// StaticContext exposes budgets without providing access to compiler state.
type StaticContext interface {
	Limits() StaticLimits
	Spend(units int) error
}

// CircuitPlan is a reusable immutable circuit recipe. Build must return fresh
// measured IR within the limits supplied at production time.
type CircuitPlan interface {
	Build() (*ir.Program, error)
}

// Parameter accepts a concrete type when Rank is zero. For positive Rank, Type
// constrains the numeric array leaf to Int or Float. An empty Type accepts either.
type Parameter struct {
	Type Type
	Rank int
}

func (p Parameter) String() string {
	if p.Rank == 0 {
		return string(p.Type)
	}
	if p.Type != "" {
		return fmt.Sprintf("rank-%d %s array", p.Rank, p.Type)
	}
	return fmt.Sprintf("rank-%d numeric array", p.Rank)
}

func (p Parameter) Accepts(t Type) bool {
	if p.Rank == 0 {
		return p.Type == t || p.Type == Float && t == Int
	}
	for range p.Rank {
		element, _, ok := t.Array()
		if !ok {
			return false
		}
		t = element
	}
	if p.Type != "" {
		return t == p.Type
	}
	return t == Int || t == Float
}

// Function exports a pure compile-time evaluation or a terminal circuit
// producer. CheckTypes must inspect types alone and may run during check.
type Function struct {
	Name           string
	Documentation  string
	ParameterNames []string
	Parameters     []Parameter
	Return         Type
	CheckTypes     func(StaticLimits, []Type) error
	Validate       func(StaticContext, []Value) error
	Evaluate       func(StaticContext, []Value) (Value, error)
	Produce        func(StaticContext, []Value) (CircuitPlan, error)
}

// RegisterType adds an opaque type identity to this registry.
func (r *Registry) RegisterType(t *TypeDescriptor) error {
	if t == nil || t.name == "" {
		return fmt.Errorf("invalid opaque type descriptor")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.types[t.name] != nil {
		return fmt.Errorf("duplicate structured type %s", t.name)
	}
	if r.types == nil {
		r.types = make(map[Type]*TypeDescriptor)
	}
	r.types[t.name] = t
	return nil
}

func (r *Registry) LookupType(name Type) *TypeDescriptor {
	if r == nil {
		return nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.types[name]
}

// RegisterModule atomically adds explicitly exported functions to a trusted module.
func (r *Registry) RegisterModule(path string, functions ...Function) error {
	if !validModulePath(path) || len(functions) == 0 {
		return fmt.Errorf("invalid or empty trusted module %q", path)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.modules[path] != nil {
		return fmt.Errorf("duplicate trusted module %q", path)
	}
	exports := make(map[string]Function, len(functions))
	known := func(t Type) bool { return t.Classical() || r.types[t] != nil }
	for _, f := range functions {
		switch f.Name {
		case "fn", "pub", "import", "from", "as", "let", "return":
			return fmt.Errorf("export %q is a keyword", f.Name)
		}
		if !namePattern.MatchString(f.Name) || exports[f.Name].Name != "" {
			return fmt.Errorf("invalid or duplicate export %q in %s", f.Name, path)
		}
		if namespace, ok := strings.CutPrefix(path, "host:"); ok && r.entries[namespace+":"+f.Name].Name != "" {
			return fmt.Errorf("duplicate export %s/%s", path, f.Name)
		}
		if f.Produce != nil {
			if f.Evaluate != nil || f.Return != MeasuredResult {
				return fmt.Errorf("terminal producer %s requires MeasuredResult and only a Produce callback", f.Name)
			}
		} else if f.Evaluate == nil || !known(f.Return) {
			return fmt.Errorf("invalid compile-time return type or callback for %s", f.Name)
		}
		for _, p := range f.Parameters {
			if p.Rank == 0 && !known(p.Type) || p.Rank != 0 && (p.Rank < 1 || p.Rank > 256 || p.Type != "" && p.Type != Int && p.Type != Float) {
				return fmt.Errorf("invalid parameter %s for %s", p, f.Name)
			}
		}
		f.Parameters = slices.Clone(f.Parameters)
		f.ParameterNames = slices.Clone(f.ParameterNames)
		exports[f.Name] = f
	}
	if r.modules == nil {
		r.modules = make(map[string]map[string]Function)
	}
	r.modules[path] = exports
	return nil
}

func validModulePath(path string) bool {
	if namespace, ok := strings.CutPrefix(path, "host:"); ok {
		return namespacePattern.MatchString(namespace)
	}
	suffix, ok := strings.CutPrefix(path, "std/")
	if !ok {
		return false
	}
	for _, part := range strings.Split(suffix, "/") {
		if !namespacePattern.MatchString(part) {
			return false
		}
	}
	return true
}

func (r *Registry) HasModule(path string) bool {
	if r == nil {
		return false
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.modules[path] != nil
}

func (r *Registry) LookupFunction(path, name string) (Function, bool) {
	if r == nil {
		return Function{}, false
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	f, ok := r.modules[path][name]
	f.Parameters = slices.Clone(f.Parameters)
	f.ParameterNames = slices.Clone(f.ParameterNames)
	return f, ok
}

// Clone snapshots registrations without copying or exposing registered payloads.
// Module export maps are immutable after registration.
func (r *Registry) Clone() *Registry {
	if r == nil {
		return &Registry{}
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	return &Registry{entries: maps.Clone(r.entries), types: maps.Clone(r.types), modules: maps.Clone(r.modules)}
}

// ModuleExports returns a copy of the module's trusted export descriptions.
func (r *Registry) ModuleExports(path string) map[string]Function {
	if r == nil {
		return nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make(map[string]Function, len(r.modules[path]))
	for name, f := range r.modules[path] {
		f.Parameters = slices.Clone(f.Parameters)
		f.ParameterNames = slices.Clone(f.ParameterNames)
		out[name] = f
	}
	return out
}

// TypeNames returns the registered structured type names.
func (r *Registry) TypeNames() []Type {
	if r == nil {
		return nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]Type, 0, len(r.types))
	for name := range r.types {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}
