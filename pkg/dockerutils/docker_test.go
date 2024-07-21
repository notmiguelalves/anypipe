package dockerutils

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"log/slog"
	"net"
	"os"
	"strings"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/notmiguelalves/anypipe/pkg/wrapper"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestClose(t *testing.T) {
	ctrl := gomock.NewController(t)
	testLogger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	t.Run("close client with no spawned containers", func(t *testing.T) {
		mockClient := wrapper.NewMockDockerClient(ctrl)
		du := NewWithClient(testLogger, mockClient)

		mockClient.EXPECT().ContainerRemove(gomock.Any(), gomock.Any()).Times(0)
		mockClient.EXPECT().Close().Times(1).Return(nil)

		assert.NoError(t, du.Close())
	})

	t.Run("close client with spawned containers, fail to remove", func(t *testing.T) {
		mockClient := wrapper.NewMockDockerClient(ctrl)
		du := NewWithClient(testLogger, mockClient)
		du.spawnedContainers = []*Container{
			{
				id: "123",
			},
			{
				id: "321",
			},
		}

		mockClient.EXPECT().ContainerRemove("123", gomock.Any()).Times(1).Return(errors.New("some error"))
		mockClient.EXPECT().ContainerRemove("321", gomock.Any()).Times(1).Return(errors.New("some error"))
		mockClient.EXPECT().Close().Times(1).Return(nil)

		assert.NoError(t, du.Close())
	})

	t.Run("close client with spawned containers, successfully remove", func(t *testing.T) {
		mockClient := wrapper.NewMockDockerClient(ctrl)
		du := NewWithClient(testLogger, mockClient)
		du.spawnedContainers = []*Container{
			{
				id: "123",
			},
			{
				id: "321",
			},
		}

		mockClient.EXPECT().ContainerRemove("123", gomock.Any()).Times(1).Return(nil)
		mockClient.EXPECT().ContainerRemove("321", gomock.Any()).Times(1).Return(nil)
		mockClient.EXPECT().Close().Times(1).Return(nil)

		assert.NoError(t, du.Close())
	})

}

func TestPullImage(t *testing.T) {
	ctrl := gomock.NewController(t)
	testLogger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	t.Run("failed to pull", func(t *testing.T) {
		mockClient := wrapper.NewMockDockerClient(ctrl)
		du := NewWithClient(testLogger, mockClient)

		mockClient.EXPECT().ImagePull("someref", gomock.Any()).Times(1).Return(nil, errors.New("some error"))

		assert.Error(t, du.pullImage("someref"))
	})

	t.Run("happy path", func(t *testing.T) {
		mockClient := wrapper.NewMockDockerClient(ctrl)
		du := NewWithClient(testLogger, mockClient)

		mockClient.EXPECT().ImagePull("someref", gomock.Any()).Times(1).Return(io.NopCloser(strings.NewReader("done")), nil)

		assert.NoError(t, du.pullImage("someref"))
	})
}

func TestCreateContainer(t *testing.T) {
	ctrl := gomock.NewController(t)
	testLogger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	t.Run("failed to pull image", func(t *testing.T) {
		mockClient := wrapper.NewMockDockerClient(ctrl)
		du := NewWithClient(testLogger, mockClient)

		mockClient.EXPECT().ImagePull("someref", gomock.Any()).Times(1).Return(nil, errors.New("some error"))

		_, err := du.CreateContainer("someref")
		assert.Error(t, err)
	})

	t.Run("failed to create container", func(t *testing.T) {
		mockClient := wrapper.NewMockDockerClient(ctrl)
		du := NewWithClient(testLogger, mockClient)

		mockClient.EXPECT().ImagePull("someref", gomock.Any()).Times(1).Return(io.NopCloser(strings.NewReader("done")), nil)
		mockClient.EXPECT().ContainerCreate(&container.Config{
			Image: "someref",
			Cmd:   []string{"sleep", "infinity"},
			Tty:   false,
		}).Times(1).Return(container.CreateResponse{}, errors.New("some error"))

		_, err := du.CreateContainer("someref")
		assert.Error(t, err)
	})

	t.Run("failed to start container", func(t *testing.T) {
		mockClient := wrapper.NewMockDockerClient(ctrl)
		du := NewWithClient(testLogger, mockClient)

		mockClient.EXPECT().ImagePull("someref", gomock.Any()).Times(1).Return(io.NopCloser(strings.NewReader("done")), nil)

		mockClient.EXPECT().ContainerCreate(&container.Config{
			Image: "someref",
			Cmd:   []string{"sleep", "infinity"},
			Tty:   false,
		}).Times(1).Return(container.CreateResponse{ID: "123"}, nil)

		mockClient.EXPECT().ContainerStart("123", gomock.Any()).Times(1).Return(errors.New("some error"))

		assert.Len(t, du.spawnedContainers, 0)
		_, err := du.CreateContainer("someref")
		assert.Error(t, err)
		assert.Len(t, du.spawnedContainers, 1)
	})

	t.Run("happy path", func(t *testing.T) {
		mockClient := wrapper.NewMockDockerClient(ctrl)
		du := NewWithClient(testLogger, mockClient)

		mockClient.EXPECT().ImagePull("someref", gomock.Any()).Times(1).Return(io.NopCloser(strings.NewReader("done")), nil)

		mockClient.EXPECT().ContainerCreate(&container.Config{
			Image: "someref",
			Cmd:   []string{"sleep", "infinity"},
			Tty:   false,
		}).Times(1).Return(container.CreateResponse{ID: "123"}, nil)

		mockClient.EXPECT().ContainerStart("123", gomock.Any()).Times(1).Return(nil)

		assert.Len(t, du.spawnedContainers, 0)
		c, err := du.CreateContainer("someref")
		assert.NoError(t, err)
		assert.Equal(t, "123", c.id)
		assert.Len(t, du.spawnedContainers, 1)
	})
}

func TestExec(t *testing.T) {
	ctrl := gomock.NewController(t)
	testLogger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	t.Run("failed to create exec operation", func(t *testing.T) {
		mockClient := wrapper.NewMockDockerClient(ctrl)
		du := NewWithClient(testLogger, mockClient)
		c := &Container{id: "123"}
		createResp := types.IDResponse{ID: "777"}

		mockClient.EXPECT().ContainerExecCreate("123", gomock.Any()).Times(1).Return(createResp, errors.New("some error"))

		assert.Error(t, du.Exec(c, "echo test"))
	})

	t.Run("failed to attach to container", func(t *testing.T) {
		mockClient := wrapper.NewMockDockerClient(ctrl)
		du := NewWithClient(testLogger, mockClient)
		c := &Container{id: "123"}
		createResp := types.IDResponse{ID: "777"}
		attachResp := types.HijackedResponse{}

		mockClient.EXPECT().ContainerExecCreate("123", gomock.Any()).Times(1).Return(createResp, nil)
		mockClient.EXPECT().ContainerExecAttach(createResp.ID, gomock.Any()).Times(1).Return(attachResp, errors.New("some error"))

		assert.Error(t, du.Exec(c, "echo test"))
	})

	t.Run("failed to start exec operation", func(t *testing.T) {
		mockClient := wrapper.NewMockDockerClient(ctrl)
		du := NewWithClient(testLogger, mockClient)
		c := &Container{id: "123"}
		createResp := types.IDResponse{ID: "777"}
		conn1, conn2 := net.Pipe() // a bit of an ugly way to get a dummy/mock net.Conn
		defer conn2.Close()
		attachResp := types.HijackedResponse{Conn: conn1}

		mockClient.EXPECT().ContainerExecCreate("123", gomock.Any()).Times(1).Return(createResp, nil)
		mockClient.EXPECT().ContainerExecAttach(createResp.ID, gomock.Any()).Times(1).Return(attachResp, nil)
		mockClient.EXPECT().ContainerExecStart(createResp.ID, gomock.Any()).Times(1).Return(errors.New("some error"))

		assert.Error(t, du.Exec(c, "echo test"))
	})

	t.Run("happy path", func(t *testing.T) {
		mockClient := wrapper.NewMockDockerClient(ctrl)
		du := NewWithClient(testLogger, mockClient)
		c := &Container{id: "123"}
		createResp := types.IDResponse{ID: "777"}
		conn1, conn2 := net.Pipe() // a bit of an ugly way to get a dummy/mock net.Conn
		defer conn2.Close()
		buf := bytes.Buffer{}
		attachResp := types.HijackedResponse{Conn: conn1, Reader: bufio.NewReader(&buf)}

		mockClient.EXPECT().ContainerExecCreate("123", gomock.Any()).Times(1).Return(createResp, nil)
		mockClient.EXPECT().ContainerExecAttach(createResp.ID, gomock.Any()).Times(1).Return(attachResp, nil)
		mockClient.EXPECT().ContainerExecStart(createResp.ID, gomock.Any()).Times(1).Return(nil)

		assert.NoError(t, du.Exec(c, "echo test"))
	})
}

func TestCopyTo(t *testing.T) {
	ctrl := gomock.NewController(t)
	testLogger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	t.Run("failed to copy", func(t *testing.T) {
		mockClient := wrapper.NewMockDockerClient(ctrl)
		du := NewWithClient(testLogger, mockClient)
		c := &Container{id: "123"}

		tarbytes, err := os.ReadFile("../utils/testdata/tmp.tar")
		assert.NoError(t, err)

		mockClient.EXPECT().CopyToContainer("123", "/dst/tmp.txt", bytes.NewBuffer(tarbytes), container.CopyToContainerOptions{}).Times(1).Return(errors.New("some error"))

		assert.Error(t, du.CopyTo(c, "../utils/testdata/tmp.txt", "/dst/tmp.txt"))
	})

	t.Run("happy path", func(t *testing.T) {
		mockClient := wrapper.NewMockDockerClient(ctrl)
		du := NewWithClient(testLogger, mockClient)
		c := &Container{id: "123"}

		tarbytes, err := os.ReadFile("../utils/testdata/tmp.tar")
		assert.NoError(t, err)

		mockClient.EXPECT().CopyToContainer("123", "/dst/tmp.txt", bytes.NewBuffer(tarbytes), container.CopyToContainerOptions{}).Times(1).Return(nil)

		assert.NoError(t, du.CopyTo(c, "../utils/testdata/tmp.txt", "/dst/tmp.txt"))
	})

}

func TestCopyFrom(t *testing.T) {
	ctrl := gomock.NewController(t)
	testLogger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	t.Run("failed to copy", func(t *testing.T) {
		mockClient := wrapper.NewMockDockerClient(ctrl)
		du := NewWithClient(testLogger, mockClient)
		c := &Container{id: "123"}

		mockClient.EXPECT().CopyFromContainer("123", "/dst/tmp.txt").Times(1).Return(nil, container.PathStat{}, errors.New("some error"))

		assert.Error(t, du.CopyFrom(c, "/dst/tmp.txt", "testdata/result.txt"))
	})

	t.Run("happy path", func(t *testing.T) {
		mockClient := wrapper.NewMockDockerClient(ctrl)
		du := NewWithClient(testLogger, mockClient)
		c := &Container{id: "123"}

		tarbytes, err := os.ReadFile("../utils/testdata/tmp.tar")
		assert.NoError(t, err)

		rc := io.NopCloser(bytes.NewBuffer(tarbytes))

		mockClient.EXPECT().CopyFromContainer("123", "/dst/tmp.txt").Times(1).Return(rc, container.PathStat{}, nil)

		defer os.RemoveAll("testdata/result.txt")
		assert.NoError(t, du.CopyFrom(c, "/dst/tmp.txt", "testdata/result.txt"))

		expected, err := os.ReadFile("testdata/expected.txt")
		assert.NoError(t, err)

		result, err := os.ReadFile("testdata/result.txt")
		assert.NoError(t, err)

		assert.Equal(t, expected, result)
	})
}
