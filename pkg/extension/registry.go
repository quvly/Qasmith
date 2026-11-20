// Package extension provides registration and lowering interfaces for trusted Go intrinsics.
package extension

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
	"sync"

	"github.com/quvly/Qasmith/pkg/ir"
)

type Type string

const (
	Int            Type = "Int"
	Float          Type = "Float"
	Bool           Type = "Bool"
	Qubit          Type = "Qubit"
	MeasuredResult Type = "MeasuredResult"
	Void           Type = "Void"
)

type Value struct {
	Type    Type
	Integer int64
	Qubit   ir.QubitID
	Float   float64
	Bool    bool
	Array   *ArrayValue
	opaque  *opaqueValue
}

// ArrayValue holds the elements of a fixed-size classical array.
type ArrayValue struct{ Elements []Value }

type Signature struct {
	Parameters []Type
	Return     Type
	// Distinct lists pairs of parameter indices that must reference different qubits.
	Distinct [][2]int
}

type LoweringContext interface {
	ApplyGate(ir.Gate, ...ir.QubitID) error
}

// ParameterizedLoweringContext is implemented by the compiler's lowering context.
type ParameterizedLoweringContext interface {
	LoweringContext
	ApplyParameterizedGate(ir.Gate, []float64, ...ir.QubitID) error
}

type Intrinsic struct {
	Namespace, Name string
	Signature       Signature
	// Validate can enforce additional value constraints during lowering.
	Validate func([]Value) error
	Lower    func(LoweringContext, []Value) (Value, error)
}

type Registry struct {
	mu      sync.RWMutex
	entries map[string]Intrinsic
	types   map[Type]*TypeDescriptor
	modules map[string]map[string]Function
}

var namespacePattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_.-]*$`)
var namePattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func (r *Registry) Register(i Intrinsic) error {
	if !namespacePattern.MatchString(i.Namespace) || !namePattern.MatchString(i.Name) || i.Lower == nil {
		return fmt.Errorf("invalid intrinsic namespace, name, or lowering callback")
	}
	switch i.Name {
	case "fn", "pub", "import", "from", "as", "let", "return":
		return fmt.Errorf("intrinsic name %q is a keyword", i.Name)
	}
	if i.Signature.Return != Void && !i.Signature.Return.Classical() {
		return fmt.Errorf("intrinsic must return a classical type or Void")
	}
	for _, typ := range i.Signature.Parameters {
		if !typ.Classical() && typ != Qubit {
			return fmt.Errorf("unsupported intrinsic parameter type %q", typ)
		}
		if i.Signature.Return != Void && typ == Qubit {
			return fmt.Errorf("classical intrinsic cannot borrow qubits")
		}
	}
	for _, pair := range i.Signature.Distinct {
		if pair[0] == pair[1] {
			return fmt.Errorf("distinct operands must be separate parameters")
		}
		for _, index := range pair {
			if index < 0 || index >= len(i.Signature.Parameters) || i.Signature.Parameters[index] != Qubit {
				return fmt.Errorf("invalid distinct qubit parameter index %d", index)
			}
		}
	}
	key := i.Namespace + ":" + i.Name
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.entries[key]; exists || r.modules["host:"+i.Namespace][i.Name].Name != "" {
		return fmt.Errorf("duplicate intrinsic %s", key)
	}
	if r.entries == nil {
		r.entries = make(map[string]Intrinsic)
	}
	r.entries[key] = clone(i)
	return nil
}

// Lookup resolves an explicit host namespace and exported intrinsic name.
func (r *Registry) Lookup(path, name string) (Intrinsic, bool) {
	if r == nil {
		return Intrinsic{}, false
	}
	namespace, ok := strings.CutPrefix(path, "host:")
	if !ok {
		return Intrinsic{}, false
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	i, found := r.entries[namespace+":"+name]
	return clone(i), found
}

func clone(i Intrinsic) Intrinsic {
	i.Signature.Parameters = slices.Clone(i.Signature.Parameters)
	i.Signature.Distinct = slices.Clone(i.Signature.Distinct)
	return i
}
