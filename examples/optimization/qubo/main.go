// Command qubo demonstrates the standalone Qasmith optimization libraries.
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"

	"github.com/quvly/Qasmith/pkg/optimization/bruteforce"
	"github.com/quvly/Qasmith/pkg/optimization/ising"
	"github.com/quvly/Qasmith/pkg/optimization/qubo"
)

func main() {
	if err := run(os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(out io.Writer) error {
	builder, err := qubo.New(3)
	if err != nil {
		return err
	}
	for i, coefficient := range []float64{-3, 2, -1} {
		if err := builder.AddLinear(i, coefficient); err != nil {
			return err
		}
	}
	if err := builder.AddQuadratic(0, 1, 4); err != nil {
		return err
	}
	if err := builder.AddOffset(1.25); err != nil {
		return err
	}
	if err := builder.AddExactlyOne([]int{0, 1, 2}, 10); err != nil {
		return err
	}
	model, err := builder.Build()
	if err != nil {
		return err
	}
	bits := []bool{true, false, false}
	value, err := model.Evaluate(bits)
	if err != nil {
		return err
	}
	hamiltonian, err := model.ToIsing()
	if err != nil {
		return err
	}
	spinEnergy, err := hamiltonian.Evaluate(ising.Spins(bits))
	if err != nil {
		return err
	}
	if math.Abs(value-spinEnergy) > 1e-12 {
		return fmt.Errorf("energies disagree: %g and %g", value, spinEnergy)
	}
	qJSON, err := json.MarshalIndent(model, "", "  ")
	if err != nil {
		return err
	}
	hJSON, err := json.MarshalIndent(hamiltonian, "", "  ")
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(out, "QUBO\n%s\n\nIsing\n%s\n\n", qJSON, hJSON); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(out, "Assignment %v has energy %g in both representations\n", binary(bits), value); err != nil {
		return err
	}
	var restored qubo.Model
	if err := json.Unmarshal(qJSON, &restored); err != nil {
		return err
	}
	var restoredIsing ising.Model
	if err := json.Unmarshal(hJSON, &restoredIsing); err != nil {
		return err
	}
	reloadedEnergy, err := restoredIsing.Evaluate(ising.Spins(bits))
	if err != nil {
		return err
	}
	if reloadedEnergy != spinEnergy {
		return fmt.Errorf("Ising JSON changed energy")
	}
	result, err := bruteforce.Solve(restored)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(out, "Best assignment %v with objective energy %g\n", binary(result.Assignment), result.Energy)
	return err
}

func binary(bits []bool) []int {
	out := make([]int, len(bits))
	for i, bit := range bits {
		if bit {
			out[i] = 1
		}
	}
	return out
}
