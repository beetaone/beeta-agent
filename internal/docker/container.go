// data_access
package docker

import (
	"fmt"

	"bytes"
	"context"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	log "github.com/sirupsen/logrus"
	"gitlab.com/weeve/edge-server/edge-pipeline-service/internal/model"
	"gitlab.com/weeve/edge-server/edge-pipeline-service/internal/util"

	"github.com/davecgh/go-spew/spew"
)

func ReadAllContainers() []types.Container {
	log.Debug("Docker_container -> ReadAllContainers")
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Error(err)
		return nil
	}
	options := types.ContainerListOptions{All: true}
	containers, err := cli.ContainerList(context.Background(), options)
	if err != nil {
		log.Error(err)
	}
	log.Debug("Docker_container -> ReadAllContainers response", containers)

	return containers
}

func GetContainerLog(container string) string {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Error("Env Error ", err)
	}

	options := types.ContainerLogsOptions{
		ShowStderr: true,
		ShowStdout: true,
		Timestamps: false,
		Follow:     true,
		Tail:       "40",
	}

	logs, err := cli.ContainerLogs(context.Background(), container, options)
	if err != nil {
		log.Error("Log fetch Error ", err)
	}
	log.Debug("Logs ", logs)
	buf := new(bytes.Buffer)
	buf.ReadFrom(logs)
	logStr := buf.String()

	log.Debug("Log string ", logStr)

	return logStr
}

func StopContainers() bool {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Error(err)
	}

	options := types.ContainerListOptions{All: true}
	containers, err := cli.ContainerList(ctx, options)
	if err != nil {
		log.Error(err)
	}

	for _, container := range containers {
		fmt.Print("Stopping container ", container.ID[:10], "... ", container.State)
		// if "State": "running"

		if container.State == "running" {
			if err := cli.ContainerStop(ctx, container.ID, nil); err != nil {
				log.Error(err)
			}
		}
		fmt.Println("Success")
	}
	return true
}

func StopContainer(containerId string) bool {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Error(err)
	}

	if err := cli.ContainerStop(ctx, containerId, nil); err != nil {
		log.Error(err)
	}

	return true
}

func CreateContainerOptsArgs(startCmd model.StartCommand, networkName string) bool {

	// fmt.Println(startCmd)
	spew.Dump(startCmd)

	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Error(err)
		// log.Error(err)
		return false
	}

	containerConfig := &container.Config{
		Image:        startCmd.ImageName + ":" + startCmd.ImageTag,
		AttachStdin:  false,
		AttachStdout: false,
		AttachStderr: false,
		Cmd:          startCmd.EntryPointArgs,
		Tty:          false,
		ExposedPorts: startCmd.ExposedPorts,
	}

	hostConfig := &container.HostConfig{
		PortBindings: startCmd.PortBinding,
		NetworkMode: startCmd.NetworkMode,
	}

	resp, err := cli.ContainerCreate(ctx,
		containerConfig,
		hostConfig,
		&startCmd.NetworkConfig,
		// &network.NetworkingConfig{},
		nil,
		startCmd.ContainerName)
	// fmt.Println(resp)
	if err != nil {
		log.Error(err)
		// return "CreateFailed"
		return false
	}
	log.Debug("Created container " + startCmd.ContainerName)

	containerStarted := StartContainer(resp.ID)

	if !containerStarted {
		log.Debug("Did not start container")
		return false
	}

	// statusCh, errCh := cli.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
	// select {
	// case err := <-errCh:
	// 	if err != nil {
	// 		panic(err)
	// 	}
	// case <-statusCh:
	// }

	// out, err := cli.ContainerLogs(ctx, resp.ID, types.ContainerLogsOptions{ShowStdout: true})
	// if err != nil {
	// 	panic(err)
	// }

	// stdcopy.StdCopy(os.Stdout, os.Stderr, out)

	// cli.NetworkConnect(ctx, "TEST", resp.ID, config *network.EndpointSettings)
	var netConfig network.EndpointSettings
	err = cli.NetworkConnect(ctx, networkName, resp.ID, &netConfig)
	if err != nil {
		panic(err)
	}
	log.Debug("Connected ", resp.ID, "to network", networkName)

	return true
}

// StopAndRemoveContainer Stop and remove a container
func StopAndRemoveContainer(containerName string) error {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Error(err)
	}

	if err := cli.ContainerStop(ctx, containerName, nil); err != nil {
		log.Printf("Unable to stop container %s: %s", containerName, err)
	}

	removeOptions := types.ContainerRemoveOptions{
		RemoveVolumes: true,
		Force:         true,
	}

	if err := cli.ContainerRemove(ctx, containerName, removeOptions); err != nil {
		log.Printf("Unable to remove container: %s", err)
		return err
	}

	return nil
}

// ContainerExists returns status of container existance as true or false
func ContainerExists(containerName string) bool {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Error(err)
	}
	options := types.ContainerListOptions{All: true}
	containers, err := cli.ContainerList(context.Background(), options)
	if err != nil {
		log.Error(err)
	}

	for _, container := range containers {
		// fmt.Printf("%s %s\n", container.ID[:10], container.Image)
		findContainer := util.StringArrayContains(container.Names, "/"+containerName)
		if findContainer {
			return true
		}
	}

	return false
}
