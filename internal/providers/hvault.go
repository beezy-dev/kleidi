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

// type VaultClientOptions func(*VaultClient) error

func NewVaultClientRemoteService(configFilePath, keyID string) (service.Service, error) {
	ctx, err := os.ReadFile(configFilePath)
	// config, err := os.ReadFile(configFilePath)
	if err != nil {
		log.Fatalln("EXIT: failed to read vault config file with error:", err.Error())
	}
	if len(keyID) == 0 {
		log.Fatalln("EXIT: invalid keyID")
	}

	vaultService := &hvaultRemoteService{
		keyID: keyID,
	}

	// vaultclient := &VaultClient{}
	json.Unmarshal(([]byte(ctx)), &vaultService)
	vaultconfig := api.DefaultConfig()
	vaultconfig.Address = vaultService.Address

	log.Println("DEBUG: json.Unmarshal output from configFile:", vaultService.Address, vaultService.Namespace, vaultService.Transitkey, vaultService.Vaultrole)

	client, err := api.NewClient(vaultconfig)
	if err != nil {
		log.Fatalln("EXIT: failed to initialize Vault client with error:", err.Error())
	}

	k8sAuth, err := auth.NewKubernetesAuth(
		vaultService.Vaultrole,
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

	vaultService = &hvaultRemoteService{
		Client: client,
	}

	keypath := fmt.Sprintf("transit/keys/%s", vaultService.Transitkey)
	// keypath:= fmt.Sprintf("transit/keys/kleidii")

	key, err := client.Logical().Read(keypath)
	if err != nil {
		log.Fatalln("EXIT: unable to find transit key with error:", err.Error())
	}

	// log.Println(resp.Data)
	log.Println("INFO: latest key version:", key.Data["latest_version"], "for provided transit key:", key.Data["name"])

	return vaultService, nil
}

func (s *hvaultRemoteService) Encrypt(ctx context.Context, uid string, plaintext []byte) (*service.EncryptResponse, error) {

	// log.Println("--------------------------------------------------------------------------------------------------")
	// log.Println("DEBUG: starting encryption/decryption test with vault client using keyID")
	// log.Println("INFO: unencrypted keyID:", string([]byte(payload)))

	keypath := fmt.Sprintf("transit/keys/%s", s.Transitkey)
	encodepayload := map[string]interface{}{
		"plaintext": base64.StdEncoding.EncodeToString(plaintext),
	}

	encrypt, err := s.Logical().WriteWithContext(ctx, keypath, encodepayload)
	if err != nil {
		log.Fatalln("EXIT: with error:", err.Error())
	}
	enresult, ok := encrypt.Data["ciphertext"].(string)
	if !ok {
		log.Fatalln("EXIT: invalid response")
	}

	// log.Println("INFO: encrypted keyID:", string([]byte(enresult)))

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

	keypath := fmt.Sprintf("transit/keys/%s", s.Transitkey)
	encryptedPayload := map[string]interface{}{
		"ciphertext": string([]byte(req.Ciphertext)),
	}
	encryptedResponse, err := s.Logical().WriteWithContext(ctx, keypath, encryptedPayload)
	if err != nil {
		log.Fatalln("EXIT: with error:", err.Error())
	}

	response, ok := encryptedResponse.Data["plaintext"].(string)
	if !ok {
		log.Fatalln("EXIT: invalid response")
	}
	decodepayload, err := base64.StdEncoding.DecodeString(response)
	if err != nil {
		log.Fatalln("EXIT: with error:", err.Error())
	}

	return decodepayload, nil

}

func (s *hvaultRemoteService) Status(ctx context.Context) (*service.StatusResponse, error) {
	return &service.StatusResponse{
		Version: "v2",
		Healthz: "ok",
		KeyID:   s.keyID,
	}, nil
}
