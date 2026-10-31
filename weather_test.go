package main

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type weatherTransport func(*http.Request) (*http.Response, error)

func (f weatherTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestFetchWeatherSelectsStartHour(t *testing.T) {
	client := &http.Client{Transport: weatherTransport(func(r *http.Request) (*http.Response, error) {
		if r.URL.Query().Get("start_date") != "2025-05-01" || r.URL.Query().Get("timezone") != "auto" {
			t.Fatal("incorrect request")
		}
		body := `{"hourly":{"time":["2025-05-01T08:00","2025-05-01T09:00"],"temperature_2m":[10,15],"weather_code":[0,3],"wind_speed_10m":[null,8]}}`
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body))}, nil
	})}
	a := activity{Start: "2025-05-01T09:35:00Z", StartLatLng: []float64{45, -93}}
	got, err := fetchWeather(context.Background(), client, a, time.Date(2025, 5, 3, 0, 0, 0, 0, time.UTC))
	if err != nil || got == nil || got.Temperature == nil || *got.Temperature != 15 || got.Wind == nil || *got.Wind != 8 || got.Humidity != nil || got.Code == nil || weatherCondition(*got.Code) != "Overcast" {
		t.Fatalf("unexpected weather: %+v, %v", got, err)
	}
}

func TestFetchWeatherWithoutCoordinates(t *testing.T) {
	client := &http.Client{Transport: weatherTransport(func(*http.Request) (*http.Response, error) { t.Fatal("unexpected request"); return nil, nil })}
	got, err := fetchWeather(context.Background(), client, activity{}, time.Now())
	if got != nil || err != nil {
		t.Fatalf("unexpected result: %+v, %v", got, err)
	}
}
