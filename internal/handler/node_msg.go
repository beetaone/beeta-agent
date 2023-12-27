package handler

import (
	mqtt "github.com/eclipse/paho.mqtt.golang"
	log "github.com/sirupsen/logrus"

	"github.com/beetaone/beeta-agent/internal/edgeapp"
	"github.com/beetaone/beeta-agent/internal/model"
)

var NodeDeleteHandler mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
	log.Debugln("Received message on topic:", msg.Topic(), "Payload:", string(msg.Payload()))
	DeleteNode(model.NodeDeleted)
}

func DeleteNode(nodeStatus string) {
	log.Debug("Deleting node...")

	err := edgeapp.RemoveAll()
	if err != nil {
		log.Error("Deletion of node failed! CAUSE --> ", err)
	}

	edgeapp.SetNodeStatus(nodeStatus)
	edgeapp.SendStatus()
}
