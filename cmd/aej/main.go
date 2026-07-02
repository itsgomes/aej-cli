package main

import (
	"context"
	"os"
	"os/signal"

	"github.com/itsgomes/aej-cli/cmd"
	"github.com/itsgomes/aej-cli/internal/cli"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	if err := cmd.Execute(ctx); err != nil {
		cli.NewPrinter(os.Stdout, os.Stderr).Error(err.Error())
		os.Exit(1)
	}
}
