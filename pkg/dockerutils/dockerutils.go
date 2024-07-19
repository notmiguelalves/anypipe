package dockerutils

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
)

type DockerUtils struct {
	ctx          context.Context
	dockerClient *client.Client
}

func New(ctx context.Context) (*DockerUtils, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}

	return &DockerUtils{
		ctx:          ctx,
		dockerClient: cli,
	}, nil
}

func (nd *DockerUtils) Close() error {
	return nd.dockerClient.Close()
}

func (nd *DockerUtils) CreateContainer(image string) (*Container, error) {
	resp, err := nd.dockerClient.ContainerCreate(nd.ctx, &container.Config{
		Image: image,
		Cmd:   []string{"sleep", "infinity"},
		Tty:   false,
	}, nil, nil, nil, "")
	if err != nil {
		return nil, err
	}

	c := Container{id: resp.ID}

	if err := nd.dockerClient.ContainerStart(nd.ctx, c.id, container.StartOptions{}); err != nil {
		return &c, err
	}

	// TODO : poll container state and only return from this function when the container is fully up and running ?

	return &c, nil
}

func (nd *DockerUtils) Exec(c *Container, cmd []string) error {
	resp, err := nd.dockerClient.ContainerExecCreate(nd.ctx, c.id, container.ExecOptions{Cmd: cmd, Detach: false, AttachStderr: true, AttachStdout: true, WorkingDir: "/home"})
	if err != nil {
		return err
	}

	attachResp, err := nd.dockerClient.ContainerExecAttach(nd.ctx, resp.ID, container.ExecAttachOptions{})
	if err != nil {
		return err
	}
	defer attachResp.Close()

	err = nd.dockerClient.ContainerExecStart(nd.ctx, resp.ID, container.ExecStartOptions{})
	if err != nil {
		return err
	}

	errChan := make(chan error, 1)

	go func() {
		_, errorInContainer := stdcopy.StdCopy(os.Stdout, os.Stderr, attachResp.Reader)
		errChan <- errorInContainer
	}()

	go func() {
		_, err := io.Copy(attachResp.Conn, os.Stdin)
		errChan <- err
	}()

	select {
	case errorInContainer := <-errChan:
		if errorInContainer != nil {
			fmt.Printf("dtools exec error: %s\n", errorInContainer.Error())
			os.Exit(-1)
		}
		return nil
	}
}

type Container struct {
	id string
}
