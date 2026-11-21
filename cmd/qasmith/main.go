package main

import (
	"context"
	"os"
	"os/signal"

	"github.com/quvly/Qasmith/internal/cli"
)

func main() {
	if len(os.Args) < 2 || os.Args[1] != "qaoa" && os.Args[1] != "tsp" && os.Args[1] != "run" && os.Args[1] != "optimize" && os.Args[1] != "csv-to-smith" {
		os.Exit(cli.Run(os.Args[1:], os.Stdout, os.Stderr))
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	code := cli.RunContext(ctx, os.Args[1:], os.Stdout, os.Stderr)
	stop()
	os.Exit(code)
}
