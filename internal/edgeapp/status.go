package edgeapp

import (
	"strings"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"

	"github.com/beetaone/beeta-agent/internal/com"
	"github.com/beetaone/beeta-agent/internal/docker"
	"github.com/beetaone/beeta-agent/internal/manifest"
	"github.com/beetaone/beeta-agent/internal/model"
	"github.com/beetaone/beeta-agent/internal/secret"
	ioutility "github.com/beetaone/beeta-agent/internal/utility/io"
	traceutility "github.com/beetaone/beeta-agent/internal/utility/trace"
)

var nodeStatus string = model.NodeDisconnected

func SetNodeStatus(status string) {
	nodeStatus = status
}

func SendStatus() error {
	msg, err := GetStatusMessage()
	if err != nil {
		return traceutility.Wrap(err)
	}
	err = com.SendHeartbeat(msg)
	if err != nil {
		return traceutility.Wrap(err)
	}
	return nil
}

func GetStatusMessage() (com.StatusMsg, error) {
	edgeApps, err := GetEdgeAppStatus()
	if err != nil {
		return com.StatusMsg{}, traceutility.Wrap(err)
	}

	deviceParams, err := getDeviceParams()
	if err != nil {
		return com.StatusMsg{}, traceutility.Wrap(err)
	}

	msg := com.StatusMsg{
		Status:           nodeStatus,
		EdgeApplications: edgeApps,
		DeviceParams:     deviceParams,
		AgentVersion:     model.Version,
		OrgKeyHash:       secret.OrgKeyHash,
	}

	return msg, nil
}

func GetEdgeAppStatus() ([]com.EdgeAppMsg, error) {
	edgeApps := []com.EdgeAppMsg{}

	for _, manif := range manifest.GetKnownManifests() {
		edgeApplication := com.EdgeAppMsg{ManifestID: manif.Manifest.ID, Status: manif.Status}

		if manif.Status == model.EdgeAppUndeployed {
			edgeApps = append(edgeApps, edgeApplication)
			continue
		}

		appContainers, err := docker.ReadEdgeAppContainers(manif.Manifest.UniqueID)
		if err != nil {
			return edgeApps, traceutility.Wrap(err)
		}

		if (manif.Status == model.EdgeAppRunning || manif.Status == model.EdgeAppStopped) && len(appContainers) != len(manif.Manifest.Modules) {
			edgeApplication.Status = model.EdgeAppError
		}

		containersStat := []com.ContainerMsg{}
		for _, con := range appContainers {
			containerJSON, err := docker.InspectContainer(con.ID)
			if err != nil {
				return edgeApps, traceutility.Wrap(err)
			}
			// The Status of each container is (assumed to be): Running, Restarting, Created, Exited
			container := com.ContainerMsg{Name: strings.Join(con.Names, ", "), Status: ioutility.FirstToUpper(con.State)}
			containersStat = append(containersStat, container)

			if (manif.Status != model.EdgeAppInitiated && manif.Status != model.EdgeAppExecuting) && edgeApplication.Status != model.EdgeAppError {
				if manif.Status == model.EdgeAppRunning && con.State != strings.ToLower(model.ModuleRunning) {
					edgeApplication.Status = model.EdgeAppError
				}
				if manif.Status == model.EdgeAppStopped && (con.State != strings.ToLower(model.ModuleExited) || (containerJSON.State.ExitCode != 0 && containerJSON.State.ExitCode != 137)) {
					edgeApplication.Status = model.EdgeAppError
				}
			}
		}

		edgeApplication.Containers = containersStat
		edgeApps = append(edgeApps, edgeApplication)
	}

	return edgeApps, nil
}

func CompareEdgeAppStatus(edgeApps []com.EdgeAppMsg) ([]com.EdgeAppMsg, bool, error) {
	statusChange := false

	latestEdgeApps, err := GetEdgeAppStatus()
	if err != nil {
		return nil, false, traceutility.Wrap(err)
	}
	if len(edgeApps) == len(latestEdgeApps) {
		for index, edgeApp := range edgeApps {
			if edgeApp.Status != latestEdgeApps[index].Status {
				statusChange = true
			}
		}
	} else {
		statusChange = true
	}
	return latestEdgeApps, statusChange, nil
}

func getDeviceParams() (com.DeviceParamsMsg, error) {
	uptime, err := host.Uptime()
	if err != nil {
		return com.DeviceParamsMsg{}, traceutility.Wrap(err)
	}

	cpu, err := cpu.Percent(0, false)
	if err != nil {
		return com.DeviceParamsMsg{}, traceutility.Wrap(err)
	}

	diskStat, err := disk.Usage("/")
	if err != nil {
		return com.DeviceParamsMsg{}, traceutility.Wrap(err)
	}

	verMem, err := mem.VirtualMemory()
	if err != nil {
		return com.DeviceParamsMsg{}, traceutility.Wrap(err)
	}

	params := com.DeviceParamsMsg{
		SystemUpTime: uptime,
		SystemLoad:   cpu[0],
		StorageFree:  100.0 - diskStat.UsedPercent,
		RamFree:      float64(verMem.Available) / float64(verMem.Total) * 100.0,
	}

	return params, nil
}
