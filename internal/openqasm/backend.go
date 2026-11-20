package openqasm

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"github.com/quvly/Qasmith/pkg/ir"
)

type Backend struct{}

func (Backend) Name() string { return "OpenQASM 3.0" }

func (Backend) Compile(program *ir.Program) ([]byte, error) {
	layout, err := program.Validate()
	if err != nil {
		return nil, err
	}
	var out bytes.Buffer
	out.WriteString("OPENQASM 3.0;\ninclude \"stdgates.inc\";\n")
	if layout.Qubits > 0 {
		fmt.Fprintf(&out, "\nqubit[%d] q;\n", layout.Qubits)
	}
	if layout.Bits > 0 {
		fmt.Fprintf(&out, "bit[%d] c;\n", layout.Bits)
	}
	gates, measurements := false, false
	for _, op := range program.Operations {
		switch op := op.(type) {
		case ir.ApplyGate:
			if !gates {
				out.WriteByte('\n')
				gates = true
			}
			operands := make([]string, len(op.Qubits))
			for i, id := range op.Qubits {
				operands[i] = fmt.Sprintf("q[%d]", id)
			}
			name := strings.ToLower(op.Gate.Spec().Name)
			if len(op.Parameters) != 0 {
				parameters := make([]string, len(op.Parameters))
				for i, value := range op.Parameters {
					literal := strconv.FormatFloat(value, 'g', -1, 64)
					if !strings.ContainsAny(literal, ".e") {
						literal += ".0"
					}
					parameters[i] = literal
				}
				name += "(" + strings.Join(parameters, ", ") + ")"
			}
			fmt.Fprintf(&out, "%s %s;\n", name, strings.Join(operands, ", "))
		case ir.Measure:
			if !measurements {
				out.WriteByte('\n')
				measurements = true
			}
			fmt.Fprintf(&out, "c[%d] = measure q[%d];\n", op.Bit, op.Qubit)
		}
	}
	return out.Bytes(), nil
}
