package model

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
)

func LoadJsonBytes(manName string) []byte {
	wd, _ := os.Getwd()
	fmt.Println()
	manifestPath := path.Join(wd, "testdata", manName)
	// fmt.Println("Loading manifest from ", manifestPath)

	var err error = nil
	manifestBytes, err := ioutil.ReadFile(manifestPath)
	if err != nil {
		log.Fatal(err)
	}
	return manifestBytes
}