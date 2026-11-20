package qasmith

import "github.com/quvly/Qasmith/pkg/ir"

// CancelAdjacentX removes consecutive X gates acting on the same qubit.
type CancelAdjacentX struct{}

func (CancelAdjacentX) Name() string { return "cancel-adjacent-x" }

func (CancelAdjacentX) Run(p *ir.Program) error {
	if _, err := p.Validate(); err != nil {
		return err
	}
	out := p.Operations[:0]
	for _, op := range p.Operations {
		gate, ok := op.(ir.ApplyGate)
		if ok && gate.Gate == ir.X && len(out) > 0 {
			last, ok := out[len(out)-1].(ir.ApplyGate)
			if ok && last.Gate == ir.X && last.Qubits[0] == gate.Qubits[0] {
				out = out[:len(out)-1]
				continue
			}
		}
		out = append(out, op)
	}
	clear(p.Operations[len(out):])
	p.Operations = out
	return nil
}
