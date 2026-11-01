package main

import (
	"fmt"
	"math"
)

// comparisonValue returns metrics in the same units shown on screen.
func comparisonValue(a activity, label string) (float64, bool) {
	switch label {
	case "DISTANCE":
		return a.Distance / 1609.344, a.Distance > 0
	case "MOVING":
		return float64(a.MovingTime), a.MovingTime > 0
	case "ELEVATION":
		return a.Elevation * 3.28084, a.Elevation >= 0
	case "AVG PACE":
		if a.AverageSpeed != nil && *a.AverageSpeed > 0 {
			return 1609.344 / *a.AverageSpeed, true
		}
		if a.Distance > 0 && a.MovingTime > 0 {
			return float64(a.MovingTime) * 1609.344 / a.Distance, true
		}
	case "AVG HR":
		if a.AverageHR != nil && *a.AverageHR > 0 {
			return *a.AverageHR, true
		}
	case "MAX HR":
		if a.MaximumHR != nil && *a.MaximumHR > 0 {
			return *a.MaximumHR, true
		}
	case "AVG CADENCE":
		if a.AverageCadence != nil && *a.AverageCadence > 0 {
			return *a.AverageCadence * 2, true
		}
	case "AVG POWER":
		if a.AverageWatts != nil && *a.AverageWatts > 0 {
			return *a.AverageWatts, true
		}
	case "EFFORT":
		if a.Effort != nil && *a.Effort > 0 {
			return *a.Effort, true
		}
	case "RELATIVE EFFORT":
		if a.SufferScore != nil && *a.SufferScore > 0 {
			return *a.SufferScore, true
		}
	}
	return 0, false
}

func metricDifference(current activity, history []activity, label string) (string, int) {
	value, ok := comparisonValue(current, label)
	if !ok {
		return "", 0
	}
	var total float64
	count := 0
	for _, run := range history {
		if run.ID == current.ID || (run.Sport != "Run" && run.Sport != "TrailRun" && run.Sport != "VirtualRun") {
			continue
		}
		if sample, ok := comparisonValue(run, label); ok {
			total += sample
			count++
		}
	}
	if count < 5 {
		return "", 0
	}
	delta := value - total/float64(count)
	precision := 0
	if label == "DISTANCE" {
		precision = 2
	}
	if label == "EFFORT" {
		precision = 1
	}
	if math.Round(math.Abs(delta)*math.Pow10(precision)) == 0 {
		return "", 0
	}
	direction := 1
	sign := "+"
	if delta < 0 {
		direction, sign = -1, "-"
	}
	delta = math.Abs(delta)
	var change string
	switch label {
	case "DISTANCE":
		change = fmt.Sprintf("%.2f mi", delta)
	case "MOVING", "AVG PACE":
		seconds := int(math.Round(delta))
		change = fmt.Sprintf("%d:%02d", seconds/60, seconds%60)
		if label == "AVG PACE" {
			change += "/mi"
		}
	case "ELEVATION":
		change = fmt.Sprintf("%.0f ft", delta)
	case "AVG HR", "MAX HR":
		change = fmt.Sprintf("%.0f bpm", delta)
	case "AVG CADENCE":
		change = fmt.Sprintf("%.0f spm", delta)
	case "AVG POWER":
		change = fmt.Sprintf("%.0f W", delta)
	case "EFFORT":
		change = fmt.Sprintf("%.1f", delta)
	case "RELATIVE EFFORT":
		change = fmt.Sprintf("%.0f", delta)
	}
	return sign + change, direction
}
