package controller

import (
	"errors"
	"net/http"

	log "github.com/sirupsen/logrus"
	"gitlab.com/weeve/edge-server/edge-pipeline-service/internal/docker"
	"gitlab.com/weeve/edge-server/edge-pipeline-service/internal/model"
	"gitlab.com/weeve/edge-server/edge-pipeline-service/internal/util"
)

// PostPipelines function to,
// 1) Receive manifest
// 2) Iterate over each image
// 3) IF image not existing locally, PULL
//		ELSE: Continue
// 4) Run the container
func PostPipelines(w http.ResponseWriter, r *http.Request) {
	log.Info("POST /pipeline")

	// Decode the JSON manifest into Golang struct
	manifest := model.ManifestReq{}
	err := util.DecodeJSONBody(w, r, &manifest)
	if err != nil {
		var mr *util.MalformedRequest
		if errors.As(err, &mr) {
			log.Error(err.Error())
			http.Error(w, mr.Msg, mr.Status)
		} else {
			log.Error(err.Error())
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return
	}

	//******** STEP 1 - Pull all *************//
	// Pull all images as required
	log.Debug("Iterate modules, Docker Pull the image into host if missing")
	imagesPulled := PullImages(manifest)

	//******** STEP 2 - Check if pulled *************//
	// Check if all images pulled, else return
	if imagesPulled == false {
		msg := "Unable to pull all images"
		log.Error(msg)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}


	//******** STEP 3 - Check containers, stop and remove *************//
	// Create and start containers
	log.Debug("Iterate modules, check if containers exist, remove")
	for i := range manifest.Modules {
		mod := manifest.Modules[i]
		log.Debug("\tModule: ", mod.ImageName)

		// Build container name
		containerName := GetContainerName(manifest.ID, mod.Name)
		log.Info("\tContainer name:", containerName)

		// Check if container already exists
		containerExists := docker.ContainerExists(containerName)
		log.Info("\tContainer exists:", containerExists)

		// Create container if not exists
		if containerExists {
			log.Debug("\tStopAndRemoveContainer - ", containerName)
			// Stop and delete container
			err := docker.StopAndRemoveContainer(containerName)
			if err != nil {
				// msg := ""
				log.Error(err)
				http.Error(w, string(err.Error()), http.StatusInternalServerError)
			}
			log.Debug("\tContainer ", containerName, " removed")
		}

	}

	//******** STEP 4 - Start all containers *************//
	// Start all containers iteratively
	log.Debug("Iterate modules, start each container")
	for i := range manifest.Modules {
		mod := manifest.Modules[i]
		// log.Debug("\tContainer: ", mod.ImageName)

		// Build container name
		containerName := GetContainerName(manifest.ID, mod.Name)
		log.Info("\tCreateContainer - Container name:", containerName)

		// Create and start container
		docker.CreateContainer(containerName, mod.ImageName)
		log.Info("\tCreateContainer - successfully started:", containerName)
	}

	log.Info("Pipeline successfully instantiated from manifest ", manifest.ID)

	// Finally, return 200
	// Return payload: pipeline started / list of container IDs
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("200 - Request processed successfully!"))
	return
}

// // CreateStartContainers iterates modules, and creates and starts containers
// func CreateStartContainers(manifest model.ManifestReq) bool {
// 	for i := range manifest.Modules {
// 		mod := manifest.Modules[i]
// 		log.Debug("\tCreateStartContainers - Module: ", mod.ImageName)

// 		// Build container name
// 		containerName := GetContainerName(manifest.ID, mod.Name)
// 		log.Info("\tCreateStartContainers - Container name:", containerName)

// 		// Check if container already exists
// 		containerExists := docker.ContainerExists(containerName)
// 		log.Info("\tCreateStartContainers - Container exists:", containerExists)

// 		// Create container if not exists
// 		if containerExists {
// 			// Stop and delete container
// 			err := docker.StopAndRemoveContainer(containerName)
// 			if err != nil {
// 				return false
// 			}
// 		}

// 		// Create and start container
// 		docker.CreateContainer(containerName, mod.ImageName)
// 	}

// 	return true
// }

// PullImages iterates modules and pulls images
func PullImages(manifest model.ManifestReq) bool {
	for i := range manifest.Modules {
		mod := manifest.Modules[i]
		log.Debug("\tImageName: ", mod.ImageName)
		// Check if image exist in local
		exists := docker.ImageExists(mod.ImageName)
		log.Debug("\tImage exists: ", exists)

		if exists == false {
			// Pull image if not exist in local
			log.Debug("\t\tPulling ", mod.ImageName)
			exists = docker.PullImage(mod.ImageName)
			if exists == false {
				return false
			}
		}
	}

	return true
}

// GetContainerName build container name
func GetContainerName(pipelineID string, containerName string) string {
	return pipelineID + "_" + containerName
}