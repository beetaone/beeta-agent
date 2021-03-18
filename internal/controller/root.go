package controller

import (
	"net/http"

	"github.com/bitly/go-simplejson"
	log "github.com/sirupsen/logrus"
)

// Status retruns status
func Status(w http.ResponseWriter, r *http.Request) {
	log.Debug("GET / (status)")
	json := simplejson.New()
	json.Set("status", "ok")
	json.Set("name", "Node Service")
	json.Set("location", "SIMULATION")
	json.Set("version", "0.1.1")
	payload, err := json.MarshalJSON()
	if err != nil {
		log.Println(err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(payload)
}
