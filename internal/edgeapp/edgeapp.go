package edgeapp

import (
	"fmt"
	"strings"

	"errors"

	log "github.com/sirupsen/logrus"

	"github.com/beetaone/beeta-agent/internal/docker"
	"github.com/beetaone/beeta-agent/internal/manifest"
	"github.com/beetaone/beeta-agent/internal/model"
	traceutility "github.com/beetaone/beeta-agent/internal/utility/trace"
)

const (
	CMDDeploy   = "DEPLOY"
	CMDStop     = "STOP"
	CMDResume   = "RESUME"
	CMDUndeploy = "UNDEPLOY"
	CMDRemove   = "REMOVE"
)

func DeployEdgeApp(man manifest.Manifest) error {
	deploymentID := man.UniqueID.String() + " | "

	log.Info(deploymentID, "Deploying edge app ...")

	//******** STEP 1 - Check if a version of the edge app is already deployed *************//
	edgeAppRecord := manifest.GetKnownManifest(man.UniqueID)
	if edgeAppRecord != nil && edgeAppRecord.Status != model.EdgeAppUndeployed {
		if edgeAppRecord.Manifest.UpdatedAt.Before(man.UpdatedAt) {
			// remove the old version of the edge app, except for the images that are used by the new edge app
			var newImages []string
			for _, module := range man.Modules {
				newImages = append(newImages, module.ImageNameFull)
			}
			RemoveEdgeApp(man.UniqueID, newImages)
		} else {
			return errors.New("edge app " + man.UniqueID.String() + " already exist")
		}
	}

	manifest.AddKnownManifest(man)

	//******** STEP 2 - Pull all images *************//
	log.Info(deploymentID, "Iterating modules, pulling image into host if missing ...")

	for _, module := range man.Modules {
		// Check if image exist in local
		exists, err := docker.ImageExists(module.ImageNameFull)
		if err != nil {
			log.Error(deploymentID, "Deployment failed! CAUSE --> ", err)
			return traceutility.Wrap(err)
		}
		if exists { // Image already exists, continue
			log.Info(deploymentID, fmt.Sprintf("Image %v, already exists on host", module.ImageNameFull))
		} else { // Pull this image
			log.Info(deploymentID, fmt.Sprintf("Image %v, does not exist on host", module.ImageNameFull))
			log.Info(deploymentID, "Pulling ", module.ImageNameFull)
			err = docker.PullImage(module.AuthConfig, module.ImageNameFull)
			if err != nil {
				log.Error(deploymentID, "Unable to pull image/s, "+err.Error())
				setAndSendStatus(man.UniqueID, model.EdgeAppError)
				log.Info(deploymentID, "Initiating rollback ...")
				RemoveEdgeApp(man.UniqueID, nil)
				return errors.New("unable to pull image/s")
			}
		}
	}

	//******** STEP 3 - Create the network *************//
	log.Info(deploymentID, "Creating network ...")

	networkName, err := docker.CreateNetwork(man.ManifestName, man.Labels)
	if err != nil {
		log.Error("CreateNetwork failed! CAUSE --> ", err)
		setAndSendStatus(man.UniqueID, model.EdgeAppError)
		log.Info(deploymentID, "Initiating rollback ...")
		RemoveEdgeApp(man.UniqueID, nil)
		return traceutility.Wrap(err)
	}

	man.UpdateManifest(networkName)

	log.Info(deploymentID, "Created network >> ", networkName)

	//******** STEP 4 - Create, Start, attach all containers *************//
	log.Info(deploymentID, "Starting all containers ...")
	containerConfigs := man.Modules

	if len(containerConfigs) == 0 {
		log.Error(deploymentID, "No valid containers in Manifest")
		setAndSendStatus(man.UniqueID, model.EdgeAppError)
		log.Info(deploymentID, "Initiating rollback ...")
		RemoveEdgeApp(man.UniqueID, nil)
		return errors.New("no valid contianers in manifest")
	}

	// start containers in reverse order to prevent connectivity issues
	for i := len(containerConfigs) - 1; i >= 0; i-- {
		log.Info(deploymentID, "Creating ", containerConfigs[i].ContainerName, " from ", containerConfigs[i].ImageNameFull)
		containerID, err := docker.CreateAndStartContainer(containerConfigs[i])
		if err != nil {
			log.Error(deploymentID, "Failed to create and start container ", containerConfigs[i].ContainerName, " CAUSE --> ", err)
			log.Info(deploymentID, "Initiating rollback ...")
			RemoveEdgeApp(man.UniqueID, nil)
			setAndSendStatus(man.UniqueID, model.EdgeAppError)
			return traceutility.Wrap(err)
		}
		log.Info(deploymentID, "Successfully created container ", containerID)
		log.Info(deploymentID, "Started!")
	}

	setAndSendStatus(man.UniqueID, model.EdgeAppRunning)

	return nil
}

func StopEdgeApp(manifestUniqueID model.ManifestUniqueID) error {
	log.Infoln("Stopping edge app:", manifestUniqueID)

	status, err := manifest.GetEdgeAppStatus(manifestUniqueID)
	if err != nil {
		return traceutility.Wrap(err)
	}
	if status != model.EdgeAppRunning {
		return errors.New("can't stop edge application " + manifestUniqueID.String() + " with status " + status)
	}

	containers, err := docker.ReadEdgeAppContainers(manifestUniqueID)
	if err != nil {
		log.Error("Failed to read edge app containers! CAUSE --> ", err)
		return traceutility.Wrap(err)
	}

	if len(containers) == 0 {
		setAndSendStatus(manifestUniqueID, model.EdgeAppError)
		return errors.New("no edge app containers found")
	}

	setAndSendStatus(manifestUniqueID, model.EdgeAppExecuting)

	for _, container := range containers {
		if container.State == strings.ToLower(model.ModuleRunning) {
			log.Info("Stopping container:", strings.Join(container.Names[:], ","))
			err := docker.StopContainer(container.ID)
			if err != nil {
				log.Error("Could not stop a container! CAUSE --> ", err)
				setAndSendStatus(manifestUniqueID, model.EdgeAppError)

				return traceutility.Wrap(err)
			}

			log.Info(strings.Join(container.Names[:], ","), ": ", container.Status, " --> exited")
		} else {
			log.Debugln("Container", container.ID, "is", container.State, "and", container.Status)
		}
	}

	setAndSendStatus(manifestUniqueID, model.EdgeAppStopped)

	return nil
}

func ResumeEdgeApp(manifestUniqueID model.ManifestUniqueID) error {
	log.Infoln("Resuming edge app:", manifestUniqueID)

	status, err := manifest.GetEdgeAppStatus(manifestUniqueID)
	if err != nil {
		return traceutility.Wrap(err)
	}
	if status != model.EdgeAppStopped {
		return errors.New("can't resume edge application " + manifestUniqueID.String() + " with status " + status)
	}

	containers, err := docker.ReadEdgeAppContainers(manifestUniqueID)
	if err != nil {
		log.Error("Unable to resume edge app! CAUSE --> ", err)
		log.Error("Failed to read edge app containers.")
		setAndSendStatus(manifestUniqueID, model.EdgeAppError)
		return traceutility.Wrap(err)
	}

	if len(containers) == 0 {
		setAndSendStatus(manifestUniqueID, model.EdgeAppError)
		return errors.New("no edge app containers found")
	}

	setAndSendStatus(manifestUniqueID, model.EdgeAppExecuting)

	// start containers in reverse order to prevent connectivity issues
	for i := len(containers) - 1; i >= 0; i-- {
		if containers[i].State != strings.ToLower(model.ModuleRunning) {
			log.Info("Starting container:", strings.Join(containers[i].Names[:], ","))
			err := docker.StartContainer(containers[i].ID)
			if err != nil {
				log.Errorln("Could not start a container", err)
				setAndSendStatus(manifestUniqueID, model.EdgeAppError)
				return traceutility.Wrap(err)
			}

			log.Info(strings.Join(containers[i].Names[:], ","), ": ", containers[i].State, "--> running")
		} else {
			log.Debugln("Container", containers[i].ID, "is", containers[i].State, "and", containers[i].Status)
		}
	}

	setAndSendStatus(manifestUniqueID, model.EdgeAppRunning)

	return nil
}

func UndeployEdgeApp(manifestUniqueID model.ManifestUniqueID) error {
	log.Infoln("Undeploying edge app:", manifestUniqueID)

	undeploymentID := manifestUniqueID.String() + " | "

	// Check if edge app exist
	edgeAppRecord := manifest.GetKnownManifest(manifestUniqueID)
	if edgeAppRecord == nil {
		return errors.New("edge application " + manifestUniqueID.String() + " does not exist")
	}

	setAndSendStatus(manifestUniqueID, model.EdgeAppExecuting)

	//******** STEP 1 - Stop and Remove Containers *************//
	log.Info(undeploymentID, "Stopping and removing containers ...")
	dsContainers, err := docker.ReadEdgeAppContainers(manifestUniqueID)
	if err != nil {
		log.Error("Undeployment failed! CAUSE --> ", err)
		log.Error(undeploymentID, "Failed to read edge app containers.")
		setAndSendStatus(manifestUniqueID, model.EdgeAppError)
		return traceutility.Wrap(err)
	}

	var errorlist string
	for _, dsContainer := range dsContainers {
		err := docker.StopAndRemoveContainer(dsContainer.ID)
		if err != nil {
			log.Errorf("Undeployment failed! UndeploymentID --> %s, CAUSE --> %v", undeploymentID, err)
			setAndSendStatus(manifestUniqueID, model.EdgeAppError)
			errorlist = fmt.Sprintf("%v,%v", errorlist, err)
		}
	}

	//******** STEP 2 - Remove Network *************//
	log.Info(undeploymentID, "Pruning networks ...")

	err = docker.NetworkPrune(manifestUniqueID)
	if err != nil {
		log.Errorf("Undeployment failed! UndeploymentID --> %s, CAUSE --> %v", undeploymentID, err)
		setAndSendStatus(manifestUniqueID, model.EdgeAppError)
		errorlist = fmt.Sprintf("%v,%v", errorlist, err)
	}

	if errorlist != "" {
		return errors.New("Edge app could not be undeployed completely. Cause(s): " + errorlist)
	}

	setAndSendStatus(manifestUniqueID, model.EdgeAppUndeployed)

	return nil
}

func RemoveEdgeApp(manifestUniqueID model.ManifestUniqueID, keepImages []string) error {
	log.Infoln("Removing edge app:", manifestUniqueID)

	removalID := manifestUniqueID.String() + " | "

	//******** STEP 1 - Undeploy the edge app *************//
	err := UndeployEdgeApp(manifestUniqueID)
	if err != nil {
		return traceutility.Wrap(err)
	}

	//******** STEP 2 - Remove Images WITHOUT Containers *************//
	log.Info(removalID, "Removing images that are not needed anymore ...")
	usedImageNames, err := manifest.GetUsedImages(manifestUniqueID)
	if err != nil {
		log.Errorf("Edge app removal failed! RemovalID --> %s, CAUSE --> %v", removalID, err)
		setAndSendStatus(manifestUniqueID, model.EdgeAppError)
		return traceutility.Wrap(err)
	}

	// make sure that the images that should be kept are not removed
	var removeImageNames []string
	if len(keepImages) > 0 {
		removeImageNames = subtractArray(usedImageNames, keepImages)
	} else {
		removeImageNames = usedImageNames
	}

	// check if there are images that should be removed
	if len(removeImageNames) > 0 {
		removeImageIDs, err := docker.GetImagesByName(removeImageNames)
		if err != nil {
			log.Error("Unable to get images! CAUSE --> ", err)
			log.Error(removalID, "Failed to read the used images.")
			setAndSendStatus(manifestUniqueID, model.EdgeAppError)
			return traceutility.Wrap(err)
		}

		numContainersPerImage := make(map[string]int) // map { imageID: number_of_allocated_containers }
		for _, image := range removeImageIDs {
			numContainersPerImage[image.ID] = 0
		}
		containers, err := docker.ReadAllContainers()
		if err != nil {
			log.Error("Unable to read containers! CAUSE --> ", err)
			log.Error(removalID, "Failed to read all containers.")
			setAndSendStatus(manifestUniqueID, model.EdgeAppError)
			return traceutility.Wrap(err)
		}

		var errorlist string
		for imageID := range numContainersPerImage {
			for _, container := range containers {
				if container.ImageID == imageID {
					numContainersPerImage[imageID]++
				}
			}

			if numContainersPerImage[imageID] == 0 {
				log.Info(removalID, "Remove Image - ", imageID)
				err := docker.ImageRemove(imageID)
				if err != nil {
					log.Errorf("Edge app removal failed! RemovalID --> %s, CAUSE --> %v", removalID, err)
					setAndSendStatus(manifestUniqueID, model.EdgeAppError)
					errorlist = fmt.Sprintf("%v,%v", errorlist, err)
				}
			}
		}

		if errorlist != "" {
			return errors.New("Edge app could not be removed completely. Cause(s): " + errorlist)
		}
	}

	//******** STEP 3 - Remove Manifest *************//
	manifest.DeleteKnownManifest(manifestUniqueID)
	err = SendStatus()
	if err != nil {
		log.Errorf("Failed to delete known manifest! RemovalID --> %s, CAUSE --> %v", removalID, err)
		return traceutility.Wrap(err)
	}

	return nil
}

func RemoveAll() error {
	log.Info("Removing all edge apps")

	for uniqueID := range manifest.GetKnownManifests() {
		err := RemoveEdgeApp(uniqueID, nil)
		if err != nil {
			return traceutility.Wrap(err)
		}
	}

	return nil
}

func setAndSendStatus(manifestUniqueID model.ManifestUniqueID, status string) {
	log.Debug("Setting and sending edge app status...")

	err := manifest.SetStatus(manifestUniqueID, status)
	if err != nil {
		log.Error("SetStatus failed! CAUSE --> ", err)
		return
	}

	err = SendStatus()
	if err != nil {
		log.Error("SendStatus failed! CAUSE --> ", err)
	}
}

func subtractArray(minuend, subtrahend []string) (difference []string) {
	subtrahendMap := make(map[string]struct{}, len(subtrahend))
	for _, key := range subtrahend {
		subtrahendMap[key] = struct{}{}
	}
	for _, key := range minuend {
		if _, found := subtrahendMap[key]; !found {
			difference = append(difference, key)
		}
	}
	return
}
