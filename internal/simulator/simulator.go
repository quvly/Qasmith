package simulator

import (
	"context"
	"fmt"
	"math"
	"math/bits"
	"math/rand"
	"sort"

	"github.com/quvly/Qasmith/pkg/ir"
)

const MaxQubits = 16

type Result struct {
	Qubits, Shots int
	Counts        map[string]int
}

func Statevector(p *ir.Program) ([]complex128, []ir.QubitID, error) {
	return StatevectorContext(context.Background(), p)
}

func StatevectorContext(ctx context.Context, p *ir.Program) ([]complex128, []ir.QubitID, error) {
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}
	layout, err := p.Validate()
	if err != nil {
		return nil, nil, err
	}
	if layout.Qubits > MaxQubits {
		return nil, nil, fmt.Errorf("simulator limit is %d qubits (received %d)", MaxQubits, layout.Qubits)
	}
	state := make([]complex128, 1<<layout.Qubits)
	state[0] = 1
	measured := make([]ir.QubitID, 0, layout.Bits)
	for _, op := range p.Operations {
		if err := ctx.Err(); err != nil {
			return nil, nil, err
		}
		switch op := op.(type) {
		case ir.ApplyGate:
			apply(state, op)
		case ir.Measure:
			measured = append(measured, op.Qubit)
		}
	}
	return state, measured, nil
}

func apply(state []complex128, op ir.ApplyGate) {
	target := 1 << op.Qubits[len(op.Qubits)-1]
	control := 0
	if op.Gate == ir.CX {
		control = 1 << op.Qubits[0]
	}
	var cosine, sine float64
	if len(op.Parameters) != 0 {
		sine, cosine = math.Sincos(op.Parameters[0] / 2)
	}
	c, s := complex(cosine, 0), complex(sine, 0)
	for i := range state {
		if i&target != 0 || i&control != control {
			continue
		}
		j := i | target
		a, b := state[i], state[j]
		switch op.Gate {
		case ir.H:
			state[i], state[j] = (a+b)*math.Sqrt2/2, (a-b)*math.Sqrt2/2
		case ir.RX:
			state[i], state[j] = c*a-1i*s*b, c*b-1i*s*a
		case ir.RY:
			state[i], state[j] = c*a-s*b, s*a+c*b
		case ir.RZ:
			state[i], state[j] = (c-1i*s)*a, (c+1i*s)*b
		case ir.X, ir.CX:
			state[i], state[j] = b, a
		}
	}
}

func Run(p *ir.Program, shots int, seed int64) (Result, error) {
	return RunContext(context.Background(), p, shots, seed)
}

func RunContext(ctx context.Context, p *ir.Program, shots int, seed int64) (Result, error) {
	if shots <= 0 {
		return Result{}, fmt.Errorf("shots must be positive")
	}
	state, measured, err := StatevectorContext(ctx, p)
	if err != nil {
		return Result{}, err
	}
	if len(measured) == 0 {
		return Result{}, fmt.Errorf("circuit has no measurements")
	}
	cumulative := make([]float64, len(state))
	total := 0.0
	for i, a := range state {
		total += real(a)*real(a) + imag(a)*imag(a)
		cumulative[i] = total
	}
	random := rand.New(rand.NewSource(seed))
	counts := make(map[string]int)
	outcome := make([]byte, len(measured))
	for shot := 0; shot < shots; shot++ {
		if shot%256 == 0 {
			if err := ctx.Err(); err != nil {
				return Result{}, err
			}
		}
		sample := random.Float64() * total
		// Rounding can place the sample at the upper edge of the distribution.
		if sample >= total {
			sample = math.Nextafter(total, 0)
		}
		basis := sort.Search(len(cumulative), func(i int) bool { return cumulative[i] > sample })
		for i, id := range measured {
			outcome[i] = '0' + byte((basis>>id)&1)
		}
		counts[string(outcome)]++
	}
	return Result{Qubits: bits.Len(uint(len(state))) - 1, Shots: shots, Counts: counts}, nil
}
