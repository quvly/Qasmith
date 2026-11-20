// Package ir defines the backend-independent Qasmith circuit format.
package ir

import "fmt"

type QubitID int
type BitID int

type Gate string

const (
	X  Gate = "X"
	H  Gate = "H"
	CX Gate = "CX"
	RX Gate = "RX"
	RY Gate = "RY"
	RZ Gate = "RZ"
)

type Location struct {
	File         string
	Line, Column int
}

// Origin records the operation's source followed by its innermost-to-outermost call sites.
type Origin struct {
	Location
	CallSites []Location
}

type Operation interface {
	operation()
}
type AllocateQubit struct {
	Qubit  QubitID
	Origin Origin
}
type ApplyGate struct {
	Gate       Gate
	Qubits     []QubitID
	Origin     Origin
	Parameters []float64
}
type Measure struct {
	Qubit  QubitID
	Bit    BitID
	Origin Origin
}

func (AllocateQubit) operation() {}
func (ApplyGate) operation()     {}
func (Measure) operation()       {}

type Program struct{ Operations []Operation }
type Layout struct{ Qubits, Bits int }

// Validate checks contiguous allocation and measurement IDs, gate arity, and resource liveness.
func (p *Program) Validate() (Layout, error) {
	var layout Layout
	if p == nil {
		return layout, fmt.Errorf("nil circuit")
	}
	var measured []bool
	live := func(id QubitID) bool { return id >= 0 && int(id) < len(measured) && !measured[id] }
	for i, op := range p.Operations {
		var err error
		switch op := op.(type) {
		case AllocateQubit:
			if int(op.Qubit) != len(measured) || layout.Bits > 0 {
				err = fmt.Errorf("invalid allocation q%d", op.Qubit)
				break
			}
			measured = append(measured, false)
			layout.Qubits++
		case ApplyGate:
			if layout.Bits > 0 {
				err = fmt.Errorf("gate %s appears after measurement", op.Gate)
				break
			}
			if err = op.ValidateOperands(); err != nil {
				break
			}
			for _, id := range op.Qubits {
				if !live(id) {
					err = fmt.Errorf("qubit q%d is unavailable", id)
					break
				}
			}
		case Measure:
			if !live(op.Qubit) || int(op.Bit) != layout.Bits {
				err = fmt.Errorf("invalid measurement q%d -> c%d", op.Qubit, op.Bit)
				break
			}
			measured[op.Qubit] = true
			layout.Bits++
		default:
			err = fmt.Errorf("unsupported operation %T", op)
		}
		if err != nil {
			return Layout{}, fmt.Errorf("operation %d: %w", i, err)
		}
	}
	return layout, nil
}

type Backend interface {
	Name() string
	Compile(*Program) ([]byte, error)
}

type Pass interface {
	Name() string
	Run(*Program) error
}
