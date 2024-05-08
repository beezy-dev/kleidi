package providers

import (
	"encoding/json"
	"log"
	"os"
)

func HvaultPlaceholder() {
	log.Fatalln("/!\\ implementation in progress - stay tune!")
}

// var _ service.Service = &hvaultRemoteService{}

// type hvaultRemoteService struct {
// 	keyID string
// 	aead  cipher.AEAD
// }

// defaults
// address 127.0.0.1:8200
// transitKey: kleidikey
// vaultRole: kleidirole
// nameSpace: admin (always for community, can be differnent in enterprise)
type ClientConfig struct {
	Address    string `json:"Address"`
	Transitkey string `json:"Transitkey"`
	Vaultrole  string `json:"Vaultrole"`
	Namespace  string `json:"Namespace"`
}

func ConfigFileToClientConfig(configFilePath string) {
	c, err := os.ReadFile(configFilePath)
	if err != nil {
		log.Fatalln("EXIT: failed to read vault config file with error:", err.Error())
	}
	var clientconfig ClientConfig
	json.Unmarshal([]byte(c), &clientconfig)
	log.Println("Vault Client Config:", clientconfig.Address, clientconfig.Namespace, clientconfig.Transitkey, clientconfig.Vaultrole)

}

// func NewVaultRemoteService(configFilePath, keyID string) (service.Service, error) {

// 	config := vault.DefaultConfig()
// 	config.Address = clientConfig.adress

// }
