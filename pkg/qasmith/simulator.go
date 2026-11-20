package qasmith

import (
	"context"

	"github.com/quvly/Qasmith/internal/simulator"
	"github.com/quvly/Qasmith/pkg/ir"
)

// Statevector returns newly allocated amplitudes and terminal measurement qubit
// IDs. The amplitudes are taken before terminal measurements.
func Statevector(ctx context.Context, circuit *ir.Program) ([]complex128, []ir.QubitID, error) {
	return simulator.StatevectorContext(ctx, circuit)
}

// SimulateIR samples terminal measurements from validated circuit IR.
func SimulateIR(ctx context.Context, circuit *ir.Program, shots int, seed int64) (Simulation, error) {
	return simulator.RunContext(ctx, circuit, shots, seed)
}
