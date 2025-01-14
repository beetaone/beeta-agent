package manifest

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/go-connections/nat"
	"github.com/go-playground/validator/v10"
	"github.com/go-playground/validator/v10/non-standard/validators"
	log "github.com/sirupsen/logrus"

	"github.com/beetaone/beeta-agent/internal/config"
	"github.com/beetaone/beeta-agent/internal/model"
	"github.com/beetaone/beeta-agent/internal/secret"
	traceutility "github.com/beetaone/beeta-agent/internal/utility/trace"
)

type Manifest struct {
	UniqueID     model.ManifestUniqueID // this is the only field that the manifest should be identified by, the rest of identifiers are just metadata
	ID           string
	ManifestName string
	UpdatedAt    time.Time
	Modules      []ContainerConfig
	Labels       map[string]string
	Connections  connectionsInt
}

// This struct holds information for starting a container
type ContainerConfig struct {
	ContainerName string
	ImageNameFull string
	EnvArgs       []string
	NetworkName   string
	ExposedPorts  nat.PortSet // This must be set for the container create
	PortBinding   nat.PortMap // This must be set for the containerStart
	NetworkConfig network.NetworkingConfig
	MountConfigs  []mount.Mount
	Labels        map[string]string
	AuthConfig    types.AuthConfig
	Resources     container.Resources
}

type connectionsInt map[int][]int
type connectionsString map[string][]string

var validate *validator.Validate

func init() {
	validate = validator.New()
	validate.RegisterValidation("notblank", validators.NotBlank)
}

func Parse(payload []byte) (Manifest, error) {
	var man manifestMsg
	err := json.Unmarshal(payload, &man)
	if err != nil {
		return Manifest{}, traceutility.Wrap(err)
	}

	log.Debug("Parsed manifest json >> ", man)

	err = validate.Struct(man)
	if err != nil {
		return Manifest{}, traceutility.Wrap(err)
	}

	updatedAt, err := time.Parse(time.RFC3339, man.UpdatedAt)
	if err != nil {
		return Manifest{}, traceutility.Wrap(err)
	}

	uniqueID := model.ManifestUniqueID{ID: man.ID}

	labels := map[string]string{
		"manifestUniqueID": uniqueID.String(),
	}

	var containerConfigs []ContainerConfig

	for _, module := range man.Modules {
		err = validate.Struct(module)
		if err != nil {
			return Manifest{}, traceutility.Wrap(err)
		}

		var containerConfig ContainerConfig

		containerConfig.Labels = labels

		if module.Image.Tag == "" {
			containerConfig.ImageNameFull = module.Image.Name
		} else {
			containerConfig.ImageNameFull = module.Image.Name + ":" + module.Image.Tag
		}

		containerConfig.AuthConfig = types.AuthConfig{
			ServerAddress: module.Image.Registry.Url,
			Username:      module.Image.Registry.UserName,
			Password:      module.Image.Registry.Password,
		}

		envArgs, err := parseArguments(module.Envs)
		if err != nil {
			return Manifest{}, traceutility.Wrap(err)
		}

		if man.DebugMode {
			envArgs = append(envArgs, fmt.Sprintf("%v=%v", "LOG_LEVEL", "DEBUG"))
		} else {
			envArgs = append(envArgs, fmt.Sprintf("%v=%v", "LOG_LEVEL", "INFO"))
		}

		envArgs = append(envArgs, fmt.Sprintf("%v=%v", "MANIFEST_ID", man.ID))
		envArgs = append(envArgs, fmt.Sprintf("%v=%v", "MODULE_NAME", containerConfig.ImageNameFull))
		envArgs = append(envArgs, fmt.Sprintf("%v=%v", "INGRESS_PORT", 80))
		envArgs = append(envArgs, fmt.Sprintf("%v=%v", "INGRESS_PATH", "/"))
		envArgs = append(envArgs, fmt.Sprintf("%v=%v", "MODULE_TYPE", module.Type))
		envArgs = append(envArgs, fmt.Sprintf("%v=%v", "NODE_ID", config.Params.NodeId))
		envArgs = append(envArgs, fmt.Sprintf("%v=%v", "NODE_NAME", config.Params.NodeName))

		containerConfig.EnvArgs = envArgs
		containerConfig.MountConfigs, err = parseMounts(module.Mounts)
		if err != nil {
			return Manifest{}, traceutility.Wrap(err)
		}

		devices, err := parseDevices(module.Devices)
		if err != nil {
			return Manifest{}, traceutility.Wrap(err)
		}
		containerConfig.Resources = container.Resources{Devices: devices}

		containerConfig.ExposedPorts, containerConfig.PortBinding = parsePorts(module.Ports)
		containerConfigs = append(containerConfigs, containerConfig)
	}

	connections, err := parseConnections(man.Connections)
	if err != nil {
		return Manifest{}, traceutility.Wrap(err)
	}

	manifest := Manifest{
		UniqueID:     uniqueID,
		ID:           man.ID,
		ManifestName: man.ManifestName,
		UpdatedAt:    updatedAt,
		Modules:      containerConfigs,
		Labels:       labels,
		Connections:  connections,
	}

	return manifest, nil
}

func GetCommand(payload []byte) (string, error) {
	var msg commandMsg
	err := json.Unmarshal(payload, &msg)
	if err != nil {
		return "", traceutility.Wrap(err)
	}

	err = validate.Struct(msg)
	if err != nil {
		return "", traceutility.Wrap(err)
	}

	return msg.Command, nil
}

func GetEdgeAppUniqueID(payload []byte) (model.ManifestUniqueID, error) {
	var uniqueID uniqueIDmsg
	err := json.Unmarshal(payload, &uniqueID)
	if err != nil {
		return model.ManifestUniqueID{}, traceutility.Wrap(err)
	}

	err = validate.Struct(uniqueID)
	if err != nil {
		return model.ManifestUniqueID{}, traceutility.Wrap(err)
	}

	return model.ManifestUniqueID{ID: uniqueID.ID}, nil
}

func (m Manifest) UpdateManifest(networkName string) {
	for i, module := range m.Modules {
		m.Modules[i].NetworkName = networkName
		m.Modules[i].ContainerName = makeContainerName(networkName, module.ImageNameFull, i)

		m.Modules[i].EnvArgs = append(m.Modules[i].EnvArgs, fmt.Sprintf("%v=%v", "INGRESS_HOST", m.Modules[i].ContainerName))
	}

	for start, ends := range m.Connections {
		var endpointStrings []string
		for _, end := range ends {
			endpointStrings = append(endpointStrings, fmt.Sprintf("http://%v:80/", m.Modules[end].ContainerName))
		}
		m.Modules[start].EnvArgs = append(m.Modules[start].EnvArgs, fmt.Sprintf("%v=%v", "EGRESS_URLS", strings.Join(endpointStrings, ",")))
	}
}

// makeContainerName is a simple utility to return a standard container name
// This function appends the pipelineID and containerName with _
func makeContainerName(networkName string, imageName string, index int) string {
	containerName := fmt.Sprint(networkName, ".", imageName, ".", index)

	// create regular expression for all alphanumeric characters and _ . -
	reg, err := regexp.Compile("[^A-Za-z0-9_.-]+")
	if err != nil {
		log.Fatal("Regular expression parsing failed! CAUSE --> ", err)
	}

	containerName = strings.ReplaceAll(containerName, " ", "")
	containerName = strings.ReplaceAll(containerName, ":", "_")
	containerName = reg.ReplaceAllString(containerName, "_")

	return containerName
}

func parseArguments(options []envMsg) ([]string, error) {
	log.Debug("Parsing environment arguments")

	var args []string
	for _, env := range options {
		var value string
		if env.Secret {
			var err error
			value, err = secret.DecryptEnv(env.Value)
			if err != nil {
				return nil, traceutility.Wrap(err)
			}
		} else {
			value = env.Value
		}
		args = append(args, fmt.Sprintf("%v=%v", env.Key, value))
	}
	return args, nil
}

func parseMounts(mnts []mountMsg) ([]mount.Mount, error) {
	log.Debug("Parsing mount points")

	mounts := []mount.Mount{}

	for _, mnt := range mnts {
		mount := mount.Mount{
			Type:        "bind",
			Source:      mnt.Host,
			Target:      mnt.Container,
			ReadOnly:    false,
			Consistency: "default",
			BindOptions: &mount.BindOptions{Propagation: "rprivate", NonRecursive: true},
		}

		mounts = append(mounts, mount)
	}

	return mounts, nil
}

func parseDevices(devs []deviceMsg) ([]container.DeviceMapping, error) {
	log.Debug("Parsing devices to attach")

	devices := []container.DeviceMapping{}

	for _, dev := range devs {
		device := container.DeviceMapping{
			PathOnHost:        dev.Host,
			PathInContainer:   dev.Container,
			CgroupPermissions: "rw",
		}

		devices = append(devices, device)
	}

	return devices, nil
}

func parsePorts(ports []portMsg) (nat.PortSet, nat.PortMap) {
	log.Debug("Parsing ports to bind")

	exposedPorts := nat.PortSet{}
	portBinding := nat.PortMap{}
	for _, port := range ports {
		hostPort := port.Host
		containerPort := port.Container
		exposedPorts[nat.Port(containerPort)] = struct{}{}
		portBinding[nat.Port(containerPort)] = []nat.PortBinding{{HostPort: hostPort}}
	}

	return exposedPorts, portBinding
}

func parseConnections(connectionsStringMap connectionsString) (connectionsInt, error) {
	log.Debug("Parsing modules' connections")

	connectionsIntMap := make(connectionsInt)

	for key, values := range connectionsStringMap {
		// if values is nill or empty, skip
		if values == nil || len(values) == 0 {
			continue
		}
		var valuesInt []int
		for _, value := range values {
			valueInt, err := strconv.Atoi(value)
			if err != nil {
				return nil, traceutility.Wrap(err)
			}
			// if valueInt is negative, skip
			if valueInt < 0 {
				continue
			}
			valuesInt = append(valuesInt, valueInt)
		}
		keyInt, err := strconv.Atoi(key)
		if err != nil {
			return nil, traceutility.Wrap(err)
		}
		connectionsIntMap[keyInt] = valuesInt
	}

	return connectionsIntMap, nil
}

func clearSecretValues(man Manifest) Manifest {
	// perform a deep copy, while removing env variables and passwords
	manCopy := man
	manCopy.Modules = make([]ContainerConfig, len(man.Modules))
	copy(manCopy.Modules, man.Modules)
	for i := range manCopy.Modules {
		manCopy.Modules[i].EnvArgs = nil
		manCopy.Modules[i].AuthConfig.Password = ""
	}
	return manCopy
}
