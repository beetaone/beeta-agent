package controller

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	// "github.com/bitly/go-simplejson"
	log "github.com/sirupsen/logrus"
	"gitlab.com/weeve/edge-server/edge-pipeline-service/internal/docker"
	"gitlab.com/weeve/edge-server/edge-pipeline-service/internal/model"

	"github.com/golang/gddo/httputil/header"
)

//TODO: Add the code for instantiating a pipeline in the node:
// 1) Receive manifest
// 2) Iterate over each image
// 3) IF image not existing locally, PULL
//		ELSE: Continue
// 4) Run the container
func BuildPipeline(w http.ResponseWriter, r *http.Request) {
	log.Info("POST /pipeline")

	// Enforce content type exists
	if r.Header.Get("Content-Type") == "" {
		msg := "Content-Type header is not application/json"
		log.Error(msg)
		http.Error(w, msg, http.StatusUnsupportedMediaType)
		return
	}

	// Enforce content type is application/json
	// Note that we are using the gddo/httputil/header
	// package to parse and extract the value here, so the check works
	// even if the client includes additional charset or boundary
	// information in the header.
	value, _ := header.ParseValueAndParams(r.Header, "Content-Type")
	if value != "application/json" {
		msg := "Content-Type header is not application/json"
		log.Error(msg)
		http.Error(w, msg, http.StatusUnsupportedMediaType)
		return
	}

	// Now handle the payload, start by converting to []bytes
	log.Debug("Raw POST body:", r.Body)
	bodyBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Error(err)
		msg := "Error in decoding JSON payload, check for valid JSON"
		http.Error(w, msg, http.StatusBadRequest)
		return
	}
	log.Debug("POST body as string:", string(bodyBytes))

	// Decode the JSON manifest into Golang struct
	manifest := model.ManifestReq{}

	err = json.NewDecoder(r.Body).Decode(&manifest)
	// err = json.NewDecoder(bodyBytes).Decode(manifest)
	log.Error(err)
	// if err != nil {
	// 	msg := "Manifest does not match schema"
	// 	log.Error(msg)
	// 	http.Error(w, msg, http.StatusBadRequest)
	// 	return
	// }

	log.Debug("Recieved manifest: ", manifest.Name)
	log.Debug("Number of modules: ", len(manifest.Modules))

	// Iterate over the modules inside the manifest
	// Pull all images as required
	log.Debug("Iterate modules, pull into host if missing")
	for i := range manifest.Modules {
		mod := manifest.Modules[i]
		log.Debug("\tModule: ", mod.ImageName)
		exists := docker.ImageExists(mod.ImageID)
		log.Debug("\tImage exists: ", exists)

		if exists == false {
			// TODO:
			// Logic for pulling the image
			exists = docker.PullImage(mod.ImageName)
		}

		if exists == true {
			docker.CreateContainer(mod.Name, mod.ImageName)

		}
	}

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
}
