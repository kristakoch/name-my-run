package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

// The list is reused until explicitly refreshed; detailed activities are keyed by ID.
type activityCache struct {
	HistoryLoaded bool               `json:"history_loaded"`
	Activities    []activity         `json:"activities"`
	Details       map[int64]activity `json:"details"`
}

const cacheFile = ".strava-activities.json"

func loadOlderRuns(ctx context.Context, client *http.Client) (int, error) {
	cache, err := loadActivityCache()
	if err != nil {
		return 0, err
	}
	if len(cache.Activities) == 0 {
		return 0, nil
	}
	older, err := fetchOlderRuns(ctx, client, cache.Activities[len(cache.Activities)-1].ID)
	if err != nil {
		return 0, err
	}
	seen := make(map[int64]bool, len(cache.Activities))
	for _, item := range cache.Activities {
		seen[item.ID] = true
	}
	added := 0
	for _, item := range older {
		if !seen[item.ID] {
			cache.Activities = append(cache.Activities, item)
			seen[item.ID] = true
			added++
		}
	}
	if added > 0 {
		if err := saveActivityCache(cache); err != nil {
			return 0, err
		}
	}
	return added, nil
}

func loadActivityCache() (activityCache, error) {
	cache := activityCache{Details: make(map[int64]activity)}
	data, err := os.ReadFile(cacheFile)
	if os.IsNotExist(err) {
		return cache, nil
	}
	if err != nil {
		return cache, fmt.Errorf("read activity cache: %w", err)
	}
	if err := json.Unmarshal(data, &cache); err != nil {
		return cache, fmt.Errorf("parse activity cache: %w", err)
	}
	if cache.Details == nil {
		cache.Details = make(map[int64]activity)
	}
	return cache, nil
}

func saveActivityCache(cache activityCache) error {
	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return err
	}
	file, err := os.CreateTemp(".", ".strava-cache-*")
	if err != nil {
		return fmt.Errorf("create activity cache: %w", err)
	}
	defer os.Remove(file.Name())
	if err := file.Chmod(0600); err != nil {
		file.Close()
		return err
	}
	if _, err := file.Write(data); err != nil {
		file.Close()
		return fmt.Errorf("write activity cache: %w", err)
	}
	if err := file.Close(); err != nil {
		return err
	}
	if err := os.Rename(file.Name(), cacheFile); err != nil {
		return fmt.Errorf("save activity cache: %w", err)
	}
	return nil
}

func nextDefaultActivity(ctx context.Context, client *http.Client, refresh bool, skipped map[int64]bool) (*activity, error) {
	cache, err := loadActivityCache()
	if err != nil {
		return nil, err
	}
	token := ""
	if refresh || !cache.HistoryLoaded {
		activities, accessToken, err := fetchActivities(ctx, client)
		if err != nil {
			return nil, err
		}
		cache.Activities, cache.HistoryLoaded, token = activities, true, accessToken
		if err := saveActivityCache(cache); err != nil {
			return nil, err
		}
	}
	if len(cache.Activities) == 0 {
		return nil, nil
	}
	var selected *activity
	for i := range cache.Activities {
		if isDefaultName(cache.Activities[i].Name) && !skipped[cache.Activities[i].ID] {
			selected = &cache.Activities[i]
			break
		}
	}
	if selected == nil {
		return nil, nil
	}
	if refresh {
		delete(cache.Details, selected.ID)
	}
	if detail, ok := cache.Details[selected.ID]; ok {
		if addNearbyCity(&detail) {
			cache.Details[selected.ID] = detail
			if err := saveActivityCache(cache); err != nil {
				return nil, err
			}
		}
		return &detail, nil
	}
	if token == "" {
		token = loadTokens().Access
	}
	detail, err := fetchDetails(ctx, client, selected.ID, token)
	if err != nil {
		return nil, err
	}
	addNearbyCity(&detail)
	cache.Details[selected.ID] = detail
	if err := saveActivityCache(cache); err != nil {
		return nil, err
	}
	return &detail, nil
}
