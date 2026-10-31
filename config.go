package main

import (
	"encoding/json"
	"os"
	"strings"
)

const defaultTone = "playful and lighthearted"

type stravaTokens struct {
	Access  string `json:"access_token"`
	Refresh string `json:"refresh_token"`
}

type settings struct {
	Tone         string `json:"tone"`
	ClientID     string `json:"client_id,omitempty"`
	ClientSecret string `json:"client_secret,omitempty"`
}

const settingsFile = ".name-my-run.json"

func loadSettings() settings {
	result := settings{Tone: defaultTone}
	data, err := os.ReadFile(settingsFile)
	if err != nil {
		return result
	}
	var saved settings
	if json.Unmarshal(data, &saved) == nil {
		if strings.TrimSpace(saved.Tone) != "" {
			result.Tone = saved.Tone
		}
		result.ClientID, result.ClientSecret = saved.ClientID, saved.ClientSecret
	}
	return result
}

func saveSettings(value settings) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(settingsFile, data, 0600)
}

const tokenFile = ".strava-tokens.json"

func saveTokens(tokens stravaTokens) error {
	data, err := json.Marshal(tokens)
	if err != nil {
		return err
	}
	return os.WriteFile(tokenFile, data, 0600)
}

func loadTokens() stravaTokens {
	tokens := stravaTokens{}
	if data, err := os.ReadFile(tokenFile); err == nil {
		var saved stravaTokens
		if json.Unmarshal(data, &saved) == nil && saved.Access != "" && saved.Refresh != "" {
			tokens = saved
		}
	}
	return tokens
}
