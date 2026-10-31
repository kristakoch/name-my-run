package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

const (
	activityLimit = 50
	stravaURL     = "https://www.strava.com/api/v3"
)

// Never include request bodies or credentials in errors.
func requestJSON(ctx context.Context, client *http.Client, method, endpoint, contentType, token string, body []byte, result any) (int, error) {
	req, err := http.NewRequestWithContext(ctx, method, endpoint, bytes.NewReader(body))
	if err != nil {
		return 0, fmt.Errorf("create request: %w", err)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	resp, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return resp.StatusCode, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
		return resp.StatusCode, fmt.Errorf("decode response: %w", err)
	}
	return resp.StatusCode, nil
}

func refreshAccessToken(ctx context.Context, client *http.Client, refreshToken string) (string, error) {
	config := loadSettings()
	if config.ClientID == "" || config.ClientSecret == "" || refreshToken == "" {
		return "", fmt.Errorf("Account is not connected; follow the setup instructions in README.md")
	}
	values := url.Values{
		"client_id": {config.ClientID}, "client_secret": {config.ClientSecret},
		"refresh_token": {refreshToken}, "grant_type": {"refresh_token"},
	}
	var tokens stravaTokens
	_, err := requestJSON(ctx, client, http.MethodPost, "https://www.strava.com/oauth/token", "application/x-www-form-urlencoded", "", []byte(values.Encode()), &tokens)
	if err != nil {
		return "", fmt.Errorf("refresh access token: %w", err)
	}
	if tokens.Access == "" {
		return "", fmt.Errorf("Received an empty access token")
	}
	if tokens.Refresh == "" {
		return "", fmt.Errorf("Received an empty refresh token")
	}
	if err := saveTokens(tokens); err != nil {
		return "", fmt.Errorf("save refreshed tokens: %w", err)
	}
	return tokens.Access, nil
}

func fetchActivities(ctx context.Context, client *http.Client) ([]activity, string, error) {
	tokens := loadTokens()
	token := tokens.Access
	endpoint := fmt.Sprintf("%s/athlete/activities?page=1&per_page=%d", stravaURL, activityLimit)
	var activities []activity
	status, err := requestJSON(ctx, client, http.MethodGet, endpoint, "", token, nil, &activities)
	if status == http.StatusUnauthorized {
		token, err = refreshAccessToken(ctx, client, tokens.Refresh)
		if err != nil {
			return nil, "", err
		}
		status, err = requestJSON(ctx, client, http.MethodGet, endpoint, "", token, nil, &activities)
		if status == http.StatusUnauthorized {
			return nil, "", fmt.Errorf("Activity access denied; authorize activity:read_all as described in the instructions")
		}
	}
	if err != nil {
		return nil, "", fmt.Errorf("fetch latest activity: %w", err)
	}
	return activities, token, nil
}

func fetchDetails(ctx context.Context, client *http.Client, id int64, token string) (activity, error) {
	var detail activity
	endpoint := fmt.Sprintf("%s/activities/%d", stravaURL, id)
	status, err := requestJSON(ctx, client, http.MethodGet, endpoint, "", token, nil, &detail)
	if status == http.StatusUnauthorized {
		token, err = refreshAccessToken(ctx, client, loadTokens().Refresh)
		if err != nil {
			return detail, err
		}
		_, err = requestJSON(ctx, client, http.MethodGet, endpoint, "", token, nil, &detail)
	}
	if err != nil {
		return detail, fmt.Errorf("fetch activity details: %w", err)
	}
	return detail, nil
}

// Recheck the current name before updating, since the cache may be stale.
func renameActivity(ctx context.Context, client *http.Client, detail activity, title string) (string, error) {
	current, err := fetchDetails(ctx, client, detail.ID, loadTokens().Access)
	if err != nil {
		return "", fmt.Errorf("check current activity title: %w", err)
	}
	if current.Name != detail.Name {
		return "", fmt.Errorf("activity title changed to %q; run task refresh before trying again", current.Name)
	}
	body, err := json.Marshal(map[string]string{"name": title})
	if err != nil {
		return "", err
	}
	endpoint := fmt.Sprintf("%s/activities/%d", stravaURL, detail.ID)
	var updated activity
	status, err := requestJSON(ctx, client, http.MethodPut, endpoint, "application/json", loadTokens().Access, body, &updated)
	if status == http.StatusUnauthorized || status == http.StatusForbidden {
		return "", fmt.Errorf("Update denied (HTTP %d); authorize activity:write and try again", status)
	}
	if err != nil {
		return "", fmt.Errorf("update activity: %w; if the request timed out, check the activity before retrying", err)
	}
	if updated.ID != detail.ID || updated.Name != title {
		return "", fmt.Errorf("Unexpected update response; check the activity before retrying")
	}
	cache, err := loadActivityCache()
	if err != nil {
		return fmt.Sprintf("Updated to %q, but could not read the cache: %v. Run task refresh.", title, err), nil
	}
	for i := range cache.Activities {
		if cache.Activities[i].ID == detail.ID {
			cache.Activities[i].Name = updated.Name
		}
	}
	updated.NearbyCity, updated.NearbyState, updated.NearbyCountry = detail.NearbyCity, detail.NearbyState, detail.NearbyCountry
	updated.Weather = detail.Weather
	cache.Details[detail.ID] = updated
	if err := saveActivityCache(cache); err != nil {
		return fmt.Sprintf("Updated to %q, but could not save the cache: %v. Run task refresh.", title, err), nil
	}
	return fmt.Sprintf("Updated: %s", title), nil
}

func authorize(code string) error {
	config := loadSettings()
	if config.ClientID == "" || config.ClientSecret == "" {
		return fmt.Errorf("set your client ID and secret first; see README.md")
	}
	values := url.Values{"client_id": {config.ClientID}, "client_secret": {config.ClientSecret}, "code": {code}, "grant_type": {"authorization_code"}}
	var tokens stravaTokens
	_, err := requestJSON(context.Background(), &http.Client{Timeout: 30 * time.Second}, http.MethodPost, "https://www.strava.com/oauth/token", "application/x-www-form-urlencoded", "", []byte(values.Encode()), &tokens)
	if err != nil {
		return fmt.Errorf("exchange authorization code: %w", err)
	}
	if tokens.Access == "" || tokens.Refresh == "" {
		return fmt.Errorf("Received incomplete tokens")
	}
	// A new authorization might be for a different athlete. Refresh the list
	// before using cached activities with these credentials.
	cache, err := loadActivityCache()
	if err != nil {
		return err
	}
	cache.HistoryLoaded = false
	if err := saveActivityCache(cache); err != nil {
		return fmt.Errorf("invalidate activity cache: %w", err)
	}
	return saveTokens(tokens)
}
