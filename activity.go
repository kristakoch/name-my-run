package main

type split struct {
	Distance            float64 `json:"distance"`
	MovingTime          int     `json:"moving_time"`
	AverageSpeed        float64 `json:"average_speed"`
	ElevationDifference float64 `json:"elevation_difference"`
}

type lap struct {
	Distance      *float64 `json:"distance,omitempty"`
	MovingTime    *int     `json:"moving_time,omitempty"`
	AverageSpeed  *float64 `json:"average_speed,omitempty"`
	AverageHR     *float64 `json:"average_heartrate,omitempty"`
	ElevationGain *float64 `json:"total_elevation_gain,omitempty"`
}

// Optional metrics remain absent rather than becoming invented zero values.
type activity struct {
	ID             int64     `json:"id"`
	Name           string    `json:"name"`
	Sport          string    `json:"sport_type"`
	Description    string    `json:"description,omitempty"`
	City           string    `json:"location_city,omitempty"`
	State          string    `json:"location_state,omitempty"`
	Country        string    `json:"location_country,omitempty"`
	StartLatLng    []float64 `json:"start_latlng,omitempty"`
	NearbyCity     string    `json:"nearby_city,omitempty"`
	NearbyState    string    `json:"nearby_state,omitempty"`
	NearbyCountry  string    `json:"nearby_country,omitempty"`
	Start          string    `json:"start_date_local"`
	Distance       float64   `json:"distance"`
	MovingTime     int       `json:"moving_time"`
	Elevation      float64   `json:"total_elevation_gain"`
	AverageHR      *float64  `json:"average_heartrate,omitempty"`
	MaximumHR      *float64  `json:"max_heartrate,omitempty"`
	Effort         *float64  `json:"perceived_exertion,omitempty"`
	Temperature    *float64  `json:"average_temp,omitempty"`
	Weather        *weather  `json:"weather,omitempty"`
	Splits         []split   `json:"splits_metric,omitempty"`
	Laps           []lap     `json:"laps,omitempty"`
	ElapsedTime    *int      `json:"elapsed_time,omitempty"`
	AverageSpeed   *float64  `json:"average_speed,omitempty"`
	MaxSpeed       *float64  `json:"max_speed,omitempty"`
	AverageCadence *float64  `json:"average_cadence,omitempty"`
	AverageWatts   *float64  `json:"average_watts,omitempty"`
	SufferScore    *float64  `json:"suffer_score,omitempty"`
	Calories       *float64  `json:"calories,omitempty"`
	PRCount        *int      `json:"pr_count,omitempty"`
	WorkoutType    *int      `json:"workout_type,omitempty"`
}

func isDefaultName(name string) bool {
	return name == "Morning Run" || name == "Afternoon Run" || name == "Lunch Run"
}

func isRun(a activity) bool {
	return a.Sport == "Run" || a.Sport == "TrailRun" || a.Sport == "VirtualRun"
}
