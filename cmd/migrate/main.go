package main

import (
	"fmt"
	"os"

	"github.com/kilobyteno/dagr-chat/internal/config"
	"github.com/kilobyteno/dagr-chat/internal/dbmigrate"
)

func main() {
	cfg := config.Load()
	logger := cfg.NewLogger()

	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: migrate <up|down>")
		os.Exit(2)
	}

	var err error
	switch os.Args[1] {
	case "up":
		err = dbmigrate.Up(cfg.DatabaseURL)
	case "down":
		err = dbmigrate.Down(cfg.DatabaseURL)
	default:
		fmt.Fprintln(os.Stderr, "usage: migrate <up|down>")
		os.Exit(2)
	}

	if err != nil {
		logger.Error("migrate failed", "direction", os.Args[1], "error", err)
		os.Exit(1)
	}

	logger.Info("migrations applied", "direction", os.Args[1])
}
