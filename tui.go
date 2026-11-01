package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
	"unicode"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	stravaOrange = lipgloss.Color("#FC4C02")
	muted        = lipgloss.Color("#C8C8C8")

	appTitleStyle = lipgloss.NewStyle().Foreground(stravaOrange).Bold(true)
	labelStyle    = lipgloss.NewStyle().Foreground(muted).Bold(true)
	currentStyle  = lipgloss.NewStyle().Foreground(muted)
	titleStyle    = lipgloss.NewStyle().Foreground(muted).Bold(true)
	linkStyle     = lipgloss.NewStyle().Foreground(stravaOrange)
	suggestStyle  = lipgloss.NewStyle().Foreground(stravaOrange).Bold(true).Padding(0, 1)
	cardStyle     = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(stravaOrange).Padding(1, 2)
	metricStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#EDEDED")).Padding(0, 1).Border(lipgloss.NormalBorder()).BorderRight(true).BorderForeground(muted)
	keyStyle      = lipgloss.NewStyle().Foreground(stravaOrange).Bold(true)
	warningStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#F5C542"))
	successStyle  = lipgloss.NewStyle().Foreground(stravaOrange).Bold(true)
	errorStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5F56"))
)

type updateMsg struct {
	message string
	err     error
}

type activityMsg struct {
	detail  *activity
	history []activity
	err     error
}

type weatherMsg struct {
	id      int64
	weather *weather
}

type olderRunsMsg struct {
	count int
	err   error
}

type resultMsg struct {
	original, title, message string
	err                      error
}

type model struct {
	ctx      context.Context
	cancel   context.CancelFunc
	result   resultMsg
	loading  bool
	decision string
	detail   activity
	editing  bool
	manual   string
	warning  string
	stage    string
	refresh  bool
	history  []activity
	width    int
	notice   string
	skipped  map[int64]bool
	askOlder bool
}

func (m model) Init() tea.Cmd {
	return func() tea.Msg {
		client := &http.Client{Timeout: 30 * time.Second}
		latest, err := nextDefaultActivity(m.ctx, client, m.refresh, m.skipped)
		if err != nil {
			return activityMsg{err: err}
		}
		cache, err := loadActivityCache()
		return activityMsg{detail: latest, history: cache.Activities, err: err}
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		return m, nil
	case activityMsg:
		if msg.err != nil {
			m.loading = false
			m.result.err = msg.err
			return m, tea.Quit
		}
		if msg.detail == nil {
			m.loading = false
			if len(msg.history) > 0 {
				m.askOlder = true
				m.editing = false
				return m, nil
			}
			m.result.message = "No runs found."
			return m, tea.Quit
		}
		m.detail = *msg.detail
		m.history = msg.history
		m.result = resultMsg{original: m.detail.Name}
		m.loading = false
		m.editing = true
		m.manual = ""
		if m.detail.Weather == nil && len(m.detail.StartLatLng) == 2 {
			detail, ctx := m.detail, m.ctx
			return m, func() tea.Msg {
				forecast, _ := fetchWeather(ctx, &http.Client{Timeout: 8 * time.Second}, detail, time.Now())
				return weatherMsg{id: detail.ID, weather: forecast}
			}
		}
	case olderRunsMsg:
		m.loading = false
		if msg.err != nil {
			m.warning = msg.err.Error()
			m.askOlder = true
			return m, nil
		}
		if msg.count == 0 {
			m.result.message = "No older runs found."
			return m, tea.Quit
		}
		m.notice = fmt.Sprintf("Loaded %d older runs.", msg.count)
		m.loading = true
		m.stage = "Finding the next activity…"
		m.refresh = false
		return m, m.Init()
	case weatherMsg:
		if msg.id != m.detail.ID || msg.weather == nil || m.loading {
			return m, nil
		}
		m.detail.Weather = msg.weather
		if cache, err := loadActivityCache(); err == nil {
			if detail, ok := cache.Details[msg.id]; ok {
				detail.Weather = msg.weather
				cache.Details[msg.id] = detail
				_ = saveActivityCache(cache)
			}
		}
	case updateMsg:
		m.loading = false
		if msg.err != nil {
			m.warning = msg.err.Error()
			m.editing = true
			return m, nil
		}
		// A cache error or a default replacement could select the same activity again.
		if !strings.HasPrefix(msg.message, "Updated: ") || isDefaultName(m.result.title) {
			m.decision = msg.message
			return m, tea.Quit
		}
		m.notice = msg.message
		m.warning = ""
		m.loading = true
		m.stage = "Loading next activity…"
		m.refresh = false
		return m, m.Init()
	case tea.KeyMsg:
		if m.askOlder {
			switch msg.String() {
			case "y", "Y":
				m.askOlder, m.loading, m.warning = false, true, ""
				m.stage = "Loading older runs…"
				ctx := m.ctx
				return m, func() tea.Msg {
					count, err := loadOlderRuns(ctx, &http.Client{Timeout: 30 * time.Second})
					return olderRunsMsg{count: count, err: err}
				}
			case "n", "N", "q", "esc", "ctrl+c":
				m.decision = "Finished for now."
				return m, tea.Quit
			}
			return m, nil
		}
		if m.editing {
			return m.updateEditor(msg)
		}
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			if m.loading && m.stage == "Updating activity…" {
				return m, nil
			}
			m.cancel()
			m.loading = false
			m.decision = "Cancelled. No changes made."
			return m, tea.Quit
		case "e", "E":
			if !m.loading {
				m.editing = true
				m.manual = m.result.title
				m.warning = ""
			}
		case "y", "Y":
			if !m.loading && m.result.title != "" {
				m.loading = true
				m.stage = "Updating activity…"
				detail, title, ctx := m.detail, m.result.title, m.ctx
				return m, func() tea.Msg {
					message, err := renameActivity(ctx, &http.Client{Timeout: 30 * time.Second}, detail, title)
					return updateMsg{message: message, err: err}
				}
			}
		case "n", "N":
			if !m.loading && m.result.title != "" {
				m.decision = "Skipped. No changes made."
				return m, tea.Quit
			}

		}
	}
	return m, nil
}

func (m model) updateEditor(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		m.decision = "Skipped. No changes made."
		return m, tea.Quit
	case tea.KeyCtrlC:
		m.cancel()
		m.decision = "Cancelled. No changes made."
		return m, tea.Quit
	case tea.KeyCtrlS:
		if m.skipped == nil {
			m.skipped = make(map[int64]bool)
		}
		m.skipped[m.detail.ID] = true
		m.editing = false
		m.loading = true
		m.refresh = false
		m.stage = "Loading next activity…"
		m.notice = "Skipped this activity for now."
		m.warning = ""
		return m, m.Init()
	case tea.KeyBackspace, tea.KeyDelete:
		r := []rune(m.manual)
		if len(r) > 0 {
			m.manual = string(r[:len(r)-1])
		}
	case tea.KeyEnter:
		title := strings.TrimSpace(m.manual)
		if len(strings.Fields(title)) == 0 || len(strings.Fields(title)) > 5 || strings.ContainsFunc(title, unicode.IsControl) {
			m.warning = "Enter a title of one to five words."
			return m, nil
		}
		m.result.title = title
		m.editing = false
		m.warning = ""
		m.loading = true
		m.stage = "Updating activity…"
		detail, ctx := m.detail, m.ctx
		return m, func() tea.Msg {
			message, err := renameActivity(ctx, &http.Client{Timeout: 30 * time.Second}, detail, title)
			return updateMsg{message: message, err: err}
		}
	case tea.KeyRunes:
		m.manual += string(msg.Runes)
	case tea.KeySpace:
		m.manual += " "
	}
	return m, nil
}

func (m model) View() string {
	if m.loading {
		return appScreen(lipgloss.JoinVertical(lipgloss.Left,
			labelStyle.Render("WORKING"),
			lipgloss.NewStyle().Foreground(stravaOrange).Render("●")+" "+m.stage,
		))
	}
	if m.result.err != nil {
		return appScreen(errorStyle.Render("Something went wrong") + "\n" + m.result.err.Error())
	}
	if m.result.message != "" {
		return appScreen(labelStyle.Render("ALL CAUGHT UP") + "\n" + m.result.message)
	}
	if m.decision != "" {
		return appScreen(successStyle.Render("✓ " + m.decision))
	}
	if m.askOlder {
		warning := ""
		if m.warning != "" {
			warning = warningStyle.Render(m.warning) + "\n\n"
		}
		notice := ""
		if m.notice != "" {
			notice = successStyle.Render(m.notice) + "\n\n"
		}
		return appScreen(notice + warning + "No more loaded runs with default names.\n\nLoad the next 50 older runs?\n\n" + keyboardHint("y", "load more") + "  " + keyboardHint("n", "finish"))
	}
	metrics, secondColumn := activityMetricsLayout(m.detail, m.history, m.width)
	heading := "ACTIVITY"
	for i, run := range m.history {
		if run.ID == m.detail.ID {
			heading += fmt.Sprintf("  %d/%d", i+1, len(m.history))
			break
		}
	}
	if m.editing {
		warning := ""
		if m.warning != "" {
			warning = warningStyle.Render(m.warning) + "\n\n"
		}
		notice := ""
		if m.notice != "" {
			notice = successStyle.Render("✓ "+m.notice) + "\n\n"
		}
		return appScreen(notice + warning + labelStyle.Render(heading) + "\n" + activityInformation(m.detail, m.result.original, m.width, secondColumn) + "\n\n" + cardStyle.Render(labelStyle.Render("NEW NAME")+"\n"+largeTitle(m.manual, m.width)) + "\n\n" + metrics + "\n\n" + keyboardHint("enter", "update activity") + "  " + keyboardHint("ctrl+s", "skip") + "  " + keyboardHint("esc", "quit"))
	}
	warning := ""
	if m.warning != "" {
		warning = warningStyle.Render(m.warning) + "\n\n"
	}
	activityInfo := activityInformation(m.detail, m.result.original, m.width, secondColumn)
	cardLabel := "NEW NAME"
	cardText := m.result.title
	controls := keyboardHint("y", "approve & update") + "  " + keyboardHint("n", "skip") + "  " + keyboardHint("e", "edit") + "  " + keyboardHint("q", "quit")
	content := lipgloss.JoinVertical(lipgloss.Left,
		warning+labelStyle.Render(heading),
		activityInfo,
		"",
		cardStyle.Render(labelStyle.Render(cardLabel)+"\n"+suggestStyle.Render(cardText)),
		"",
		metrics,
		"",
		controls,
	)
	return appScreen(content)
}

func activityInformation(detail activity, original string, width, secondColumn int) string {
	if width <= 0 {
		width = 80
	}
	when := strings.TrimSuffix(activityWhen(detail), "\n")
	location := activityLocation(detail)
	activityInfo := titleStyle.Render(original)
	rightWidth := max(lipgloss.Width(when), lipgloss.Width(location))
	description := strings.TrimSpace(detail.Description)
	if rightWidth > 0 && secondColumn > 0 && lipgloss.Width(original)+2 <= secondColumn && secondColumn+rightWidth <= width {
		leftWidth := secondColumn
		activityInfo = lipgloss.NewStyle().Width(leftWidth).Render(activityInfo) + when
		if location != "" {
			firstLine := ""
			if description != "" {
				wrapped := strings.Split(lipgloss.NewStyle().Width(leftWidth-2).Render(description), "\n")
				firstLine = wrapped[0]
				description = strings.Join(wrapped[1:], "\n")
			}
			activityInfo += "\n" + lipgloss.NewStyle().Width(leftWidth).Render(firstLine) + location
		}
	} else {
		if when != "" {
			activityInfo += "\n" + when
		}
		if location != "" {
			activityInfo += "\n" + location
		}
	}
	if description != "" {
		activityInfo += "\n" + lipgloss.NewStyle().Width(width).Render(description)
	}
	if detail.ID > 0 {
		activityInfo += "\n" + linkStyle.Render(fmt.Sprintf("View on Strava: https://www.strava.com/activities/%d", detail.ID))
	}
	return activityInfo
}

func activityLocation(a activity) string {
	var parts []string
	if a.City == "" && a.NearbyCity != "" {
		for _, part := range []string{a.NearbyCity, stateAbbreviation(a.NearbyState, a.NearbyCountry), a.NearbyCountry} {
			if strings.TrimSpace(part) != "" {
				parts = append(parts, strings.TrimSpace(part))
			}
		}
		return labelStyle.Render("IN") + " " + strings.Join(parts, ", ")
	}
	for _, part := range []string{a.City, stateAbbreviation(a.State, a.Country), a.Country} {
		if strings.TrimSpace(part) != "" {
			parts = append(parts, strings.TrimSpace(part))
		}
	}
	if len(parts) == 0 {
		return ""
	}
	return labelStyle.Render("IN") + " " + strings.Join(parts, ", ")
}

// Terminals use fixed-size character cells, so render a compact block preview.
// Fall back to normal text when the block font cannot represent the title.
func largeTitle(title string, width int) string {
	font := map[rune][5]string{
		'A': {" ██ ", "█  █", "████", "█  █", "█  █"}, 'B': {"███ ", "█  █", "███ ", "█  █", "███ "},
		'C': {" ███", "█   ", "█   ", "█   ", " ███"}, 'D': {"███ ", "█  █", "█  █", "█  █", "███ "},
		'E': {"████", "█   ", "███ ", "█   ", "████"}, 'F': {"████", "█   ", "███ ", "█   ", "█   "},
		'G': {" ███", "█   ", "█ ██", "█  █", " ███"}, 'H': {"█  █", "█  █", "████", "█  █", "█  █"},
		'I': {"███", " █ ", " █ ", " █ ", "███"}, 'J': {"  ██", "   █", "   █", "█  █", " ██ "},
		'K': {"█  █", "█ █ ", "██  ", "█ █ ", "█  █"}, 'L': {"█   ", "█   ", "█   ", "█   ", "████"},
		'M': {"█   █", "██ ██", "█ █ █", "█   █", "█   █"}, 'N': {"█  █", "██ █", "█ ██", "█  █", "█  █"},
		'O': {" ██ ", "█  █", "█  █", "█  █", " ██ "}, 'P': {"███ ", "█  █", "███ ", "█   ", "█   "},
		'Q': {" ██ ", "█  █", "█  █", "█ ██", " ███"}, 'R': {"███ ", "█  █", "███ ", "█ █ ", "█  █"},
		'S': {" ███", "█   ", " ██ ", "   █", "███ "}, 'T': {"█████", "  █  ", "  █  ", "  █  ", "  █  "},
		'U': {"█  █", "█  █", "█  █", "█  █", " ██ "}, 'V': {"█   █", "█   █", "█   █", " █ █ ", "  █  "},
		'W': {"█   █", "█   █", "█ █ █", "██ ██", "█   █"}, 'X': {"█   █", " █ █ ", "  █  ", " █ █ ", "█   █"},
		'Y': {"█   █", " █ █ ", "  █  ", "  █  ", "  █  "}, 'Z': {"████", "  █ ", " █  ", "█   ", "████"},
		'a': {"    ", " ██ ", "   █", " ███", "█ ██"}, 'b': {"█   ", "█   ", "███ ", "█  █", "███ "},
		'c': {"    ", " ██ ", "█   ", "█   ", " ██ "}, 'd': {"   █", "   █", " ███", "█  █", " ███"},
		'e': {"    ", " ██ ", "████", "█   ", " ███"}, 'f': {"  ██", " █  ", "███ ", " █  ", " █  "},
		'g': {"    ", " ███", "█  █", " ███", "   █"}, 'h': {"█   ", "█   ", "███ ", "█  █", "█  █"},
		'i': {" █ ", "   ", "██ ", " █ ", "███"}, 'j': {"  █", "   ", "  █", "  █", "██ "},
		'k': {"█   ", "█  █", "█ █ ", "██  ", "█ █ "}, 'l': {"██ ", " █ ", " █ ", " █ ", "███"},
		'm': {"     ", "██ █ ", "█ █ █", "█ █ █", "█ █ █"}, 'n': {"    ", "███ ", "█  █", "█  █", "█  █"},
		'o': {"    ", " ██ ", "█  █", "█  █", " ██ "}, 'p': {"    ", "███ ", "█  █", "███ ", "█   "},
		'q': {"    ", " ███", "█  █", " ███", "   █"}, 'r': {"    ", "█ ██", "██  ", "█   ", "█   "},
		's': {"    ", " ███", " ██ ", "   █", "███ "}, 't': {" █  ", "███ ", " █  ", " █  ", "  ██"},
		'u': {"    ", "█  █", "█  █", "█  █", " ███"}, 'v': {"     ", "█   █", "█   █", " █ █ ", "  █  "},
		'w': {"     ", "█   █", "█ █ █", "█ █ █", " █ █ "}, 'x': {"     ", "█   █", " █ █ ", " █ █ ", "█   █"},
		'y': {"     ", "█   █", " █ █ ", "  █  ", " █   "}, 'z': {"    ", "████", "  █ ", " █  ", "████"},
		'0': {" ██ ", "█  █", "█  █", "█  █", " ██ "}, '1': {" █ ", "██ ", " █ ", " █ ", "███"},
		'2': {"███ ", "   █", " ██ ", "█   ", "████"}, '3': {"███ ", "   █", " ██ ", "   █", "███ "},
		'4': {"█  █", "█  █", "████", "   █", "   █"}, '5': {"████", "█   ", "███ ", "   █", "███ "},
		'6': {" ██ ", "█   ", "███ ", "█  █", " ██ "}, '7': {"████", "   █", "  █ ", " █  ", " █  "},
		'8': {" ██ ", "█  █", " ██ ", "█  █", " ██ "}, '9': {" ██ ", "█  █", " ███", "   █", " ██ "},
		'\'': {"█", "█", " ", " ", " "}, '’': {"█", "█", " ", " ", " "},
		'-': {"   ", "   ", "███", "   ", "   "}, '!': {"█", "█", "█", " ", "█"},
		'.': {" ", " ", " ", " ", "█"}, ',': {" ", " ", " ", "█", "█"},
		' ': {"  ", "  ", "  ", "  ", "  "},
	}
	preview := suggestStyle.Render(title+"_") + "\n \n "
	if title == "" {
		return preview
	}
	var lines [5]string
	for _, char := range title {
		glyph, ok := font[char]
		if !ok {
			return preview
		}
		for row := range lines {
			lines[row] += glyph[row] + " "
		}
	}
	if width <= 0 {
		width = 80
	}
	if lipgloss.Width(lines[0])+6 > width {
		return preview
	}
	// Half-block characters pack five pixel rows into three terminal rows.
	var compact [3]strings.Builder
	for pair := range compact {
		top := []rune(lines[pair*2])
		bottom := make([]rune, len(top))
		for i := range bottom {
			bottom[i] = ' '
		}
		if pair*2+1 < len(lines) {
			bottom = []rune(lines[pair*2+1])
		}
		for i := range top {
			switch {
			case top[i] != ' ' && bottom[i] != ' ':
				compact[pair].WriteRune('█')
			case top[i] != ' ':
				compact[pair].WriteRune('▀')
			case bottom[i] != ' ':
				compact[pair].WriteRune('▄')
			default:
				compact[pair].WriteRune(' ')
			}
		}
	}
	return lipgloss.NewStyle().Foreground(stravaOrange).Bold(true).Render(compact[0].String() + "\n" + compact[1].String() + "\n" + compact[2].String())
}

func activityWhen(a activity) string {
	if start, err := time.Parse(time.RFC3339, a.Start); err == nil {
		return labelStyle.Render("ON") + " " + start.Format("Monday, Jan 2, 2006  3:04 PM") + "\n"
	}
	return ""
}

func activityMetrics(a activity, history []activity, width int) string {
	metrics, _ := activityMetricsLayout(a, history, width)
	return metrics
}

func activityMetricsLayout(a activity, history []activity, width int) (string, int) {
	type entry struct {
		label, value, change string
	}
	var items []entry
	var weatherItems []entry
	add := func(label, value string) { items = append(items, entry{label: label, value: value}) }
	addWeather := func(label, value string) { weatherItems = append(weatherItems, entry{label: label, value: value}) }
	var averageHR, maximumHR, relativeEffort, averagePower *entry
	add("DISTANCE", fmt.Sprintf("%.2f mi", a.Distance/1609.344))
	add("MOVING", fmt.Sprintf("%d:%02d", a.MovingTime/60, a.MovingTime%60))
	add("ELEVATION", fmt.Sprintf("%.0f ft", a.Elevation*3.28084))
	if a.AverageSpeed != nil && *a.AverageSpeed > 0 {
		seconds := int(1609.344 / *a.AverageSpeed + 0.5)
		add("AVG PACE", fmt.Sprintf("%d:%02d /mi", seconds/60, seconds%60))
	} else if a.Distance > 0 && a.MovingTime > 0 {
		seconds := int(float64(a.MovingTime)*1609.344/a.Distance + 0.5)
		add("AVG PACE", fmt.Sprintf("%d:%02d /mi", seconds/60, seconds%60))
	}
	if a.AverageCadence != nil && *a.AverageCadence > 0 {
		add("AVG CADENCE", fmt.Sprintf("%.0f spm", *a.AverageCadence*2))
	}
	if a.AverageHR != nil {
		averageHR = &entry{label: "AVG HR", value: fmt.Sprintf("%.0f bpm", *a.AverageHR)}
	}
	if a.MaximumHR != nil {
		maximumHR = &entry{label: "MAX HR", value: fmt.Sprintf("%.0f bpm", *a.MaximumHR)}
	}
	if a.Effort != nil {
		add("EFFORT", fmt.Sprintf("%g/10", *a.Effort))
	}
	if a.Temperature != nil {
		addWeather("RECORDED TEMP", fmt.Sprintf("%.0f °F", *a.Temperature*9/5+32))
	}
	if a.Weather != nil {
		if a.Weather.Temperature != nil {
			addWeather("TEMP", fmt.Sprintf("%.0f °F", *a.Weather.Temperature*9/5+32))
		}
		if a.Weather.FeelsLike != nil {
			addWeather("FEELS LIKE", fmt.Sprintf("%.0f °F", *a.Weather.FeelsLike*9/5+32))
		}
		if a.Weather.Humidity != nil {
			addWeather("HUMIDITY", fmt.Sprintf("%.0f%%", *a.Weather.Humidity))
		}
		if a.Weather.Wind != nil {
			addWeather("WIND", fmt.Sprintf("%.0f mph", *a.Weather.Wind*0.621371))
		}
		if a.Weather.Precipitation != nil {
			addWeather("PRECIPITATION", fmt.Sprintf("%.2f in", *a.Weather.Precipitation/25.4))
		}
		if a.Weather.Code != nil {
			if condition := weatherCondition(*a.Weather.Code); condition != "" {
				addWeather("CONDITIONS", condition)
			}
		}
	}
	if a.SufferScore != nil {
		relativeEffort = &entry{label: "RELATIVE EFFORT", value: fmt.Sprintf("%g", *a.SufferScore)}
	}
	if a.AverageWatts != nil {
		averagePower = &entry{label: "AVG POWER", value: fmt.Sprintf("%.0f W", *a.AverageWatts)}
	}
	decorate := func(item *entry) {
		if item == nil {
			return
		}
		change, direction := metricDifference(a, history, item.label)
		if direction == 0 {
			return
		}
		arrow := "▲"
		if (direction < 0) != (item.label == "AVG PACE") {
			arrow = "▼"
		}
		color := lipgloss.Color("#65D68A")
		if (direction < 0) != (item.label == "AVG PACE") {
			color = lipgloss.Color("#FF7777")
		}
		item.change = lipgloss.NewStyle().Foreground(color).Render(arrow + " " + change)
	}
	for i := range items {
		decorate(&items[i])
	}
	for _, item := range []*entry{averageHR, maximumHR, relativeEffort, averagePower} {
		decorate(item)
	}
	if width <= 0 {
		width = 80
	}
	// Use aligned columns and a separate line for the date so labels and values
	// remain easy to scan even as optional metrics appear or disappear.
	var lines []string
	comparisonRuns := 0
	for _, run := range history {
		if run.ID != a.ID && (run.Sport == "Run" || run.Sport == "TrailRun" || run.Sport == "VirtualRun") {
			comparisonRuns++
		}
	}
	if comparisonRuns >= 5 {
		lines = append(lines, labelStyle.Render(fmt.Sprintf("Compared with up to %d cached runs (varies by metric)", comparisonRuns)))
	}
	valueWidth := 0
	for _, item := range items {
		if item.change != "" && lipgloss.Width(item.value) > valueWidth {
			valueWidth = lipgloss.Width(item.value)
		}
	}
	for _, item := range []*entry{averageHR, maximumHR, relativeEffort, averagePower} {
		if item != nil && item.change != "" && lipgloss.Width(item.value) > valueWidth {
			valueWidth = lipgloss.Width(item.value)
		}
	}
	cellText := func(item entry) string {
		value := item.value
		if item.change != "" {
			value += strings.Repeat(" ", valueWidth-lipgloss.Width(item.value)+2) + item.change
		}
		return labelStyle.Render(fmt.Sprintf("%-15s", item.label)) + " " + value
	}
	columnWidth := 0
	for _, item := range items {
		if n := lipgloss.Width(cellText(item)); n > columnWidth {
			columnWidth = n
		}
	}
	for _, item := range weatherItems {
		if n := 16 + lipgloss.Width(item.value); n > columnWidth {
			columnWidth = n
		}
	}
	for _, item := range []*entry{averageHR, maximumHR, relativeEffort, averagePower} {
		if item != nil {
			n := lipgloss.Width(cellText(*item))
			if n > columnWidth {
				columnWidth = n
			}
		}
	}
	columns := 1
	if width >= 2*columnWidth+6 {
		columns = 2
	}
	if width >= 3*columnWidth+12 {
		columns = 3
	}
	for i := 0; i < len(items); {
		end := i + columns
		if end > len(items) {
			end = len(items)
		}
		var row []string
		for _, item := range items[i:end] {
			text := cellText(item)
			row = append(row, lipgloss.NewStyle().Width(columnWidth+3).Render(text))
		}
		lines = append(lines, strings.Join(row, ""))
		i = end
	}
	metricCell := func(item *entry) string {
		if item == nil {
			return lipgloss.NewStyle().Width(columnWidth + 3).Render("")
		}
		text := cellText(*item)
		return lipgloss.NewStyle().Width(columnWidth + 3).Render(text)
	}
	if columns == 1 {
		for _, item := range []*entry{averageHR, maximumHR, relativeEffort, averagePower} {
			if item != nil {
				lines = append(lines, metricCell(item))
			}
		}
	} else {
		if averageHR != nil || relativeEffort != nil {
			lines = append(lines, metricCell(averageHR)+metricCell(relativeEffort))
		}
		if maximumHR != nil || averagePower != nil {
			lines = append(lines, metricCell(maximumHR)+metricCell(averagePower))
		}
	}
	if len(weatherItems) > 0 {
		lines = append(lines, "")
		for i := 0; i < len(weatherItems); i += columns {
			end := min(i+columns, len(weatherItems))
			var row []string
			for _, item := range weatherItems[i:end] {
				item := item
				row = append(row, metricCell(&item))
			}
			lines = append(lines, strings.Join(row, ""))
		}
	}
	secondColumn := 0
	if columns > 1 {
		secondColumn = columnWidth + 3
	}
	return strings.Join(lines, "\n"), secondColumn
}

func appScreen(content string) string {
	header := appTitleStyle.Render("RUN RENAMER") + " · " + currentStyle.Render("COMPATIBLE WITH STRAVA")
	return "\n" + header + "\n" + lipgloss.NewStyle().Foreground(stravaOrange).Render("━━━━━━━━━━━━━━━━━━━━━━━━") + "\n\n" + content + "\n"
}

func keyboardHint(key, label string) string {
	return keyStyle.Render("["+key+"]") + " " + labelStyle.Render(label)
}
