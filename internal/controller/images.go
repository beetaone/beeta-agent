package controller

import (
	"encoding/json"
	"fmt"
	"net/http"

	log "github.com/sirupsen/logrus"

	// "gitlab.com/weeve/edge-server/edge-manager-service/internal/aws"
	// "gitlab.com/weeve/edge-server/edge-manager-service/internal/constants"
	// "gitlab.com/weeve/edge-server/edge-manager-service/internal/docker"
	"gitlab.com/weeve/edge-server/edge-pipeline-service/internal/docker"

	"github.com/gorilla/mux"
)

// ShowImages godoc
// @Summary Get all images
// @Description Get all images
// @Tags images
// @Accept  json
// @Produce  json
// @Success 200
// @Failure 400
// @Router /images [get]
// @Security ApiKeyAuth
// @param Authorization header string true "Token"
func GETimages(w http.ResponseWriter, r *http.Request) {
	// fmt.Println("Endpoint: returnAllImages")
	log.Info("GET /images")
	images := docker.ReadAllImages()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(images)
	log.Debug("Returning ", len(images), " images")
}

// ShowImages godoc
// @Summary Get all images
// @Description Get all images
// @Tags images
// @Accept  json
// @Produce  json
// @Success 200
// @Failure 400
// @Param id path int true "Image ID"
// @Router /images/{id} [get]
// @Security ApiKeyAuth
// @param Authorization header string true "Token"
func GETimagesID(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	log.Info("GET /images/{id}")
	vars := mux.Vars(r)
	image := docker.ReadImage(vars["id"])
	if image.ID == "" {
		msg := "Image " + vars["id"] + " not found"
		log.Warn(msg)
		http.Error(w, msg, http.StatusNotFound)
		return
	} else {
		log.Debug("Returning image ", vars["id"])
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("200 - Request processed successfully!"))
		json.NewEncoder(w).Encode(image)
	}
}

func POSTimage(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Endpoint: createImage")
	vars := mux.Vars(r)
	// image := dao.SaveData(vars)
	json.NewEncoder(w).Encode(vars)
}

func PUTimagesID(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Endpoint: createImage")
	vars := mux.Vars(r)
	key := vars["id"]
	// image := dao.EditData(key, vars)
	json.NewEncoder(w).Encode(key)
}

func DELETEimagesID(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Endpoint: createImage")
	vars := mux.Vars(r)
	key := vars["id"]
	// image := dao.DeleteData(key)
	json.NewEncoder(w).Encode(key)
}


/* OUT OF SCOPE - WE ONLY USE DOCKERHUB!
// GetAllEcrImages returns all images from ECR respository
// @Summary Get all images from Registry
// @Description Get all images
// @Tags images
// @Accept  json
// @Produce  json
// @Success 200
// @Failure 400
// @Param parentPath path string true "parentPath"
// @Param imageName path string true "imageName"
// @Router /ecrimages/{parentPath}/{imageName} [get]
// @Security ApiKeyAuth
// @param Authorization header string true "Token"
func GetAllEcrImages(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Endpoint: returnAllImages")
	vars := mux.Vars(r)
	fmt.Println("vars", vars)
	repoName := vars["parentPath"]
	imageName := vars["imageName"]

	if repoName == "" {
		panic("Repository Name is required")
	}

	if imageName == "" {
		panic("Image Name is required")
	}

	repoName = repoName + "/" + imageName

	//TODO: AWS WAS PUT OUT OF SCOPE!
	// images := aws.ReadAllEcrImages(repoName, constants.RoleArn)
	json.NewEncoder(w).Encode(images)
}

*/

// type Image struct {
// 	Id   string `json:"Id"`
// 	Name string `json:"Name"`
// 	tag  string `json:"tag"`
// }

// var Images []Image
