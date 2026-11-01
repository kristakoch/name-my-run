package main

import (
	"context"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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

func TestNameCardStartsWideAndFitsTerminal(t *testing.T) {
	for _, width := range []int{40, 80, 120} {
		empty := lipgloss.Width(nameCard("", width, true, true))
		typed := lipgloss.Width(nameCard("Morning", width, true, true))
		if empty != typed || empty > width {
			t.Fatalf("terminal %d: empty=%d typed=%d", width, empty, typed)
		}
	}
}

func TestLargeTitleSupportsDigitsAndApostrophes(t *testing.T) {
	for _, title := range []string{"Runner's 5", "Run 0123456789", "It’s 5"} {
		result := largeTitle(title, 120, true)
		if strings.Contains(result, title+"▏") {
			t.Errorf("%q fell back to small text", title)
		}
		if got := strings.Count(result, "\n"); got != 2 {
			t.Errorf("%q: got %d line breaks, want 2", title, got)
		}
	}
}

func TestMetricComparisonAlignment(t *testing.T) {
	current := activity{ID: 1, Distance: 1609.344 * 4, MovingTime: 3000}
	var history []activity
	for i := 0; i < 5; i++ {
		history = append(history, activity{ID: int64(i + 2), Sport: "Run", Distance: 1609.344 * 3, MovingTime: 2000})
	}
	var positions []int
	for _, line := range strings.Split(activityMetrics(current, history, 40), "\n") {
		if at := strings.Index(line, "▲"); at >= 0 {
			positions = append(positions, at)
		}
	}
	if len(positions) < 2 || positions[0] != positions[1] {
		t.Fatalf("comparison arrows not aligned: %v", positions)
	}
}

func TestActivityInformationAlignsWithMetrics(t *testing.T) {
	a := activity{ID: 1, Name: "Morning Run", Start: "2025-01-01T09:00:00Z", NearbyCity: "Minneapolis", NearbyState: "Minnesota", NearbyCountry: "US", Distance: 5000, MovingTime: 1800}
	_, column := activityMetricsLayout(a, nil, 100)
	lines := strings.Split(activityInformation(a, a.Name, 100, column), "\n")
	if column == 0 || len(lines) < 2 || strings.Index(lines[0], "ON ") != column || strings.Index(lines[1], "IN ") != column {
		t.Fatalf("activity details do not align at column %d: %q", column, lines)
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

func TestEndOfRunsPromptsBeforeLoading(t *testing.T) {
	m := model{ctx: context.Background(), loading: true}
	next, cmd := m.Update(activityMsg{history: []activity{{ID: 1, Name: "Custom"}}})
	m = next.(model)
	if !m.askOlder || m.loading || cmd != nil {
		t.Fatal("expected a prompt without a network request")
	}
	next, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	m = next.(model)
	if !m.loading || m.askOlder || cmd == nil {
		t.Fatal("expected older-runs request after approval")
	}
}

func TestBlinkingCursorKeepsCardSize(t *testing.T) {
	for _, title := range []string{"", "Morning", "☀"} {
		visible := nameCard(title, 80, true, true)
		hidden := nameCard(title, 80, true, false)
		if !strings.Contains(visible, "▏") || strings.Contains(hidden, "▏") {
			t.Errorf("cursor visibility incorrect for %q", title)
		}
		if lipgloss.Width(visible) != lipgloss.Width(hidden) || lipgloss.Height(visible) != lipgloss.Height(hidden) {
			t.Errorf("card dimensions change when cursor blinks for %q", title)
		}
	}
	next, cmd := (model{}).Update(cursorBlinkMsg{})
	if !next.(model).cursorVisible || cmd == nil {
		t.Fatal("cursor should toggle and schedule the next blink")
	}
}
