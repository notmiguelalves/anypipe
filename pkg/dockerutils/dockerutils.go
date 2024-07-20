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
	ctx               context.Context
	dockerClient      *client.Client
	spawnedContainers []*Container
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

func (du *DockerUtils) Close() error {
	for _, c := range du.spawnedContainers {
		err := du.dockerClient.ContainerRemove(du.ctx, c.id, container.RemoveOptions{
			RemoveVolumes: false,
			RemoveLinks:   false,
			Force:         true,
		})

		if err != nil {
			fmt.Printf("failed to remove container %s: %s\n", c.id, err.Error())
		}
	}

	return du.dockerClient.Close()
}

func (du *DockerUtils) CreateContainer(image string) (*Container, error) {
	resp, err := du.dockerClient.ContainerCreate(du.ctx, &container.Config{
		Image: image,
		Cmd:   []string{"sleep", "infinity"},
		Tty:   false,
	}, nil, nil, nil, "")
	if err != nil {
		return nil, err
	}

	c := Container{id: resp.ID}
	du.spawnedContainers = append(du.spawnedContainers, &c)

	if err := du.dockerClient.ContainerStart(du.ctx, c.id, container.StartOptions{}); err != nil {
		return &c, err
	}

	// TODO : poll container state and only return from this function when the container is fully up and running ?

	return &c, nil
}

func (du *DockerUtils) Exec(c *Container, cmd []string) error {
	resp, err := du.dockerClient.ContainerExecCreate(du.ctx, c.id, container.ExecOptions{Cmd: cmd, Detach: false, AttachStderr: true, AttachStdout: true, WorkingDir: "/home"})
	if err != nil {
		return err
	}

	attachResp, err := du.dockerClient.ContainerExecAttach(du.ctx, resp.ID, container.ExecAttachOptions{})
	if err != nil {
		return err
	}
	defer attachResp.Close()

	err = du.dockerClient.ContainerExecStart(du.ctx, resp.ID, container.ExecStartOptions{})
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
