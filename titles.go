package main

import "fmt"

type metric struct {
	name    string
	current float64
	values  []float64
}

func personalStandouts(current activity, history []activity) []string {
	metrics := []metric{
		{name: "distance", current: current.Distance},
		{name: "moving time", current: float64(current.MovingTime)},
		{name: "elevation gain", current: current.Elevation},
	}
	if current.AverageSpeed != nil {
		metrics = append(metrics, metric{name: "average speed", current: *current.AverageSpeed})
	}
	if current.AverageHR != nil {
		metrics = append(metrics, metric{name: "average heart rate", current: *current.AverageHR})
	}
	if current.Effort != nil {
		metrics = append(metrics, metric{name: "perceived effort", current: *current.Effort})
	}

	for i := range metrics {
		for _, run := range history {
			if run.ID == current.ID || (run.Sport != "Run" && run.Sport != "TrailRun" && run.Sport != "VirtualRun") {
				continue
			}
			var value float64
			switch metrics[i].name {
			case "distance":
				value = run.Distance
			case "moving time":
				value = float64(run.MovingTime)
			case "elevation gain":
				value = run.Elevation
			case "average speed":
				if run.AverageSpeed == nil {
					continue
				}
				value = *run.AverageSpeed
			case "average heart rate":
				if run.AverageHR == nil {
					continue
				}
				value = *run.AverageHR
			case "perceived effort":
				if run.Effort == nil {
					continue
				}
				value = *run.Effort
			}
			if value > 0 {
				metrics[i].values = append(metrics[i].values, value)
			}
		}
	}

	type standout struct {
		text  string
		score float64
	}
	var found []standout
	for _, item := range metrics {
		if len(item.values) < 5 || item.current <= 0 {
			continue
		}
		var total float64
		for _, value := range item.values {
			total += value
		}
		average := total / float64(len(item.values))
		if average == 0 {
			continue
		}
		delta := (item.current - average) / average
		if delta < 0.15 {
			continue
		}
		found = append(found, standout{
			text:  fmt.Sprintf("%s was %.0f%% above your average across %d cached runs", item.name, delta*100, len(item.values)),
			score: delta,
		})
	}

	for i := 0; i < len(found); i++ {
		for j := i + 1; j < len(found); j++ {
			if found[j].score > found[i].score {
				found[i], found[j] = found[j], found[i]
			}
		}
	}
	result := make([]string, len(found))
	for i, item := range found {
		result[i] = item.text
	}
	return result
}
