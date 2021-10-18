package docker

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
)

// Test utilities for working with docker

type ExposedPort struct {
	HostPort      int
	ContainerPort int
}

func LaunchContainer(image string, exposePorts []ExposedPort, cmd, env []string) (containerID string) {
	ctx := context.Background()

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}

	reader, err := cli.ImagePull(ctx, image, types.ImagePullOptions{})
	if err != nil {
		panic(err)
	}

	_, _ = io.Copy(os.Stdout, reader)

	cfg := &container.Config{
		Image:        image,
		ExposedPorts: nat.PortSet{},
		Env:          env,
	}
	if cmd != nil {
		cfg.Cmd = cmd
	}

	hostCfg := &container.HostConfig{
		PortBindings: nat.PortMap{},
	}

	for _, port := range exposePorts {
		p, _ := nat.NewPort("tcp", fmt.Sprintf("%d", port.ContainerPort))
		cfg.ExposedPorts[p] = struct{}{}

		hostCfg.PortBindings[p] = append(hostCfg.PortBindings[p], nat.PortBinding{
			// HostIP: "127.0.0.1",
			HostPort: fmt.Sprintf("%d", port.HostPort),
		})
	}

	resp, err := cli.ContainerCreate(ctx, cfg, hostCfg, nil, nil, "")
	if err != nil {
		panic(err)
	}

	if err := cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		panic("container starting: " + err.Error())
	}

	// make sure they're up
	deadline := time.Now().Add(60 * time.Second)

	for _, port := range exposePorts {
	retry:
		con, err := net.DialTCP("tcp", nil, &net.TCPAddr{
			IP:   net.IPv4(127, 0, 0, 1),
			Port: port.HostPort,
		})

		if err != nil {
			if time.Now().After(deadline) {
				panic("timed out waiting for container to start: " + err.Error())
			}

			time.Sleep(10 * time.Millisecond)

			goto retry
		}

		con.Close()
	}

	return resp.ID
}

func KillContainer(containerID string) {
	ctx := context.Background()

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}

	timeout := 10 * time.Second

	err = cli.ContainerStop(ctx, containerID, &timeout)
	if err != nil {
		fmt.Println("error stopping container: " + err.Error())
	}

	err = cli.ContainerRemove(ctx, containerID, types.ContainerRemoveOptions{Force: true})
	if err != nil {
		fmt.Println("error removing container: " + err.Error())
	}
}
