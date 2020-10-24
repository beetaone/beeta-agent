package controller

import (
	"errors"
	"net/http"

	// "github.com/bitly/go-simplejson"

	log "github.com/sirupsen/logrus"
	"gitlab.com/weeve/edge-server/edge-pipeline-service/internal/docker"
	"gitlab.com/weeve/edge-server/edge-pipeline-service/internal/model"
	"gitlab.com/weeve/edge-server/edge-pipeline-service/internal/util"
)

//TODO: Add the code for instantiating a pipeline in the node:
// 1) Receive manifest
// 2) Iterate over each image
// 3) IF image not existing locally, PULL
//		ELSE: Continue
// 4) Run the container
func POST_pipelines(w http.ResponseWriter, r *http.Request) {
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

	// log.Info(w, "Person: %+v", manifest)

	log.Debug("Recieved manifest: ", manifest.Name)
	log.Debug("Number of modules in manifest: ", len(manifest.Modules))

	// Iterate over the modules inside the manifest
	// Pull all images as required
	log.Debug("Iterate modules, Docker Pull into host if missing")
	for i := range manifest.Modules {
		mod := manifest.Modules[i]
		log.Debug("\tModule: ", mod.ImageName)
		exists := docker.ImageExists(mod.ImageName)
		log.Debug("\tImage exists: ", exists)

		if exists == false {
			// Logic for pulling the image
			exists = docker.PullImage(mod.ImageName)
		}
	}



	log.Info("Pulled all images")

	/*
	// Iterate over the modules inside the manifest
	// Pull all images as required
	log.Debug("Iterate modules, check if containers exist")
	for i := range manifest.Modules {
		mod := manifest.Modules[i]
		log.Debug("\tModule: ", mod.ImageName)
		// image := docker.ReadImage(mod.ImageID)
		// log.Debug("Image: ", image)
		exists := docker.ImageExists(mod.ImageID)

		if exists == false {
			// TODO: Add error handling?
			msg := "Missing image on local machine:" + mod.ImageName
			// &model.ErrorResponse{}
			// &model.ErrorResponse{Err: "Mesage"}
			log.Error(msg)
			http.Error(w, msg, http.StatusInternalServerError)
			return
		}

		// TODO:
		// Logic for checking container exist
	}

	// Start all containers iteratively
	log.Debug("Iterate modules, start each container")
	for i := range manifest.Modules {
		mod := manifest.Modules[i]
		log.Debug("\tModule: ", mod.ImageName)

		// Starting container...

		// Error handling for case container start fails...

		// Log container ID, status

		// Wait for all....

	}

	// Wait for all ...

	// Finally, return 200
	// Return payload: pipeline started / list of container IDs
	*/
}
