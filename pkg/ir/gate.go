package ir

import (
	"fmt"
	"math"
)

// GateSpec describes operand counts and whether qubit operands must be distinct.
type GateSpec struct {
	Name               string
	Qubits, Parameters int
	DistinctQubits     bool
}

func (g Gate) Spec() GateSpec {
	switch g {
	case X, H:
		return GateSpec{Name: string(g), Qubits: 1}
	case CX:
		return GateSpec{Name: string(g), Qubits: 2, DistinctQubits: true}
	case RX, RY, RZ:
		return GateSpec{Name: string(g), Qubits: 1, Parameters: 1}
	default:
		return GateSpec{}
	}
}

func (g Gate) Arity() int { return g.Spec().Qubits }

// ValidateOperands checks gate shape and numeric parameters independently of resource liveness.
func (op ApplyGate) ValidateOperands() error {
	spec := op.Gate.Spec()
	if spec.Qubits == 0 || len(op.Qubits) != spec.Qubits || len(op.Parameters) != spec.Parameters {
		return fmt.Errorf("invalid operands for gate %q", op.Gate)
	}
	if spec.DistinctQubits {
		for i, id := range op.Qubits {
			for _, previous := range op.Qubits[:i] {
				if id == previous {
					return fmt.Errorf("invalid operands for gate %s: requires distinct qubits", op.Gate)
				}
			}
		}
	}
	for _, parameter := range op.Parameters {
		if math.IsNaN(parameter) || math.IsInf(parameter, 0) {
			return fmt.Errorf("gate %s requires finite parameters", op.Gate)
		}
	}
	return nil
}
