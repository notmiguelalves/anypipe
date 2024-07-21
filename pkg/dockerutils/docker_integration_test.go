//go:build integration

package dockerutils_test

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/notmiguelalves/anypipe/pkg/dockerutils"
	"github.com/stretchr/testify/assert"
)

func TestDockeruritl(t *testing.T) {
	ctx := context.Background()
	assert.NoError(t, os.MkdirAll("testdata/integration", os.ModePerm))
	defer os.RemoveAll("testdata/integration")

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	du, err := dockerutils.New(ctx, logger)
	assert.NoError(t, err)
	defer du.Close()

	container, err := du.CreateContainer("docker.io/library/alpine")
	assert.NoError(t, err)

	container.AddEnv("help", "me")
	_, _, ec, err := du.Exec(container, "echo help=${help} > /home/tmp.txt")
	assert.NoError(t, err)
	assert.Equal(t, 0, ec)

	err = du.CopyFrom(container, "/home/tmp.txt", "./testdata/integration")
	assert.NoError(t, err)

	_, _, ec, err = du.Exec(container, "mkdir /dummy")
	assert.NoError(t, err)
	assert.Equal(t, 0, ec)

	err = du.CopyTo(container, "./testdata/integration", "/dummy")
	assert.NoError(t, err)

	_, _, ec, err = du.Exec(container, "ls -l /")
	assert.NoError(t, err)
	assert.Equal(t, 0, ec)

	_, _, ec, err = du.Exec(container, "ls -l /dummy")
	assert.NoError(t, err)
	assert.Equal(t, 0, ec)

	_, _, ec, err = du.Exec(container, "cat /dummy/tmp.txt")
	assert.NoError(t, err)
	assert.Equal(t, 0, ec)

	stdout, stderr, ec, err := du.Exec(container, "echo 'Hello World!'")
	assert.NoError(t, err)
	assert.Equal(t, 0, ec)
	assert.Equal(t, "Hello World!\n", stdout.String())
	assert.Equal(t, "", stderr.String())

	stdout, stderr, ec, err = du.Exec(container, "ls somebaddir123")
	assert.NoError(t, err)
	assert.Equal(t, 1, ec)
	assert.Equal(t, "", stdout.String())
	assert.Equal(t, "ls: somebaddir123: No such file or directory\n", stderr.String())

}