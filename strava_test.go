package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestFetchActivitiesCollectsOneHundredRunsAcrossPages(t *testing.T) {
	calls := 0
	client := &http.Client{Transport: weatherTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.URL.Query().Get("per_page") != "100" {
			t.Fatal("incorrect page size")
		}
		var batch []activity
		for i := 0; i < 100; i++ {
			sport := "Run"
			if calls == 1 && i%2 == 0 {
				sport = "Ride"
			}
			batch = append(batch, activity{ID: int64((calls-1)*100 + i + 1), Sport: sport})
		}
		data, err := json.Marshal(batch)
		if err != nil {
			t.Fatal(err)
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(string(data)))}, nil
	})}
	runs, _, err := fetchActivities(context.Background(), client)
	if err != nil || calls != 2 || len(runs) != 100 || runs[0].ID != 2 || runs[99].ID != 150 {
		t.Fatalf("unexpected result: runs=%d calls=%d err=%v", len(runs), calls, err)
	}
}

func TestFetchOlderRunsFindsAnchorAndCollectsRuns(t *testing.T) {
	calls := 0
	client := &http.Client{Transport: weatherTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		var batch []activity
		for i := 0; i < 100; i++ {
			id := (calls-1)*100 + i + 1
			sport := "Run"
			if id > 120 && id%2 == 0 {
				sport = "Ride"
			}
			batch = append(batch, activity{ID: int64(id), Sport: sport})
		}
		data, err := json.Marshal(batch)
		if err != nil {
			t.Fatal(err)
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(string(data)))}, nil
	})}
	runs, err := fetchOlderRuns(context.Background(), client, 120)
	if err != nil || calls != 3 || len(runs) != 50 || runs[0].ID != 121 || runs[49].ID != 219 {
		t.Fatalf("unexpected older runs: count=%d calls=%d err=%v", len(runs), calls, err)
	}
}
