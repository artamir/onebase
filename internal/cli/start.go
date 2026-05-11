package cli

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/ivantit66/onebase/internal/launcher"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Open the information bases launcher",
	RunE:  runStart,
}

func runStart(_ *cobra.Command, _ []string) error {
	store, err := launcher.NewStore()
	if err != nil {
		return fmt.Errorf("start: store: %w", err)
	}

	runner := launcher.NewRunner()

	srv, err := launcher.NewServer(store, runner)
	if err != nil {
		return fmt.Errorf("start: server: %w", err)
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil {
			fmt.Fprintln(os.Stderr, "launcher server:", err)
		}
	}()

	// Force exit after 3s regardless of lingering threads.
	go func() {
		time.Sleep(3 * time.Second)
		os.Exit(0)
	}()

	// OpenWindow blocks until the window/browser is closed or /quit is called.
	// For the webview build it MUST run on the main goroutine (Win32 requirement).
	_ = launcher.OpenWindow(srv.URL(), "onebase — Информационные базы", srv.Done())

	srv.Close()
	os.Exit(0)
	return nil
}
