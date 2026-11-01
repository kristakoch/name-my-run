package main

import "testing"

func TestMetricDifference(t *testing.T) {
	currentSpeed, earlierSpeed := 4.0, 3.0
	current := activity{ID: 1, Distance: 1609.344 * 4, AverageSpeed: &currentSpeed}
	history := []activity{{ID: 1, Sport: "Run", Distance: 100000}}
	for i := 0; i < 5; i++ {
		history = append(history, activity{ID: int64(i + 2), Sport: "Run", Distance: 1609.344 * 3, AverageSpeed: &earlierSpeed})
	}
	history = append(history, activity{ID: 20, Sport: "Ride", Distance: 1609.344 * 100})
	if text, direction := metricDifference(current, history, "DISTANCE"); text != "+1.00 mi" || direction != 1 {
		t.Fatalf("distance: %q %d", text, direction)
	}
	if text, direction := metricDifference(current, history, "AVG PACE"); text != "-2:14/mi" || direction != -1 {
		t.Fatalf("pace: %q %d", text, direction)
	}
	if text, direction := metricDifference(current, history[:5], "DISTANCE"); text != "" || direction != 0 {
		t.Fatal("expected no comparison with fewer than five runs")
	}
}
