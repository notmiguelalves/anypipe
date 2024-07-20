package dockerutils

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
)

type DockerUtils struct {
	ctx               context.Context
	logger            *slog.Logger
	dockerClient      *client.Client
	spawnedContainers []*Container
}

func New(ctx context.Context, logger *slog.Logger) (*DockerUtils, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		logger.Error(fmt.Sprintf("failed to initialize docker client: %s", err.Error()))
		return nil, err
	}

	return &DockerUtils{
		ctx:          ctx,
		dockerClient: cli,
		logger:       logger,
	}, nil
}

func (du *DockerUtils) Close() error {
	du.logger.Info("cleaning up spawned containers")

	for _, c := range du.spawnedContainers {
		du.logger.Info(fmt.Sprintf("going to cleanup %s", c.id))

		err := du.dockerClient.ContainerRemove(du.ctx, c.id, container.RemoveOptions{
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

func (du *DockerUtils) pullImage(img string) error {
	rc, err := du.dockerClient.ImagePull(du.ctx, img, image.PullOptions{})
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
		du.logger.Info(strings.ReplaceAll(l, "\"", "'"))
	}

	return nil
}

func (du *DockerUtils) CreateContainer(image string) (*Container, error) {
	err := du.pullImage(image)
	if err != nil {
		du.logger.Error(fmt.Sprintf("failed to pull image %s : %s", image, err.Error()))
		return nil, err
	}

	resp, err := du.dockerClient.ContainerCreate(du.ctx, &container.Config{
		Image: image,
		Cmd:   []string{"sleep", "infinity"},
		Tty:   false,
	}, nil, nil, nil, "")
	if err != nil {
		du.logger.Error(fmt.Sprintf("failed to create container from '%s' : %s", image, err.Error()))
		return nil, err
	}

	c := Container{
		id:  resp.ID,
		env: map[string]string{},
	}
	du.spawnedContainers = append(du.spawnedContainers, &c)

	du.logger.Info(fmt.Sprintf("going to start container %s created from image %s", c.id, image))
	if err := du.dockerClient.ContainerStart(du.ctx, c.id, container.StartOptions{}); err != nil {
		du.logger.Error(fmt.Sprintf("failed to start container '%s' : %s", c.id, err.Error()))
		return &c, err
	}

	du.logger.Info(fmt.Sprintf("started container %s", c.id))
	return &c, nil
}

func (du *DockerUtils) Exec(c *Container, cmd string) error {
	du.logger.Info(fmt.Sprintf("going to execute %s on container %s", cmd, c.id))

	shcmd := []string{"sh", "-c", cmd} // TODO @Miguel : this smells kinda bad

	resp, err := du.dockerClient.ContainerExecCreate(du.ctx, c.id, container.ExecOptions{Cmd: shcmd, Env: c.Env(), Detach: false, AttachStderr: true, AttachStdout: true, WorkingDir: "/home"})
	if err != nil {
		du.logger.Error("failed to create exec operation on container %s : %s", c.id, err.Error())
		return err
	}

	attachResp, err := du.dockerClient.ContainerExecAttach(du.ctx, resp.ID, container.ExecAttachOptions{})
	if err != nil {
		du.logger.Error("failed to attach exec operation to container %s : %s", c.id, err.Error())
		return err
	}
	defer attachResp.Close()

	err = du.dockerClient.ContainerExecStart(du.ctx, resp.ID, container.ExecStartOptions{})
	if err != nil {
		du.logger.Error("failed to start exec operation on container %s : %s", c.id, err.Error())
		return err
	}

	_, err = stdcopy.StdCopy(os.Stdout, os.Stderr, attachResp.Reader)
	if err != nil {
		du.logger.Error(fmt.Sprintf("failed to fetch container stdout/stderr from container %s : %s", c.id, err.Error()))
		return err
	}

	// TODO @Miguel : instead of writing to stdout and stderr, function should receive two params where to write
	// TODO @Miguel : fetch and return exit code as well

	return nil
}

func (du *DockerUtils) CopyTo(c *Container, srcPath, dstPath string) error {
	var buf bytes.Buffer
	// zr := gzip.NewWriter(&buf)
	// defer zr.Close()
	tw := tar.NewWriter(&buf)
	defer tw.Close()

	err := filepath.Walk(srcPath, func(file string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		header, err := tar.FileInfoHeader(fi, file)
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(srcPath, file)
		if err != nil {
			return err
		}
		header.Name = filepath.ToSlash(relPath)

		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		if !fi.IsDir() {
			data, err := os.Open(file)
			if err != nil {
				return err
			}
			defer data.Close()

			_, err = io.Copy(tw, data)
			if err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		return err
	}

	err = du.dockerClient.CopyToContainer(du.ctx, c.id, dstPath, &buf, container.CopyToContainerOptions{})
	if err != nil {
		return err
	}

	return nil
}

func (du *DockerUtils) CopyFrom(c *Container, srcPath, dstPath string) error {
	rc, _, err := du.dockerClient.CopyFromContainer(du.ctx, c.id, srcPath)
	if err != nil {
		du.logger.Error(fmt.Sprintf("failed to copy %s from container %s : %s", srcPath, c.id, err.Error()))
		return err
	}

	tr := tar.NewReader(rc)

	for {
		header, err := tr.Next()

		switch {
		case err == io.EOF:
			return nil

		case err != nil:
			return err

		case header == nil:
			continue
		}

		target := filepath.Join(dstPath, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			if _, err := os.Stat(target); err != nil {
				if err := os.MkdirAll(target, 0755); err != nil {
					return err
				}
			}

		case tar.TypeReg:
			f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return err
			}

			if _, err := io.Copy(f, tr); err != nil {
				return err
			}

			// defering would cause each file close to wait until all operations have completed.
			_ = f.Close()
		}
	}
}
