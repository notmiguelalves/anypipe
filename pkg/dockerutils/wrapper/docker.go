package wrapper

//go:generate mockgen -destination=docker_mock.go -package=wrapper -source=docker.go DockerClient

import (
	"context"
	"io"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
)

type DockerClient interface {
	ContainerRemove(containerID string, options container.RemoveOptions) error
	ImagePull(refStr string, options image.PullOptions) (io.ReadCloser, error)
	ContainerCreate(config *container.Config) (container.CreateResponse, error)
	ContainerStart(containerID string, options container.StartOptions) error
	ContainerExecCreate(container string, options container.ExecOptions) (types.IDResponse, error)
	ContainerExecAttach(execID string, config container.ExecAttachOptions) (types.HijackedResponse, error)
	ContainerExecStart(execID string, config container.ExecStartOptions) error
	CopyToContainer(containerID, dstPath string, content io.Reader, options container.CopyToContainerOptions) error
	CopyFromContainer(containerID, srcPath string) (io.ReadCloser, container.PathStat, error)
	Close() error
}

type WrapperClient struct {
	ctx          context.Context
	dockerClient *client.Client
}

func NewClientWithOpts(ctx context.Context, ops ...client.Opt) (DockerClient, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}

	return &WrapperClient{
		ctx:          ctx,
		dockerClient: cli,
	}, nil
}

func (wc *WrapperClient) ContainerRemove(containerID string, options container.RemoveOptions) error {
	return wc.dockerClient.ContainerRemove(wc.ctx, containerID, options)
}

func (wc *WrapperClient) ImagePull(refStr string, options image.PullOptions) (io.ReadCloser, error) {
	return wc.dockerClient.ImagePull(wc.ctx, refStr, options)
}

func (wc *WrapperClient) ContainerCreate(config *container.Config) (container.CreateResponse, error) {
	return wc.dockerClient.ContainerCreate(wc.ctx, config, nil, nil, nil, "")
}

func (wc *WrapperClient) ContainerStart(containerID string, options container.StartOptions) error {
	return wc.dockerClient.ContainerStart(wc.ctx, containerID, options)
}

func (wc *WrapperClient) ContainerExecCreate(container string, options container.ExecOptions) (types.IDResponse, error) {
	return wc.dockerClient.ContainerExecCreate(wc.ctx, container, options)
}

func (wc *WrapperClient) ContainerExecAttach(execID string, config container.ExecAttachOptions) (types.HijackedResponse, error) {
	return wc.dockerClient.ContainerExecAttach(wc.ctx, execID, config)
}

func (wc *WrapperClient) ContainerExecStart(execID string, config container.ExecStartOptions) error {
	return wc.dockerClient.ContainerExecStart(wc.ctx, execID, config)
}

func (wc *WrapperClient) CopyToContainer(containerID, dstPath string, content io.Reader, options container.CopyToContainerOptions) error {
	return wc.dockerClient.CopyToContainer(wc.ctx, containerID, dstPath, content, options)
}

func (wc *WrapperClient) CopyFromContainer(containerID, srcPath string) (io.ReadCloser, container.PathStat, error) {
	return wc.dockerClient.CopyFromContainer(wc.ctx, containerID, srcPath)
}

func (wc *WrapperClient) Close() error {
	return wc.dockerClient.Close()
}
