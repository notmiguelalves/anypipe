package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/notmiguelalves/anypipe/pkg/dockerutils"
)

func main() {
	ctx := context.Background()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	du, err := dockerutils.New(ctx, logger)
	if err != nil {
		panic(err)
	}
	defer du.Close()

	container, err := du.CreateContainer("docker.io/library/alpine")
	if err != nil {
		panic(err)
	}

	err = du.Exec(container, []string{"echo", "hello world"})
	if err != nil {
		panic(err)
	}
}
