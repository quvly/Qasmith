package limits

import "fmt"

// Limits bounds static evaluation and circuit generation. Zero fields use defaults.
type Limits struct {
	MaxLoopIterations    int
	MaxGates             int
	MaxQubits            int
	MaxArrayElements     int
	MaxDepth             int
	MaxSteps             int
	MaxMatrixElements    int
	MaxModelVariables    int
	MaxModelCoefficients int
	MaxStructuredWork    int
	MaxCircuitLayers     int
}

func Default() Limits {
	return Limits{MaxLoopIterations: 10000, MaxGates: 100000, MaxQubits: 4096,
		MaxArrayElements: 32768, MaxDepth: 256, MaxSteps: 1000000, MaxMatrixElements: 4096, MaxModelVariables: 4096, MaxModelCoefficients: 100000, MaxStructuredWork: 250000, MaxCircuitLayers: 64}
}

func (l Limits) Resolve() (Limits, error) {
	d := Default()
	for _, item := range []struct {
		name            string
		value, fallback *int
	}{
		{"MaxLoopIterations", &l.MaxLoopIterations, &d.MaxLoopIterations},
		{"MaxGates", &l.MaxGates, &d.MaxGates},
		{"MaxQubits", &l.MaxQubits, &d.MaxQubits},
		{"MaxArrayElements", &l.MaxArrayElements, &d.MaxArrayElements},
		{"MaxDepth", &l.MaxDepth, &d.MaxDepth},
		{"MaxSteps", &l.MaxSteps, &d.MaxSteps},
		{"MaxMatrixElements", &l.MaxMatrixElements, &d.MaxMatrixElements},
		{"MaxModelVariables", &l.MaxModelVariables, &d.MaxModelVariables},
		{"MaxModelCoefficients", &l.MaxModelCoefficients, &d.MaxModelCoefficients},
		{"MaxStructuredWork", &l.MaxStructuredWork, &d.MaxStructuredWork},
		{"MaxCircuitLayers", &l.MaxCircuitLayers, &d.MaxCircuitLayers},
	} {
		if *item.value < 0 {
			return Limits{}, fmt.Errorf("%s must be nonnegative", item.name)
		}
		if *item.value == 0 {
			*item.value = *item.fallback
		}
	}
	return l, nil
}
