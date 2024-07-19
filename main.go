package main

import (
	"context"

	"github.com/notmiguelalves/anypipe/pkg/dockerutils"
)

func main() {
	ctx := context.Background()
	du, err := dockerutils.New(ctx)
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
