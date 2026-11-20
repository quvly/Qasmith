package tsp

import (
	"fmt"

	"github.com/quvly/Qasmith/pkg/optimization/plan"
)

// ReportMetrics adapts a typed route report to the generic plan display API.
type ReportMetrics struct{ Result }

func (r ReportMetrics) Metrics() []plan.Metric {
	metrics := []plan.Metric{
		{Label: "Feasible samples", Value: fmt.Sprint(r.FeasibleSamples)},
		{Label: "Infeasible samples", Value: fmt.Sprint(r.InfeasibleSamples)},
		{Label: "Feasibility rate", Value: fmt.Sprintf("%.6f", r.FeasibilityRate)},
	}
	if r.BestFeasible == nil {
		metrics = append(metrics, plan.Metric{Label: "Best feasible route", Value: "none observed"})
	} else {
		metrics = append(metrics, plan.Metric{Label: "Best feasible route", Value: fmt.Sprint(r.BestFeasible.Decoded.Route)},
			plan.Metric{Label: "Best feasible distance", Value: fmt.Sprint(r.BestFeasible.Decoded.Distance)})
	}
	if r.Classical != nil {
		metrics = append(metrics, plan.Metric{Label: "Classical optimal route", Value: fmt.Sprint(r.Classical.Route)},
			plan.Metric{Label: "Classical optimal distance", Value: fmt.Sprint(r.Classical.Distance)},
			plan.Metric{Label: "Optimal tour samples", Value: fmt.Sprint(r.OptimalTourSamples)},
			plan.Metric{Label: "Optimal tour sample rate", Value: fmt.Sprintf("%.6f", *r.OptimalSampleRate)})
	}
	return metrics
}
