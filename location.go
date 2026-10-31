package main

import (
	"archive/zip"
	"bufio"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
)

const citiesFile = "cities5000.zip"

// nearestCity finds a nearby populated-place point, not a city boundary.
func nearestCity(coords []float64) (string, string, string, error) {
	if len(coords) != 2 || coords[0] < -90 || coords[0] > 90 || coords[1] < -180 || coords[1] > 180 {
		return "", "", "", nil
	}
	archive, err := zip.OpenReader(citiesFile)
	if err != nil {
		return "", "", "", err
	}
	defer archive.Close()
	for _, file := range archive.File {
		if file.Name != "cities5000.txt" {
			continue
		}
		reader, err := file.Open()
		if err != nil {
			return "", "", "", err
		}
		defer reader.Close()
		scan := bufio.NewScanner(reader)
		scan.Buffer(make([]byte, 4096), 1024*1024)
		best := 25.0
		var city, country, adminCode string
		for scan.Scan() {
			fields := strings.Split(scan.Text(), "\t")
			if len(fields) < 11 {
				continue
			}
			lat, e1 := strconv.ParseFloat(fields[4], 64)
			lon, e2 := strconv.ParseFloat(fields[5], 64)
			if e1 != nil || e2 != nil {
				continue
			}
			const radians = math.Pi / 180
			dlat, dlon := (lat-coords[0])*radians, (lon-coords[1])*radians
			h := math.Pow(math.Sin(dlat/2), 2) + math.Cos(coords[0]*radians)*math.Cos(lat*radians)*math.Pow(math.Sin(dlon/2), 2)
			distance := 2 * 6371 * math.Asin(math.Min(1, math.Sqrt(h)))
			if distance < best {
				best, city, country, adminCode = distance, fields[1], fields[8], fields[10]
			}
		}
		if err := scan.Err(); err != nil {
			return "", "", "", err
		}
		state := ""
		if regions, err := os.Open("admin1CodesASCII.txt"); err == nil {
			defer regions.Close()
			scanner := bufio.NewScanner(regions)
			for scanner.Scan() {
				parts := strings.Split(scanner.Text(), "\t")
				if len(parts) >= 2 && parts[0] == country+"."+adminCode {
					state = parts[1]
					break
				}
			}
		}
		return city, state, country, nil
	}
	return "", "", "", fmt.Errorf("cities5000.txt missing from %s", citiesFile)
}

func addNearbyCity(detail *activity) bool {
	if detail.City != "" || (detail.NearbyCity != "" && detail.NearbyState != "") || len(detail.StartLatLng) != 2 {
		return false
	}
	city, state, country, err := nearestCity(detail.StartLatLng)
	if err != nil || city == "" {
		return false
	}
	if detail.NearbyCity == city && detail.NearbyState == state && detail.NearbyCountry == country {
		return false
	}
	detail.NearbyCity, detail.NearbyState, detail.NearbyCountry = city, state, country
	return true
}

func stateAbbreviation(state, country string) string {
	if !strings.EqualFold(country, "US") && !strings.EqualFold(country, "United States") && !strings.EqualFold(country, "United States of America") {
		return state
	}
	states := map[string]string{
		"Alabama": "AL", "Alaska": "AK", "Arizona": "AZ", "Arkansas": "AR", "California": "CA", "Colorado": "CO", "Connecticut": "CT", "Delaware": "DE", "District of Columbia": "DC", "Florida": "FL", "Georgia": "GA", "Hawaii": "HI", "Idaho": "ID", "Illinois": "IL", "Indiana": "IN", "Iowa": "IA", "Kansas": "KS", "Kentucky": "KY", "Louisiana": "LA", "Maine": "ME", "Maryland": "MD", "Massachusetts": "MA", "Michigan": "MI", "Minnesota": "MN", "Mississippi": "MS", "Missouri": "MO", "Montana": "MT", "Nebraska": "NE", "Nevada": "NV", "New Hampshire": "NH", "New Jersey": "NJ", "New Mexico": "NM", "New York": "NY", "North Carolina": "NC", "North Dakota": "ND", "Ohio": "OH", "Oklahoma": "OK", "Oregon": "OR", "Pennsylvania": "PA", "Rhode Island": "RI", "South Carolina": "SC", "South Dakota": "SD", "Tennessee": "TN", "Texas": "TX", "Utah": "UT", "Vermont": "VT", "Virginia": "VA", "Washington": "WA", "West Virginia": "WV", "Wisconsin": "WI", "Wyoming": "WY",
	}
	for name, code := range states {
		if strings.EqualFold(strings.TrimSpace(state), name) {
			return code
		}
	}
	return state
}
