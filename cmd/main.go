package main

import (
	"context"
	"log/slog"
	"os"

	eagleapp "eagleeye/internal/app"
)

func main() {
	if err := eagleapp.Run(context.Background(), os.Args[1:]); err != nil {
		slog.Error("eagleeye failed", "error", err)

		os.Exit(1)
	}
}
