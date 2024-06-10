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

	// keyID      string
	debugMode  bool
	Address    string `json:"Address"`
	Transitkey string `json:"Transitkey"`
	Vaultrole  string `json:"Vaultrole"`
	Namespace  string `json:"Namespace"`
}

func NewVaultClientRemoteService(configFilePath string, debug bool) (service.Service, error) {
	ctx, err := os.ReadFile(configFilePath)
	if err != nil {
		log.Fatalln("EXIT:ctx: failed to read vault config file with error:\n", err.Error())
	}
	if len(keyID) == 0 {
		log.Fatalln("EXIT:keyID len: invalid keyID")
	}

	if debug {
		log.Println("DEBUG:--------------------------------------------------")
		log.Println("DEBUG: verifying keyID:", keyID)
	}

	vaultService := &hvaultRemoteService{
		// keyID: keyID,
		debugMode: debug,
	}

	json.Unmarshal(([]byte(ctx)), &vaultService)
	vaultconfig := api.DefaultConfig()
	vaultconfig.Address = vaultService.Address

	keypath := fmt.Sprintf("transit/keys/%s", vaultService.Transitkey)

	if debug {
		log.Println("DEBUG:--------------------------------------------------")
		log.Println("DEBUG: unmarshal JSON values:",
			"\n                    -> vaultService.debugMode", vaultService.debugMode,
			"\n                    -> vaultService.Address:", vaultService.Address, "\n                    -> vaultService.Transitkey:", vaultService.Transitkey, "\n                    -> vaultService.Vaultrole:", vaultService.Vaultrole, "\n                    -> vaultService.Namespace:", vaultService.Namespace, "\n                    -> keypath:", keypath)
	}

	client, err := api.NewClient(vaultconfig)
	if err != nil {
		if debug {
			log.Println("DEBUG:--------------------------------------------------")
			log.Println("DEBUG:client: json.Unmarshal output from configFile:", "\n vaultService.Address:", vaultService.Address)
			log.Println("DEBUG:--------------------------------------------------")
		}
		log.Fatalln("EXIT:client: failed to initialize Vault client with error:\n", err.Error())
	}

	k8sAuth, err := auth.NewKubernetesAuth(
		vaultService.Vaultrole,
	)

	if err != nil {
		if debug {
			log.Println("DEBUG:--------------------------------------------------")
			log.Println("DEBUG:k8sAuth: json.Unmarshal output from configFile:", "\n vaultService.Vaultrole:", vaultService.Vaultrole)
			log.Println("DEBUG:--------------------------------------------------")
		}
		log.Fatalln("EXIT:k8sAuth: unable to initialize Kubernetes auth method with error:\n", err.Error())
	}

	authInfo, err := client.Auth().Login(context.Background(), k8sAuth)
	if err != nil {
		log.Fatalln("EXIT:authInfo: unable to log in with Kubernetes auth with error:\n", err.Error())
	}
	if authInfo == nil {
		log.Fatalln("EXIT:authInfo: no kubernetes auth info was returned after login")
	}

	vaultService = &hvaultRemoteService{
		Client: client,
	}

	client.SetNamespace(vaultService.Namespace)

	key, err := client.Logical().Read(keypath)
	if err != nil {
		if debug {
			log.Println("DEBUG:--------------------------------------------------")
			log.Println("DEBUG:key: keypath:", keypath)
			log.Println("DEBUG:--------------------------------------------------")
		}
		log.Fatalln("EXIT:key: unable to find transit key:\n", err.Error())
	}

	log.Println("INFO: latest key version:", key.Data["latest_version"], "for provided transit key:", key.Data["name"])

	return vaultService, nil
}

func (s *hvaultRemoteService) Encrypt(ctx context.Context, uid string, plaintext []byte) (*service.EncryptResponse, error) {

	if s.debugMode {
		log.Println("DEBUG:--------------------------------------------------")
		log.Println("DEBUG: unencrypted payload:", string([]byte(plaintext)))
		log.Println("DEBUG:--------------------------------------------------")
	}

	enckeypath := fmt.Sprintf("transit/encrypt/%s", s.Transitkey)
	// keypath := "transit/encrypt/kleidi"
	encodepayload := map[string]interface{}{
		"plaintext": base64.StdEncoding.EncodeToString(plaintext),
	}

	encrypt, err := s.Logical().WriteWithContext(ctx, enckeypath, encodepayload)
	if err != nil {
		log.Println("--------------------------------------------------------")
		log.Println("DEBUG:encrypt:",
			"\n debugmode:", s.debugMode,
			"\nplaintext:", string([]byte(plaintext)),
			"\nkeypath:", enckeypath,
			"\nencodepayload:", encodepayload)
		log.Println("--------------------------------------------------------")
		log.Fatalln("EXIT:encrypt: with error:\n", err.Error())
	}
	enresult, ok := encrypt.Data["ciphertext"].(string)
	if !ok {
		log.Println("--------------------------------------------------------")
		log.Println("DEBUG:enresult:", "\nenresult:", string([]byte(enresult)))
		log.Println("--------------------------------------------------------")
		log.Fatalln("EXIT:enresult: invalid response")
	}

	return &service.EncryptResponse{
		Ciphertext: []byte(enresult),
		KeyID:      keyID,
		Annotations: map[string][]byte{
			annotationKey: []byte("1"),
		},
	}, nil
}

func (s *hvaultRemoteService) Decrypt(ctx context.Context, uid string, req *service.DecryptRequest) ([]byte, error) {

	if len(req.Annotations) != 1 {
		log.Println("--------------------------------------------------------")
		log.Println("DEBUG:len:", "\req.Annotations:", req.Annotations)
		log.Println("--------------------------------------------------------")
		return nil, fmt.Errorf("/!\\ invalid annotations")
	}
	if v, ok := req.Annotations[annotationKey]; !ok || string(v) != "1" {
		return nil, fmt.Errorf("/!\\ invalid version in annotations")
	}
	if req.KeyID != keyID {
		return nil, fmt.Errorf("/!\\ invalid keyID")
	}

	decryptkeypath := fmt.Sprintf("transit/decrypt/%s", s.Transitkey)
	// // keypath := fmt.Sprintf("transit/keys/%s", s.Transitkey)
	// keypath := "transit/decrypt/kleidi"
	encryptedPayload := map[string]interface{}{
		"ciphertext": string([]byte(req.Ciphertext)),
	}

	encryptedResponse, err := s.Logical().WriteWithContext(ctx, decryptkeypath, encryptedPayload)
	if err != nil {
		log.Println("--------------------------------------------------------")
		log.Println("DEBUG:encryptedResponse:", "\nkeypath:", decryptkeypath, "\nenresult:", encryptedPayload)
		log.Println("--------------------------------------------------------")
		log.Fatalln("EXIT:encryptedResponse: with error:", err.Error())
	}

	response, ok := encryptedResponse.Data["plaintext"].(string)
	if !ok {
		log.Println("--------------------------------------------------------")
		log.Println("DEBUG:response:", "\nresponse:", response)
		log.Println("--------------------------------------------------------")
		log.Fatalln("EXIT:response: invalid response")
	}

	decodepayload, err := base64.StdEncoding.DecodeString(response)
	if err != nil {
		log.Println("--------------------------------------------------------")
		log.Println("DEBUG:decodepayload:", "\npayload:", decodepayload)
		log.Println("--------------------------------------------------------")
		log.Fatalln("EXIT:decodepayload: with error:", err.Error())
	}

	return decodepayload, nil

}

func (s *hvaultRemoteService) Status(ctx context.Context) (*service.StatusResponse, error) {
	return &service.StatusResponse{
		Version: "v2",
		Healthz: "ok",
		KeyID:   keyID,
	}, nil
}
