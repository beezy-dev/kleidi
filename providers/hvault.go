/*
Hashicorp Vault iteration of the PKCS11 interface from KMSv2 mockup example
Author: rom@beezy.dev
Apache 2.0 License
*/
package providers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os"

	vault "github.com/hashicorp/vault/api"
	auth "github.com/hashicorp/vault/api/auth/kubernetes"
)

// var _ service.Service = &hvaultRemoteService{}

// type hvaultRemoteService struct {
// 	keyID string
// 	aead  cipher.AEAD
// }

type VaultClient struct {
	Address    string `json:"Address"`
	Transitkey string `json:"Transitkey"`
	Vaultrole  string `json:"Vaultrole"`
	Namespace  string `json:"Namespace"`
}

// (service.Service, error)

func NewHvaultRemoteService(configFilePath, keyID string) {
	c, err := os.ReadFile(configFilePath)
	if err != nil {
		log.Fatalln("EXIT: failed to read vault config file with error:", err.Error())
	}
	if len(keyID) == 0 {
		log.Fatalln("EXIT: invalid keyID")
	}

	// remoteService := &hvaultRemoteService{
	// 	keyID: keyID,
	// }

	var vaultclient VaultClient
	json.Unmarshal([]byte(c), &vaultclient)
	// log.Println("Vault Client Config:", vaultclient.Address, vaultclient.Namespace, vaultclient.Transitkey, vaultclient.Vaultrole)

	config := vault.DefaultConfig()
	config.Address = vaultclient.Address

	client, err := vault.NewClient(config)
	if err != nil {
		log.Fatalln("EXIT: failed to initialize Vault client with error:", err.Error())
	}

	k8sAuth, err := auth.NewKubernetesAuth(
		vaultclient.Vaultrole,
		// to consider if ServiceAccount Token would be mounted to a different location
		// auth.WithServiceAccountTokenPath("/var/run/secrets/kubernetes.io/serviceaccount/token"),
	)
	if err != nil {
		log.Fatalln("EXIT: unable to initialize Kubernetes auth method with error:", err.Error())
	}

	authInfo, err := client.Auth().Login(context.Background(), k8sAuth)
	if err != nil {
		log.Fatalln("EXIT: unable to log in with Kubernetes auth with error:", err.Error())
	}
	if authInfo == nil {
		log.Fatalln("EXIT: no kubernetes auth info was returned after login")
	}

	keypath := fmt.Sprintf("transit/keys/%s", vaultclient.Transitkey)
	// keypath:= fmt.Sprintf("transit/keys/kleidii")
	resp, err := client.Logical().Read(keypath)
	if err != nil {
		log.Fatalln("EXIT: unable to find transit key with error:", err.Error())
	}

	log.Println(resp.Data)
	log.Println("INFO: latest version:", resp.Data["latest_version"], "for provided transit key:", resp.Data["name"])

	// keys, ok := resp.Data["keys"].(map[string]interface{})
	// if !ok {
	// 	log.Fatalln("EXIT: could not get key version of transit key")
	// }

	// v, ok := keys[resp.Data["key_version"].(json.Number).String()].(json.Number)
	// if !ok {
	// 	log.Fatalln("EXIT: could not find key version for transit key", vaultclient.Transitkey)
	// }
	// log.Fatalln(v.String())

	// encryption testing
	encryptpath := fmt.Sprintf("transit/encrypt/%s", vaultclient.Transitkey)

	log.Println("--------------------------------------------------------------------------------------------------")
	log.Println("DEBUG: starting encryption/decryption test with vault client using keyID")
	mymessage := []byte(keyID)
	log.Println("INFO: unencrypted keyID:", string([]byte(mymessage)))

	encpayload := map[string]interface{}{
		"plaintext": base64.StdEncoding.EncodeToString(mymessage),
	}

	encrypt, err := client.Logical().Write(encryptpath, encpayload)
	if err != nil {
		log.Fatalln("EXIT: with error:", err.Error())
	}
	enresult, ok := encrypt.Data["ciphertext"].(string)
	if !ok {
		log.Fatalln("EXIT: invalid response")
	}

	log.Println("INFO: encrypted keyID:", string([]byte(enresult)))
	// return []byte(result), resp.Data["latest_version"], nil

	// decryption testing
	decryptpath := fmt.Sprintf("transit/decrypt/%s", vaultclient.Transitkey)
	decpayload := map[string]interface{}{
		"ciphertext": string([]byte(enresult)),
	}

	decrypt, err := client.Logical().Write(decryptpath, decpayload)
	if err != nil {
		log.Fatalln("EXIT: with error:", err.Error())
	}

	deresult, ok := decrypt.Data["plaintext"].(string)
	if !ok {
		log.Fatalln("EXIT: invalid response")
	}
	decodeb64, err := base64.StdEncoding.DecodeString(deresult)
	if err != nil {
		log.Fatalln("EXIT: with error:", err.Error())
	}

	log.Println("INFO: decrypted keyID:", string([]byte(decodeb64)))
	// return decodeb64
	log.Println("DEBUG: ending encryption/decryption test with vault client")
	log.Println("--------------------------------------------------------------------------------------------------")
	log.Fatalln("/!\\ implementation in progress - stay tune!")
}
