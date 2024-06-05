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

	"github.com/hashicorp/vault/api"
	auth "github.com/hashicorp/vault/api/auth/kubernetes"
	"k8s.io/kms/pkg/service"
)

var _ service.Service = &hvaultRemoteService{}

type hvaultRemoteService struct {
	*api.Client

	keyID      string
	Address    string `json:"Address"`
	Transitkey string `json:"Transitkey"`
	Vaultrole  string `json:"Vaultrole"`
	Namespace  string `json:"Namespace"`
}

func NewVaultClientRemoteService(configFilePath, keyID string) (service.Service, error) {
	ctx, err := os.ReadFile(configFilePath)
	if err != nil {
		log.Fatalln("EXIT:ctx: failed to read vault config file with error:", err.Error())
	}
	if len(keyID) == 0 {
		log.Fatalln("EXIT:keyID len: invalid keyID")
	}

	vaultService := &hvaultRemoteService{
		keyID: keyID,
	}

	json.Unmarshal(([]byte(ctx)), &vaultService)
	vaultconfig := api.DefaultConfig()
	vaultconfig.Address = vaultService.Address

	log.Println("--------------------------------------------------------------------------------------------------")
	log.Println("DEBUG: json.Unmarshal output from configFile:", vaultService.Address, vaultService.Namespace, vaultService.Transitkey, vaultService.Vaultrole, keyID)
	log.Println("--------------------------------------------------------------------------------------------------")

	client, err := api.NewClient(vaultconfig)
	if err != nil {
		log.Fatalln("EXIT:client: failed to initialize Vault client with error:", err.Error())
	}

	k8sAuth, err := auth.NewKubernetesAuth(
		vaultService.Vaultrole,
	)

	if err != nil {
		log.Fatalln("EXIT:k8sAuth: unable to initialize Kubernetes auth method with error:", err.Error())
	}

	authInfo, err := client.Auth().Login(context.Background(), k8sAuth)
	if err != nil {
		log.Fatalln("EXIT:authInfo: unable to log in with Kubernetes auth with error:", err.Error())
	}
	if authInfo == nil {
		log.Fatalln("EXIT:authInfo: no kubernetes auth info was returned after login")
	}

	vaultService = &hvaultRemoteService{
		Client: client,
	}

	client.SetNamespace(vaultService.Namespace)

	//keypath := fmt.Sprintf("transit/keys/%s", vaultService.Transitkey)
	keypath := "transit/keys/kleidi"

	key, err := client.Logical().Read(keypath)
	if err != nil {
		log.Fatalln("EXIT:key: unable to find transit key in:", keypath, "with error:", err.Error())
	}

	log.Println("INFO: latest key version:", key.Data["latest_version"], "for provided transit key:", key.Data["name"])

	return vaultService, nil
}

func (s *hvaultRemoteService) Encrypt(ctx context.Context, uid string, plaintext []byte) (*service.EncryptResponse, error) {

	// log.Println("--------------------------------------------------------------------------------------------------")
	// log.Println("DEBUG: unencrypted payload:", string([]byte(plaintext)))
	// log.Println("--------------------------------------------------------------------------------------------------")

	// keypath := fmt.Sprintf("transit/keys/%s", s.Transitkey)
	keypath := "transit/encrypt/kleidi"
	encodepayload := map[string]interface{}{
		"plaintext": base64.StdEncoding.EncodeToString(plaintext),
	}

	encrypt, err := s.Logical().WriteWithContext(ctx, keypath, encodepayload)
	if err != nil {
		log.Fatalln("EXIT:encrypt: with error:", err.Error())
	}
	enresult, ok := encrypt.Data["ciphertext"].(string)
	if !ok {
		// log.Println("--------------------------------------------------------------------------------------------------")
		// log.Println("DEBUG: encrypted payload:", string([]byte(enresult)))
		// log.Println("--------------------------------------------------------------------------------------------------")
		log.Fatalln("EXIT:enresult: invalid response")
	}

	return &service.EncryptResponse{
		Ciphertext: []byte(enresult),
		KeyID:      s.keyID,
		Annotations: map[string][]byte{
			annotationKey: []byte("1"),
		},
	}, nil
}

func (s *hvaultRemoteService) Decrypt(ctx context.Context, uid string, req *service.DecryptRequest) ([]byte, error) {

	if len(req.Annotations) != 1 {
		return nil, fmt.Errorf("/!\\ invalid annotations")
	}
	if v, ok := req.Annotations[annotationKey]; !ok || string(v) != "1" {
		return nil, fmt.Errorf("/!\\ invalid version in annotations")
	}
	if req.KeyID != s.keyID {
		return nil, fmt.Errorf("/!\\ invalid keyID")
	}

	// keypath := fmt.Sprintf("transit/keys/%s", s.Transitkey)
	keypath := "transit/decrypt/kleidi"
	encryptedPayload := map[string]interface{}{
		"ciphertext": string([]byte(req.Ciphertext)),
	}

	encryptedResponse, err := s.Logical().WriteWithContext(ctx, keypath, encryptedPayload)
	if err != nil {
		// log.Println("--------------------------------------------------------------------------------------------------")
		// log.Println("DEBUG: encrypted request:", string([]byte(req.Ciphertext)))
		// log.Println("--------------------------------------------------------------------------------------------------")
		log.Fatalln("EXIT:encryptedResponse: with error:", err.Error())
	}

	response, ok := encryptedResponse.Data["plaintext"].(string)
	if !ok {
		// log.Println("--------------------------------------------------------------------------------------------------")
		// log.Println("DEBUG: decrypted base64 encodeded payload:", encryptedResponse.Data["plaintext"].(string))
		// log.Println("--------------------------------------------------------------------------------------------------")
		log.Fatalln("EXIT:response: invalid response")
	}

	decodepayload, err := base64.StdEncoding.DecodeString(response)
	if err != nil {
		// log.Println("--------------------------------------------------------------------------------------------------")
		// log.Println("DEBUG: decrypted base64 decodeded payload:", base64.StdEncoding.DecodeString(response))
		// log.Println("--------------------------------------------------------------------------------------------------")
		log.Fatalln("EXIT:decodepayload: with error:", err.Error())
	}

	return decodepayload, nil

}

func (s *hvaultRemoteService) Status(ctx context.Context) (*service.StatusResponse, error) {
	return &service.StatusResponse{
		Version: "v2",
		Healthz: "ok",
		KeyID:   "kleidi-kms-plugin",
	}, nil
}
