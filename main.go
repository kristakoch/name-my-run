package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	refresh := flag.Bool("refresh", false, "refresh the activity cache")
	flag.Parse()

	settings := loadSettings()
	if id, secret := os.Getenv("STRAVA_CLIENT_ID"), os.Getenv("STRAVA_CLIENT_SECRET"); id != "" || secret != "" {
		if id == "" || secret == "" {
			fmt.Fprintln(os.Stderr, "Provide both STRAVA_CLIENT_ID and STRAVA_CLIENT_SECRET.")
			os.Exit(1)
		}
		if settings.ClientID != "" && settings.ClientID != id {
			fmt.Fprintln(os.Stderr, "Changing applications requires a separate working directory to avoid mixing accounts.")
			os.Exit(1)
		}
		settings.ClientID, settings.ClientSecret = id, secret
		if err := saveSettings(settings); err != nil {
			fmt.Fprintln(os.Stderr, "Save credentials:", err)
			os.Exit(1)
		}
	}
	if code := os.Getenv("STRAVA_AUTH_CODE"); code != "" {
		if err := authorize(code); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println("Authorization saved. Run task run.")
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	final, err := tea.NewProgram(model{ctx: ctx, cancel: cancel, cursorVisible: true, loading: true, stage: "Loading recent activities…", refresh: *refresh}, tea.WithAltScreen()).Run()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if final.(model).result.err != nil {
		os.Exit(1)
	}
}
