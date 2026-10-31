package main

import (
	"context"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestUpdateMovesToNextActivity(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	m := model{ctx: ctx, cancel: cancel, loading: true, result: resultMsg{title: "Fresh Tracks"}}
	next, cmd := m.Update(updateMsg{message: "Updated: Fresh Tracks"})
	m = next.(model)
	if cmd == nil || !m.loading || m.refresh {
		t.Fatal("expected cached next-activity lookup")
	}
	next, _ = m.Update(activityMsg{detail: &activity{ID: 2, Name: "Morning Run"}})
	m = next.(model)
	if !m.editing || m.detail.ID != 2 || m.result.title != "" || m.manual != "" {
		t.Fatal("next activity must start with a fresh title")
	}
}

func TestLargeTitleSupportsDigitsAndApostrophes(t *testing.T) {
	for _, title := range []string{"Runner's 5", "Run 0123456789", "It’s 5"} {
		result := largeTitle(title, 120)
		if strings.Contains(result, title+"_") {
			t.Errorf("%q fell back to small text", title)
		}
		if got := strings.Count(result, "\n"); got != 2 {
			t.Errorf("%q: got %d line breaks, want 2", title, got)
		}
	}
}

func TestSkipMovesOnWithoutChangingActivity(t *testing.T) {
	m := model{detail: activity{ID: 42, Name: "Morning Run"}, editing: true, manual: "Draft"}
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	m = next.(model)
	if cmd == nil || !m.loading || !m.skipped[42] || m.detail.Name != "Morning Run" {
		t.Fatal("skip should load another activity and leave the title unchanged")
	}
}

func TestUpdateDoesNotContinueAfterCacheFailure(t *testing.T) {
	m := model{result: resultMsg{title: "Fresh Tracks"}}
	next, _ := m.Update(updateMsg{message: "Updated to Fresh Tracks, but could not save the cache. Run task refresh."})
	if !strings.Contains(next.(model).decision, "could not save") {
		t.Fatal("cache warning must remain visible")
	}
}
