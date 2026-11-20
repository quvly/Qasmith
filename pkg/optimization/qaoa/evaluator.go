package qaoa

import (
	"context"
	"fmt"
	"slices"

	"github.com/quvly/Qasmith/internal/simulator"
	"github.com/quvly/Qasmith/pkg/ir"
	"github.com/quvly/Qasmith/pkg/optimization/internal/energy"
)

// EvaluationMode describes how expectation values are obtained.
type EvaluationMode string

const (
	Exact   EvaluationMode = "exact"
	Sampled EvaluationMode = "sampled"
)

// LocalMaxQubits is the qubit limit of the existing local simulator.
const LocalMaxQubits = simulator.MaxQubits

// MaxShots bounds local QAOA sampling work per evaluation.
const MaxShots = 1_000_000

// Candidate is the lowest-energy assignment observed in a measurement run.
// Frequency is Count divided by the number of shots.
type Candidate struct {
	Bitstring  string
	Assignment []bool
	Energy     float64
	Count      int
	Frequency  float64
}

// Evaluation reports one objective evaluation. Counts and BestObserved are
// populated only in sampled mode.
type Evaluation struct {
	ExpectedEnergy float64
	QubitCount     int
	LayerCount     int
	Mode           EvaluationMode
	Shots          int
	Counts         map[string]int
	BestObserved   *Candidate
}

// Evaluator is the circuit-evaluation interface used by hybrid optimization.
type Evaluator interface {
	Evaluate(context.Context, Parameters) (Evaluation, error)
}

// EvaluationOptions selects exact evaluation or reproducible local sampling.
// Budgets may lower the default circuit limits. Measurement and passes are
// controlled by the evaluator to maintain the variational circuit semantics.
type EvaluationOptions struct {
	Mode      EvaluationMode
	Shots     int
	Seed      int64
	MaxQubits int
	MaxGates  int
}

// LocalEvaluator reuses the basis energies of an immutable problem.
type LocalEvaluator struct {
	problem  Problem
	options  EvaluationOptions
	energies []float64
}

// NewLocalEvaluator checks the local qubit and shot limits before allocating
// its lookup table. Sampled evaluators use the same seed for each evaluation.
func NewLocalEvaluator(ctx context.Context, p Problem, opts EvaluationOptions) (*LocalEvaluator, error) {
	if opts.Mode == "" {
		opts.Mode = Exact
	}
	switch opts.Mode {
	case Exact:
	case Sampled:
		if err := checkShots(opts.Shots); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unknown evaluation mode %q", opts.Mode)
	}
	n := p.model.Variables()
	if n > simulator.MaxQubits {
		return nil, fmt.Errorf("simulator limit is %d qubits (received %d)", simulator.MaxQubits, n)
	}
	if _, _, err := p.gateBudget(1, BuildOptions{MaxQubits: opts.MaxQubits, MaxGates: opts.MaxGates}); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	basis := make([]float64, 1<<n)
	spins := make([]int8, n)
	for index := range basis {
		if index%256 == 0 {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
		}
		for i := range spins {
			spins[i] = 1 - 2*int8((index>>i)&1)
		}
		value, err := p.model.Evaluate(spins)
		if err != nil {
			return nil, fmt.Errorf("basis energy %d: %w", index, err)
		}
		basis[index] = value
	}
	return &LocalEvaluator{problem: p, options: opts, energies: basis}, nil
}

func checkShots(shots int) error {
	if shots < 1 || shots > MaxShots {
		return fmt.Errorf("shots must be between 1 and %d", MaxShots)
	}
	return nil
}

// EvaluateExact evaluates a single parameter set with no sampling.
func EvaluateExact(ctx context.Context, p Problem, params Parameters) (Evaluation, error) {
	if err := params.Validate(len(params.Gammas)); err != nil {
		return Evaluation{}, err
	}
	e, err := NewLocalEvaluator(ctx, p, EvaluationOptions{Mode: Exact})
	if err != nil {
		return Evaluation{}, err
	}
	return e.Evaluate(ctx, params)
}

// EvaluateSampled evaluates a single parameter set using terminal measurements.
func EvaluateSampled(ctx context.Context, p Problem, params Parameters, shots int, seed int64) (Evaluation, error) {
	if err := params.Validate(len(params.Gammas)); err != nil {
		return Evaluation{}, err
	}
	e, err := NewLocalEvaluator(ctx, p, EvaluationOptions{Mode: Sampled, Shots: shots, Seed: seed})
	if err != nil {
		return Evaluation{}, err
	}
	return e.Evaluate(ctx, params)
}

// Evaluate computes the probability-weighted energy for the given parameters.
func (e *LocalEvaluator) Evaluate(ctx context.Context, params Parameters) (Evaluation, error) {
	return e.evaluate(ctx, params, e.options.Mode, e.options.Shots, e.options.Seed)
}

// Exact evaluates unmeasured amplitudes using the same cached basis energies.
func (e *LocalEvaluator) Exact(ctx context.Context, params Parameters) (Evaluation, error) {
	return e.evaluate(ctx, params, Exact, 0, 0)
}

func (e *LocalEvaluator) evaluate(ctx context.Context, params Parameters, mode EvaluationMode, shots int, seed int64) (Evaluation, error) {
	if err := ctx.Err(); err != nil {
		return Evaluation{}, err
	}
	measured := mode == Sampled
	if measured {
		if err := checkShots(shots); err != nil {
			return Evaluation{}, err
		}
	}
	circuit, err := e.problem.Build(params, BuildOptions{Measure: measured, MaxQubits: e.options.MaxQubits, MaxGates: e.options.MaxGates})
	if err != nil {
		return Evaluation{}, err
	}
	result := Evaluation{Mode: mode, QubitCount: e.problem.model.Variables(), LayerCount: len(params.Gammas)}
	if measured {
		return e.SampleCircuit(ctx, circuit, len(params.Gammas), shots, seed)
	}
	state, _, err := simulator.StatevectorContext(ctx, circuit)
	if err != nil {
		return Evaluation{}, err
	}
	var total energy.Sum
	for _, amp := range state {
		if err := total.Add(probability(amp)); err != nil {
			return Evaluation{}, err
		}
	}
	norm, err := total.Value()
	if err != nil || norm <= 0 {
		return Evaluation{}, fmt.Errorf("invalid statevector normalization")
	}
	var expected energy.Sum
	for i, amp := range state {
		if i%256 == 0 {
			if err := ctx.Err(); err != nil {
				return Evaluation{}, err
			}
		}
		if err := expected.Add((probability(amp) / norm) * e.energies[i]); err != nil {
			return Evaluation{}, err
		}
	}
	result.ExpectedEnergy, err = expected.Value()
	return result, err
}

func probability(amp complex128) float64 { return real(amp)*real(amp) + imag(amp)*imag(amp) }

// DecodeBitstring returns a binary assignment in display order, starting with x[0].
func DecodeBitstring(bits string) ([]bool, error) {
	if bits == "" {
		return nil, fmt.Errorf("empty bitstring")
	}
	out := make([]bool, len(bits))
	for i := range bits {
		switch bits[i] {
		case '0':
		case '1':
			out[i] = true
		default:
			return nil, fmt.Errorf("invalid binary digit at index %d", i)
		}
	}
	return out, nil
}

func sortedPatterns(counts map[string]int) []string {
	keys := make([]string, 0, len(counts))
	for key := range counts {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys
}

func (e *LocalEvaluator) summarize(ctx context.Context, result *Evaluation) error {
	var sum energy.Sum
	for _, pattern := range sortedPatterns(result.Counts) {
		if err := ctx.Err(); err != nil {
			return err
		}
		// The simulator emits q[0] first while the basis index uses q[0] as its low bit.
		index := 0
		for i := range pattern {
			index |= int(pattern[i]-'0') << i
		}
		value := e.energies[index]
		count := result.Counts[pattern]
		frequency := float64(count) / float64(result.Shots)
		if err := sum.Add(frequency * value); err != nil {
			return err
		}
		if result.BestObserved == nil || value < result.BestObserved.Energy {
			assignment, err := DecodeBitstring(pattern)
			if err != nil {
				return err
			}
			result.BestObserved = &Candidate{Bitstring: pattern, Assignment: assignment, Energy: value, Count: count, Frequency: frequency}
		}
	}
	var err error
	result.ExpectedEnergy, err = sum.Value()
	return err
}

// SampleCircuit applies the existing evaluator to a prepared measured circuit
// whose register and measurement order match this problem.
func (e *LocalEvaluator) SampleCircuit(ctx context.Context, circuit *ir.Program, layers, shots int, seed int64) (Evaluation, error) {
	if err := checkShots(shots); err != nil {
		return Evaluation{}, err
	}
	_, budget, err := e.problem.gateBudget(layers, BuildOptions{MaxQubits: e.options.MaxQubits, MaxGates: e.options.MaxGates})
	if err != nil {
		return Evaluation{}, err
	}
	n := e.problem.model.Variables()
	if err := validateLayout(circuit, n, n, budget); err != nil {
		return Evaluation{}, err
	}
	sample, err := simulator.RunContext(ctx, circuit, shots, seed)
	if err != nil {
		return Evaluation{}, err
	}
	result := Evaluation{Mode: Sampled, QubitCount: n, LayerCount: layers, Shots: shots, Counts: sample.Counts}
	if err := e.summarize(ctx, &result); err != nil {
		return Evaluation{}, err
	}
	return result, nil
}
