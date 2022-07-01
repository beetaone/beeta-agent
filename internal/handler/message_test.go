package handler_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/Jeffail/gabs/v2"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/weeveiot/weeve-agent/internal/dataservice"
	"github.com/weeveiot/weeve-agent/internal/docker"
	"github.com/weeveiot/weeve-agent/internal/handler"
	"github.com/weeveiot/weeve-agent/internal/manifest"
	"github.com/weeveiot/weeve-agent/internal/model"
)

func init() {
	docker.SetupDockerClient()
}

var manCmd struct {
	ManifestName  string  `json:"manifestName"`
	VersionNumber float64 `json:"versionNumber"`
	Command       string  `json:"command"`
}

var ctx = context.Background()
var dockerCli *client.Client

func TestProcessMessagePass(t *testing.T) {
	log.Info("TESTING DEPLOYMENT...")

	// Prepare test data
	manifestPath := "../../testdata/test_manifest.json"
	jsonBytes, err := ioutil.ReadFile(manifestPath)
	if err != nil {
		t.Error(err)
	}

	jsonParsed, err := gabs.ParseJSON(jsonBytes)
	if err != nil {
		t.Error(err)
	}

	man, err := ParseManifest(jsonParsed)
	if err != nil {
		t.Error(err)
	}

	manCmd.ManifestName = man.ManifestUniqueID.ManifestName
	manCmd.VersionNumber = man.VersionNumber

	dockerCli, err = client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		t.Error(err)
	}

	err = DeployEdgeApplication(jsonBytes, man)
	if err != nil {
		t.Error(err)
	}

	err = StopEdgeApplication(man)
	if err != nil {
		t.Error(err)
	}

	err = StartEdgeApplication(man)
	if err != nil {
		t.Error(err)
	}

	err = UndeployEdgeApplication(man)
	if err != nil {
		t.Error(err)
	}
}

func DeployEdgeApplication(jsonBytes []byte, man manifest.Manifest) error {
	// Process deploy edge application
	err := handler.ProcessMessage(jsonBytes)
	if err != nil {
		return fmt.Errorf("ProcessMessage returned %v status", err)
	}

	// Verify deployment
	exist, err := CheckNetworkExist(man.ManifestUniqueID)
	if err != nil {
		return err
	}

	if exist {
		_, err := CheckContainersExistWithStatus(man.ManifestUniqueID, len(man.Modules), manifest.Running)
		if err != nil {
			return err
		}
	} else {
		return errors.New("Network not created")
	}

	return nil
}

func StopEdgeApplication(man manifest.Manifest) error {
	// Process stop edge application
	manCmd.Command = dataservice.CMDStopService
	jsonB, err := json.Marshal(manCmd)
	if err != nil {
		panic(err)
	}

	err = handler.ProcessMessage(jsonB)
	if err != nil {
		return fmt.Errorf("ProcessMessage returned %v status", err)
	}

	_, err = CheckContainersExistWithStatus(man.ManifestUniqueID, len(man.Modules), manifest.Paused)
	if err != nil {
		return err
	}

	return nil
}

func StartEdgeApplication(man manifest.Manifest) error {
	// Process start edge application
	manCmd.Command = dataservice.CMDStartService
	jsonB, err := json.Marshal(manCmd)
	if err != nil {
		return err
	}

	err = handler.ProcessMessage(jsonB)
	if err != nil {
		return fmt.Errorf("ProcessMessage returned %v status", err)
	}

	_, err = CheckContainersExistWithStatus(man.ManifestUniqueID, len(man.Modules), manifest.Running)
	if err != nil {
		return err
	}

	return nil
}

func UndeployEdgeApplication(man manifest.Manifest) error {
	// Process undeploy edge application
	manCmd.Command = dataservice.CMDUndeploy
	jsonB, err := json.Marshal(manCmd)
	if err != nil {
		panic(err)
	}

	err = handler.ProcessMessage(jsonB)
	if err != nil {
		return fmt.Errorf("ProcessMessage returned %v status", err)
	}

	exist, err := CheckNetworkExist(man.ManifestUniqueID)
	if err != nil {
		return err
	}

	if !exist {
		dsContainers, _ := docker.ReadDataServiceContainers(man.ManifestUniqueID)
		if len(dsContainers) > 0 {
			return errors.New("Edge application undeployment failed")
		}
	}

	exist, err = CheckImagesExist(man)
	if err != nil {
		return err
	}

	if !exist {
		return errors.New("Undeploy should not delete images")
	}

	return nil
}

func TestReadDeployManifestLocalPass(t *testing.T) {
	msg, err := handler.GetStatusMessage()
	if err != nil {
		t.Error("Expected status message, but got error! CAUSE --> ", err)
	}

	assert.Nil(t, msg)
	assert.NotEqual(t, nil, msg)
}

func ParseManifest(jsonParsed *gabs.Container) (manifest.Manifest, error) {
	manifestName := jsonParsed.Search("manifestName").Data().(string)
	versionNumber := jsonParsed.Search("versionNumber").Data().(float64)
	command := jsonParsed.Search("command").Data().(string)

	var containerConfigs []manifest.ContainerConfig

	modules := jsonParsed.Search("modules").Children()
	for _, module := range modules {
		var containerConfig manifest.ContainerConfig

		containerConfig.ImageName = module.Search("image").Search("name").Data().(string)
		containerConfig.ImageTag = module.Search("image").Search("tag").Data().(string)

		imageName := containerConfig.ImageName
		if containerConfig.ImageTag != "" {
			imageName = imageName + ":" + containerConfig.ImageTag
		}

		containerConfig.Registry = manifest.RegistryDetails{ImageName: imageName}
		containerConfigs = append(containerConfigs, containerConfig)
	}

	manifest := manifest.Manifest{
		ManifestUniqueID: model.ManifestUniqueID{ManifestName: manifestName, VersionNumber: fmt.Sprint(versionNumber)},
		VersionNumber:    versionNumber,
		Modules:          containerConfigs,
		Command:          command,
	}

	return manifest, nil
}

func CheckNetworkExist(manID model.ManifestUniqueID) (bool, error) {
	filter := filters.NewArgs()
	filter.Add("label", "manifestName="+manID.ManifestName)
	filter.Add("label", "versionNumber="+manID.VersionNumber)
	options := types.NetworkListOptions{Filters: filter}
	networks, err := dockerCli.NetworkList(context.Background(), options)
	if err != nil {
		return false, err
	}

	return len(networks) > 0, nil
}

func GetEdgeApplicationContainers(manifestUniqueID model.ManifestUniqueID) ([]types.Container, error) {
	filter := filters.NewArgs()
	filter.Add("label", "manifestName="+manifestUniqueID.ManifestName)
	filter.Add("label", "versionNumber="+manifestUniqueID.VersionNumber)
	options := types.ContainerListOptions{All: true, Filters: filter}
	containers, err := dockerCli.ContainerList(context.Background(), options)
	if err != nil {
		return nil, err
	}

	return containers, nil
}

func CheckContainersExistWithStatus(manID model.ManifestUniqueID, containerCount int, status string) (bool, error) {
	dsContainers, _ := GetEdgeApplicationContainers(manID)
	if containerCount > len(dsContainers) {
		return false, fmt.Errorf("Expected number of containers %v, number of available containers %v", containerCount, len(dsContainers))
	}
	for _, dsContainer := range dsContainers {
		if dsContainer.State != status {
			return false, fmt.Errorf("Container expected status %s, but current status %s", status, dsContainer.State)
		}
	}

	return true, nil
}

func CheckImagesExist(man manifest.Manifest) (bool, error) {
	for _, module := range man.Modules {
		imgDetails := module.Registry
		_, _, err := dockerCli.ImageInspectWithRaw(ctx, imgDetails.ImageName)
		if err != nil {
			if client.IsErrNotFound(err) {
				return false, nil
			} else {
				return false, err
			}
		}
	}

	return true, nil
}
