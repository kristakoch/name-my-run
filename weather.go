package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

type weather struct {
	Temperature   *float64 `json:"temperature_c,omitempty"`
	FeelsLike     *float64 `json:"feels_like_c,omitempty"`
	Humidity      *float64 `json:"humidity_percent,omitempty"`
	Wind          *float64 `json:"wind_kmh,omitempty"`
	Precipitation *float64 `json:"precipitation_mm,omitempty"`
	Code          *int     `json:"code,omitempty"`
}

func fetchWeather(ctx context.Context, client *http.Client, a activity, now time.Time) (*weather, error) {
	if len(a.StartLatLng) != 2 || a.StartLatLng[0] < -90 || a.StartLatLng[0] > 90 || a.StartLatLng[1] < -180 || a.StartLatLng[1] > 180 {
		return nil, nil
	}
	start, err := time.Parse(time.RFC3339, a.Start)
	if err != nil {
		return nil, nil
	}
	date := start.Format("2006-01-02")
	endpoint := "https://archive-api.open-meteo.com/v1/archive"
	if now.Sub(start) < 80*24*time.Hour {
		endpoint = "https://api.open-meteo.com/v1/forecast"
	}
	query := url.Values{
		"latitude": {fmt.Sprintf("%.5f", a.StartLatLng[0])}, "longitude": {fmt.Sprintf("%.5f", a.StartLatLng[1])},
		"start_date": {date}, "end_date": {date}, "timezone": {"auto"},
		"hourly": {"temperature_2m,apparent_temperature,relative_humidity_2m,wind_speed_10m,precipitation,weather_code"},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint+"?"+query.Encode(), nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("weather service HTTP %d", resp.StatusCode)
	}
	var data struct {
		Hourly struct {
			Time          []string   `json:"time"`
			Temperature   []*float64 `json:"temperature_2m"`
			FeelsLike     []*float64 `json:"apparent_temperature"`
			Humidity      []*float64 `json:"relative_humidity_2m"`
			Wind          []*float64 `json:"wind_speed_10m"`
			Precipitation []*float64 `json:"precipitation"`
			Code          []*int     `json:"weather_code"`
		} `json:"hourly"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}
	for i, hour := range data.Hourly.Time {
		if hour != start.Format("2006-01-02T15:00") {
			continue
		}
		w := &weather{}
		if i < len(data.Hourly.Temperature) {
			w.Temperature = data.Hourly.Temperature[i]
		}
		if i < len(data.Hourly.FeelsLike) {
			w.FeelsLike = data.Hourly.FeelsLike[i]
		}
		if i < len(data.Hourly.Humidity) {
			w.Humidity = data.Hourly.Humidity[i]
		}
		if i < len(data.Hourly.Wind) {
			w.Wind = data.Hourly.Wind[i]
		}
		if i < len(data.Hourly.Precipitation) {
			w.Precipitation = data.Hourly.Precipitation[i]
		}
		if i < len(data.Hourly.Code) {
			w.Code = data.Hourly.Code[i]
		}
		if w.Temperature == nil && w.FeelsLike == nil && w.Humidity == nil && w.Wind == nil && w.Precipitation == nil && w.Code == nil {
			return nil, nil
		}
		return w, nil
	}
	return nil, nil
}

func weatherCondition(code int) string {
	switch {
	case code == 0:
		return "Clear"
	case code == 1:
		return "Mostly clear"
	case code == 2:
		return "Partly cloudy"
	case code == 3:
		return "Overcast"
	case code == 45 || code == 48:
		return "Fog"
	case code >= 51 && code <= 57:
		return "Drizzle"
	case code >= 61 && code <= 67:
		return "Rain"
	case code >= 71 && code <= 77:
		return "Snow"
	case code >= 80 && code <= 82:
		return "Rain showers"
	case code == 85 || code == 86:
		return "Snow showers"
	case code >= 95 && code <= 99:
		return "Thunderstorm"
	default:
		return ""
	}
}
