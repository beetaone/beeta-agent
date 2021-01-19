package controller

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"

	"github.com/docker/docker/client"
	log "github.com/sirupsen/logrus"
	"gitlab.com/weeve/edge-server/edge-pipeline-service/internal/docker"
	"gitlab.com/weeve/edge-server/edge-pipeline-service/internal/model"

	"github.com/docker/docker/api/types"
)

func POSTpipelines(w http.ResponseWriter, r *http.Request) {
	log.Info("POST /pipeline")

	//Get the manifest as a []byte
	manifestBodyBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Error(err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	man, err := model.ParseJSONManifest(manifestBodyBytes)
	if err != nil {
		log.Error(err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = model.ValidateManifest(man)
	if err != nil {
		log.Error(err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Check if process is failed and needs to return
	failed := false

	// res := util.PrintManifestDetails(body)
	// util.PrettyPrintJson(manifestBodyBytes)
	// man.PrintManifest()
	// man.SpewManifest()
	// return

	//******** STEP 1 - Pull all *************//
	// Pull all images as required
	log.Debug("Iterating modules, pulling image into host if missing")

	for i, imgName := range man.ImageNamesList() {
		// Check if image exist in local
		exists := docker.ImageExists(imgName)
		if exists { // Image already exists, continue
			log.Info(fmt.Sprintf("\tImage %v %v, already exists on host", i, imgName))
		} else { // Pull this image
			log.Info(fmt.Sprintf("\tImage %v %v, does not exist on host", i, imgName))
			log.Info("\t\tPulling ", imgName)
			exists = docker.PullImage(imgName)
			if exists == false {
				failed = true
				msg := "404 - Unable to pull image " + imgName
				log.Error(msg)
				http.Error(w, msg, http.StatusNotFound)
				break
			}
		}
	}

	if failed {
		return
	}

	//******** STEP 2 - Check containers, stop and remove *************//
	log.Debug("Checking containers, stopping and removing")

	for _, containerName := range man.ContainerNamesList() {

		containerExists := docker.ContainerExists(containerName)
		log.Info("\tContainer exists:", containerExists)

		// Stop + remove container if exists, start fresh
		if containerExists {
			log.Debug("\tStopAndRemoveContainer - ", containerName)
			// Stop and delete container
			err := docker.StopAndRemoveContainer(containerName)
			if err != nil {
				failed = true
				log.Error(err)
				http.Error(w, string(err.Error()), http.StatusInternalServerError)
			}
			log.Debug("\tContainer ", containerName, " removed")
		}
	}

	if failed {
		return
	}

	//******** STEP 3 - Create the network *************//
	// var networkName = "my-net5"
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Error(err)
	}

	log.Debug("Pruning networks")
	filter := filters.NewArgs()

	pruneReport, err := cli.NetworksPrune(ctx, filter)
	log.Debug("Pruned:", pruneReport)
	// var report types.NetworksPruneReport
	log.Debug("Create the network")
	var networkCreateOptions types.NetworkCreate
	networkCreateOptions.CheckDuplicate = true
	networkCreateOptions.Attachable = true
	// var networkCreateOptions = &NetworkCreate

	// _ = ctx
	// _ = cli
	// fmt.Println(networkCreateOptions)
	networkName := man.GetNetworkName()
	resp, err := cli.NetworkCreate(ctx, networkName, networkCreateOptions)
	if err != nil {
		log.Error(err)
		log.Error("Error trying to create network " + networkName)
		panic(err)

	}
	log.Debug("Created network named ", networkName)

	_ = resp
	// log.Debug(resp.ID, resp.Warning)

	//******** STEP 4 - Create, Start, attach all containers *************//
	log.Debug("Start all containers")

	for _, startCommand := range man.GetContainerStart() {
		log.Info("Creating ", startCommand.ContainerName, " from ", startCommand.ImageName, ":", startCommand.ImageTag)
		imageAndTag := startCommand.ImageName + ":" + startCommand.ImageTag
		containerCreateResponse, err := docker.StartCreateContainer(imageAndTag, startCommand.ContainerName, startCommand.EntryPointArgs)
		log.Info("\tSuccessfully created with args: ", startCommand.EntryPointArgs)
		if err != nil {
			failed = true
			log.Info("Started")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("500 - Failed to create container!"))
		}

		// Attach to network
		var netConfig network.EndpointSettings
		err = cli.NetworkConnect(ctx, startCommand.NetworkName, containerCreateResponse.ID, &netConfig)
		if err != nil {
			panic(err)
		}
		log.Debug("\tConnected to network", startCommand.NetworkName)
	}

	if failed {
		return
	}

	// Finally, return 200
	// Return payload: pipeline started / list of container IDs
	log.Info("Started")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("200 - Request processed successfully!"))
	return
}
