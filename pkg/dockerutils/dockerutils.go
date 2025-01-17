package dockerutils

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/notmiguelalves/anypipe/pkg/utils"
	"github.com/notmiguelalves/anypipe/pkg/wrapper"
)

//go:generate mockgen -destination=dockerutils_mock.go -package=dockerutils -source=dockerutils.go DockerUtils
type DockerUtils interface {
	Close() error
	CreateContainer(image string) (*Container, error)
	Exec(c *Container, cmd string) (stdout, stderr string, exitcode int, err error)
	CopyTo(c *Container, srcPath, dstPath string) error
	CopyFrom(c *Container, srcPath, dstPath string) error
	CopyBetweenContainers(srcContainer, destContainer *Container, srcPath, dstPath string) error
}

type DockerUtilsImpl struct {
	logger            *slog.Logger
	dockerClient      wrapper.DockerClient
	spawnedContainers []*Container
}

// initializes a DockerUtils client - make sure to defer a call to Close() the client on exit
func New(ctx context.Context, logger *slog.Logger) (*DockerUtilsImpl, error) {
	cli, err := wrapper.NewClientWithOpts(ctx, client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		logger.Error(fmt.Sprintf("failed to initialize docker client: %s", err.Error()))
		return nil, err
	}

	return &DockerUtilsImpl{
		dockerClient: cli,
		logger:       logger,
	}, nil
}

// initialize a DockerUtils client while providing a pre-created Docker client
func NewWithClient(logger *slog.Logger, cli wrapper.DockerClient) *DockerUtilsImpl {
	return &DockerUtilsImpl{
		dockerClient: cli,
		logger:       logger,
	}
}

// closes the DockerUtils client, and removes all containers created by the client during program execution
func (du *DockerUtilsImpl) Close() error {
	du.logger.Debug("cleaning up spawned containers")

	for _, c := range du.spawnedContainers {
		du.logger.Debug(fmt.Sprintf("going to cleanup %s", c.id))

		err := du.dockerClient.ContainerRemove(c.id, container.RemoveOptions{
			RemoveVolumes: false,
			RemoveLinks:   false,
			Force:         true,
		})

		if err != nil {
			du.logger.Error(fmt.Sprintf("failed to cleanup container %s: %s", c.id, err.Error()))
		}
	}

	return du.dockerClient.Close()
}

// pull an image by ref. returns 'nil' if succeeds or if image is already present
func (du *DockerUtilsImpl) pullImage(img string) error {
	rc, err := du.dockerClient.ImagePull(img, image.PullOptions{})
	if err != nil {
		du.logger.Error(fmt.Sprintf("failed to pull image %s : %s", img, err.Error()))
		return err
	}
	defer rc.Close()

	bOut, err := io.ReadAll(rc)
	if err != nil {
		du.logger.Error(fmt.Sprintf("failed to read image pull output for %s : %s", img, err.Error()))
		return err
	}

	lines := strings.Split(string(bOut), "\r\n")
	for _, l := range lines {
		du.logger.Debug(strings.ReplaceAll(l, "\"", "'"))
	}

	return nil
}

// creates a container with the specified image
func (du *DockerUtilsImpl) CreateContainer(image string) (*Container, error) {
	err := du.pullImage(image)
	if err != nil {
		du.logger.Error(fmt.Sprintf("failed to pull image %s : %s", image, err.Error()))
		return nil, err
	}

	resp, err := du.dockerClient.ContainerCreate(&container.Config{
		Image: image,
		Cmd:   []string{"sleep", "infinity"},
		Tty:   false,
	})
	if err != nil {
		du.logger.Error(fmt.Sprintf("failed to create container from '%s' : %s", image, err.Error()))
		return nil, err
	}

	c := Container{
		id:  resp.ID,
		env: map[string]string{},
	}
	du.spawnedContainers = append(du.spawnedContainers, &c)

	du.logger.Debug(fmt.Sprintf("going to start container %s created from image %s", c.id, image))
	if err := du.dockerClient.ContainerStart(c.id, container.StartOptions{}); err != nil {
		du.logger.Error(fmt.Sprintf("failed to start container '%s' : %s", c.id, err.Error()))
		return &c, err
	}

	du.logger.Debug(fmt.Sprintf("started container %s", c.id))
	return &c, nil
}

// executes the specified command on the provided container. Note: command will be executed with `sh -c <command>`
func (du *DockerUtilsImpl) Exec(c *Container, cmd string) (stdout, stderr string, exitcode int, err error) {
	du.logger.Debug(fmt.Sprintf("going to execute %s on container %s", cmd, c.id))

	shcmd := []string{"sh", "-c", cmd}

	resp, err := du.dockerClient.ContainerExecCreate(c.id, container.ExecOptions{Cmd: shcmd, Env: c.Env(), Detach: false, AttachStderr: true, AttachStdout: true, WorkingDir: "/home"})
	if err != nil {
		du.logger.Error("failed to create exec operation on container %s : %s", c.id, err.Error())
		return
	}

	attachResp, err := du.dockerClient.ContainerExecAttach(resp.ID, container.ExecAttachOptions{})
	if err != nil {
		du.logger.Error("failed to attach exec operation to container %s : %s", c.id, err.Error())
		return
	}
	defer attachResp.Close()

	err = du.dockerClient.ContainerExecStart(resp.ID, container.ExecStartOptions{})
	if err != nil {
		du.logger.Error("failed to start exec operation on container %s : %s", c.id, err.Error())
		return
	}

	bOut := bytes.NewBuffer([]byte{})
	bErr := bytes.NewBuffer([]byte{})

	_, err = stdcopy.StdCopy(bOut, bErr, attachResp.Reader)
	if err != nil {
		du.logger.Error(fmt.Sprintf("failed to fetch container stdout/stderr from container %s : %s", c.id, err.Error()))
		return
	}

	stdout = bOut.String()
	stderr = bErr.String()

	execInfo, err := du.dockerClient.ContainerExecInspect(resp.ID)
	if err != nil {
		du.logger.Error(fmt.Sprintf("failed to inspect container exec operation for %s : %s", c.id, err.Error()))
		return
	}

	exitcode = execInfo.ExitCode

	return
}

// copies a file or directory from the host to a container
func (du *DockerUtilsImpl) CopyTo(c *Container, srcPath, dstPath string) error {
	buf, err := utils.Tar(srcPath)
	if err != nil {
		du.logger.Error(fmt.Sprintf("failed to tar %s : %s", srcPath, err.Error()))
		return err
	}

	err = du.dockerClient.CopyToContainer(c.id, dstPath, buf, container.CopyToContainerOptions{})
	if err != nil {
		du.logger.Error(fmt.Sprintf("failed to copy %s to container %s : %s", srcPath, c.id, err.Error()))
		return err
	}

	return nil
}

// copies a file or directory from a container to the host
func (du *DockerUtilsImpl) CopyFrom(c *Container, srcPath, dstPath string) error {
	rc, _, err := du.dockerClient.CopyFromContainer(c.id, srcPath)
	if err != nil {
		du.logger.Error(fmt.Sprintf("failed to copy %s from container %s : %s", srcPath, c.id, err.Error()))
		return err
	}

	err = utils.Untar(rc, dstPath)
	if err != nil {
		du.logger.Error(fmt.Sprintf("failed to untar %s : %s", srcPath, err.Error()))
		return err
	}

	return nil
}

// copies a file or directory between two containers
func (du *DockerUtilsImpl) CopyBetweenContainers(srcContainer, destContainer *Container, srcPath, dstPath string) error {
	rc, _, err := du.dockerClient.CopyFromContainer(srcContainer.id, srcPath)
	if err != nil {
		du.logger.Error(fmt.Sprintf("failed to copy %s from container %s : %s", srcPath, srcContainer.id, err.Error()))
		return err
	}

	err = du.dockerClient.CopyToContainer(destContainer.id, dstPath, rc, container.CopyToContainerOptions{})
	if err != nil {
		du.logger.Error(fmt.Sprintf("failed to copy %s to container %s : %s", srcPath, destContainer.id, err.Error()))
		return err
	}

	return nil
}
