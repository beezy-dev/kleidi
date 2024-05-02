package utils

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
)

type VaultConfig struct {
	Namespace string
	Transit   string
	Role      string
	Path      string
}

func ConfigureFromFile(configFilePath, provider string) (*VaultConfig, error) {
	configFile, err := os.Open(configFilePath)
	if err != nil {
		log.Fatalln("EXIT: fail to open -configfile", configFilePath, "with error:\n", err.Error())
	}
	defer func() {
		closeErr := configFile.Close()
		if err == nil {
			err = closeErr
		}
	}()

	decodeConfigFile := json.NewDecoder(configFile)
	config := &VaultConfig{}
	err = decodeConfigFile.Decode(config)
	if err != nil {
		log.Fatalln("EXIT: fail to open -configfile", configFilePath, "with error:\n", err.Error())
	}

	fmt.Println(config.Path)
	return config, err
}
