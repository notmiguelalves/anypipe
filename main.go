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

	container.AddEnv("help", "me")
	err = du.Exec(container, "echo help=${help} > /home/tmp.txt")
	if err != nil {
		panic(err)
	}

	err = du.CopyFrom(container, "/home/tmp.txt", "./dummy")
	if err != nil {
		panic(err)
	}
}
