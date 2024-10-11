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
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/hashicorp/vault/api"
	auth "github.com/hashicorp/vault/api/auth/kubernetes"
	"k8s.io/klog/v2"
	"k8s.io/kms/pkg/service"
)

var _ service.Service = &hvaultRemoteService{}

type hvaultRemoteService struct {
	*api.Client

	UnixSock    string
	LatestKeyID string
	// keyID      string
	Debug      bool
	Namespace  string `json:"namespace"`
	Transitkey string `json:"transitkey"`
	Vaultrole  string `json:"vaultrole"`
	Address    string `json:"address"`
}

func NewVaultClientRemoteService(configFilePath string, addr string) (service.Service, error) {
	ctx, err := os.ReadFile(configFilePath)
	if err != nil {
		klog.Fatal("EXIT:ctx: failed to read vault config file with error:\n", err.Error())
	}
	if len(keyID) == 0 {
		klog.Fatal("EXIT:keyID len: invalid keyID")
	}

	//	if debug {
	//		klog.Info("DEBUG:--------------------------------------------------")
	//		klog.Info("DEBUG: verifying keyID:", keyID)
	//	}

	// vaultService := &hvaultRemoteService{
	// 	// keyID: keyID,
	// 	Debug: debug,
	// }

	vaultService := &hvaultRemoteService{}
	//	vaultService.Debug = debug
	vaultService.UnixSock = addr
	json.Unmarshal(([]byte(ctx)), &vaultService)

	vaultconfig := api.DefaultConfig()
	vaultconfig.Address = vaultService.Address

	// keypath := fmt.Sprintf("transit/keys/%s", vaultService.Transitkey)

	//	if debug {
	//		klog.Info("DEBUG:--------------------------------------------------")
	//		klog.Info("DEBUG: unmarshal JSON values:",
	//			"\n                    -> vaultService.debug", vaultService.Debug,
	//			"\n                    -> vaultService.Address:", vaultService.Address,
	//			"\n                    -> vaultService.Transitkey:", vaultService.Transitkey,
	//			"\n                    -> vaultService.Vaultrole:", vaultService.Vaultrole,
	//			"\n                    -> vaultService.Namespace:", vaultService.Namespace,
	//			"\n                    -> keypath:", keypath)
	//	}

	client, err := api.NewClient(vaultconfig)
	if err != nil {
		//		if debug {
		//			klog.Info("DEBUG:--------------------------------------------------")
		//			klog.Info("DEBUG:client: json.Unmarshal output from configFile:", "\n vaultService.Address:", vaultService.Address)
		//			klog.Info("DEBUG:--------------------------------------------------")
		//		}
		//		klog.Fatal("EXIT:client: failed to initialize Vault client with error:\n", err.Error())
	}

	k8sAuth, err := auth.NewKubernetesAuth(
		vaultService.Vaultrole,
	)

	if err != nil {
		//		if debug {
		//			klog.Info("DEBUG:--------------------------------------------------")
		//			klog.Info("DEBUG:k8sAuth: json.Unmarshal output from configFile:", "\n vaultService.Vaultrole:", vaultService.Vaultrole)
		//			klog.Info("DEBUG:--------------------------------------------------")
		//		}
		//		klog.Fatal("EXIT:k8sAuth: unable to initialize Kubernetes auth method with error:\n", err.Error())
	}

	authInfo, err := client.Auth().Login(context.Background(), k8sAuth)
	if err != nil {
		klog.Fatal("EXIT:authInfo: unable to log in with Kubernetes auth with error:\n", err.Error())
	}
	if authInfo == nil {
		klog.Fatal("EXIT:authInfo: no kubernetes auth info was returned after login")
	}

	// vaultService = &hvaultRemoteService{
	// 	Client: client,
	// }
	vaultService.Client = client
	client.SetNamespace(vaultService.Namespace)

	// obtain latest version of the transit key and create a key ID for it
	key, err := vaultService.GetTransitKey(context.Background())
	if err != nil {
		klog.Fatal("ERROR:key: unable to find transit key, restarting:\n", err.Error())
	}
	vaultService.LatestKeyID = createLatestTransitKeyId(key)

	klog.Info("INFO: latest key version:", key.Data["latest_version"], "for provided transit key:", key.Data["name"])
	klog.Info("INFO: latest key id for plugin:", vaultService.LatestKeyID)

	// initial token check - it can happen that k8s restarted ??
	err = vaultService.CheckTokenValidity(context.Background())
	if err != nil {
		klog.Fatal("EXIT:token: could not check token validity: \n", err.Error())
		return vaultService, err
	}

	return vaultService, nil
}

func (s *hvaultRemoteService) Encrypt(ctx context.Context, uid string, plaintext []byte) (*service.EncryptResponse, error) {

	//	if s.Debug {
	//		klog.Info("DEBUG:--------------------------------------------------")
	//		klog.Info("DEBUG: unencrypted payload:", string([]byte(plaintext)))
	//		klog.Info("DEBUG:--------------------------------------------------")
	//	}

	//	klog.Info("DEBUG:--------------------------------------------------")
	//	klog.Info("DEBUG: unmarshal JSON values:",
	//		"\n                    -> vaultService.debug", s.Debug,
	//		"\n                    -> vaultService.Address:", s.Address,
	//		"\n                    -> vaultService.Transitkey:", s.Transitkey,
	//		"\n                    -> vaultService.Vaultrole:", s.Vaultrole,
	//		"\n                    -> vaultService.Namespace:", s.Namespace)

	enckeypath := fmt.Sprintf("transit/encrypt/%s", s.Transitkey)
	// keypath := "transit/encrypt/kleidi"
	encodepayload := map[string]interface{}{
		"plaintext": base64.StdEncoding.EncodeToString(plaintext),
	}

	encrypt, err := s.Logical().WriteWithContext(ctx, enckeypath, encodepayload)
	if err != nil {
		klog.Info("--------------------------------------------------------")
		klog.Info("DEBUG:encrypt:",
			"\n debug:", s.Debug,
			"\nplaintext:", string([]byte(plaintext)),
			"\nkeypath:", enckeypath,
			"\nencodepayload:", encodepayload)
		klog.Info("--------------------------------------------------------")
		klog.Fatal("EXIT:encrypt: with error:\n", err.Error())
	}
	enresult, ok := encrypt.Data["ciphertext"].(string)
	if !ok {
		klog.Info("--------------------------------------------------------")
		klog.Info("DEBUG:enresult:", "\nenresult:", string([]byte(enresult)))
		klog.Info("--------------------------------------------------------")
		klog.Fatal("EXIT:enresult: invalid response")
	}

	return &service.EncryptResponse{
		Ciphertext: []byte(enresult),
		KeyID:      s.LatestKeyID,
		Annotations: map[string][]byte{
			annotationKey: []byte("1"),
		},
	}, nil
}

func (s *hvaultRemoteService) Decrypt(ctx context.Context, uid string, req *service.DecryptRequest) ([]byte, error) {

	if len(req.Annotations) != 1 {
		klog.Info("--------------------------------------------------------")
		klog.Info("DEBUG:len:", "\req.Annotations:", req.Annotations)
		klog.Info("--------------------------------------------------------")
		return nil, fmt.Errorf("/!\\ invalid annotations")
	}
	if v, ok := req.Annotations[annotationKey]; !ok || string(v) != "1" {
		return nil, fmt.Errorf("/!\\ invalid version in annotations")
	}
	// if req.KeyID != s.LatestKeyID {
	// 	return nil, fmt.Errorf("/!\\ invalid keyID")
	// }

	decryptkeypath := fmt.Sprintf("transit/decrypt/%s", s.Transitkey)
	// // keypath := fmt.Sprintf("transit/keys/%s", s.Transitkey)
	// keypath := "transit/decrypt/kleidi"
	encryptedPayload := map[string]interface{}{
		"ciphertext": string([]byte(req.Ciphertext)),
	}

	encryptedResponse, err := s.Logical().WriteWithContext(ctx, decryptkeypath, encryptedPayload)
	if err != nil {
		klog.Info("--------------------------------------------------------")
		klog.Info("DEBUG:encryptedResponse:", "\nkeypath:", decryptkeypath, "\nenresult:", encryptedPayload)
		klog.Info("--------------------------------------------------------")
		klog.Fatal("EXIT:encryptedResponse: with error:", err.Error())
	}

	response, ok := encryptedResponse.Data["plaintext"].(string)
	if !ok {
		klog.Info("--------------------------------------------------------")
		klog.Info("DEBUG:response:", "\nresponse:", response)
		klog.Info("--------------------------------------------------------")
		klog.Fatal("EXIT:response: invalid response")
	}

	decodepayload, err := base64.StdEncoding.DecodeString(response)
	if err != nil {
		klog.Info("--------------------------------------------------------")
		klog.Info("DEBUG:decodepayload:", "\npayload:", decodepayload)
		klog.Info("--------------------------------------------------------")
		klog.Fatal("EXIT:decodepayload: with error:", err.Error())
	}

	return decodepayload, nil

}

func (s *hvaultRemoteService) Status(ctx context.Context) (*service.StatusResponse, error) {
	// check if unix socket is still present
	if _, err := os.Stat(s.UnixSock); errors.Is(err, os.ErrNotExist) {
		klog.Fatal("ERROR:status: socket removed ", err.Error())
		return s.createStatusResponse(healthNOK), err
	}
	if s.Debug {
		klog.Info("DEBUG:Status: old latest key ID:", s.LatestKeyID)
	}
	// get transit key, obtain the latest version of the transit key
	key, err := s.GetTransitKey(ctx)
	if err != nil {
		klog.Fatal("ERROR:key: unable to find transit key, restarting:\n", err.Error())
		return s.createStatusResponse(healthNOK), err
	}
	// extract the latest and create key id for it
	s.LatestKeyID = createLatestTransitKeyId(key)
	if s.Debug {
		klog.Info("DEBUG:Status: new latest key ID:", s.LatestKeyID)
	}
	// do healthcheck
	err = s.Health(ctx)
	if err != nil {
		klog.Fatal("ERROR:Status: unhealthy:\n", err.Error())
		return s.createStatusResponse(healthNOK), err
	}

	return s.createStatusResponse(healthOK), nil
}

func (s *hvaultRemoteService) Health(ctx context.Context) error {
	// check if it has valid token lease (Vault)
	err := s.CheckTokenValidity(ctx)
	if err != nil {
		klog.Fatal("ERROR:health:token: token validity check failed:\n", err.Error())
		return err
	}
	// check encrypt/decrypt if operation can be performed correctly
	enc, err := s.Encrypt(ctx, fmt.Sprintf("health-enc-%s", strconv.FormatInt(time.Now().Unix(), 10)), []byte(healthy))
	if err != nil {
		if s.Debug {
			klog.Info("DEBUG:Health: encrypt failed: ", err.Error())
		}
		return err
	}

	dec, err := s.Decrypt(ctx, fmt.Sprintf("health-dec-%s", strconv.FormatInt(time.Now().Unix(), 10)), &service.DecryptRequest{
		Ciphertext: enc.Ciphertext,
		KeyID:      s.LatestKeyID,
		Annotations: map[string][]byte{
			annotationKey: []byte("1"),
		},
	})

	if err != nil {
		if s.Debug {
			klog.Info("DEBUG:Health: decrypt failed: ", err.Error())
		}
		return err
	}

	// decrypted plaintext does not match
	if healthy != string(dec) {
		return errors.New("ERROR:Health check FAILED")
	}

	if s.Debug {
		klog.Info("DEBUG:Health check OK")
	}

	return nil
}

func (s *hvaultRemoteService) createStatusResponse(healthz string) *service.StatusResponse {
	// creates status response ok/nok with latest key ID
	return &service.StatusResponse{
		Version: "v2",
		Healthz: healthz,
		KeyID:   s.LatestKeyID,
	}
}

func (s *hvaultRemoteService) GetTransitKey(ctx context.Context) (*api.Secret, error) {
	key, err := s.Client.Logical().ReadWithContext(ctx, fmt.Sprintf("transit/keys/%s", s.Transitkey))
	if err != nil {
		// no transit key or no token
		return nil, err
	}
	if s.Debug {
		klog.Info("DEBUG: transit key: ", key)
	}
	return key, nil
}

func createLatestTransitKeyId(key *api.Secret) string {
	latest_version := fmt.Sprintf("%s", key.Data["latest_version"])
	keys := make(map[string]interface{})
	if a, ok := key.Data["keys"].(map[string]interface{}); ok {
		keys = a
	}
	// key id is concatenated from keyID (constant), field latest_version (a number),
	// field keys[latest_version] which is creation timestamp of that key version
	latest_key_id := fmt.Sprintf("%s_%s_%s", keyID, latest_version, keys[latest_version])
	return latest_key_id
}

func (s *hvaultRemoteService) GetVaultToken(ctx context.Context) (*api.Secret, error) {
	// requires policy to have: "auth/token/lookup-self read and "auth/token/renew-self" update
	path := fmt.Sprintf("auth/token/lookup-self")
	token, err := s.Client.Logical().ReadWithContext(ctx, path)
	if err != nil {
		klog.Info("ERROR:token:path: cannot read path: \n", err.Error())
		return nil, err
	}
	return token, nil
}

func (s *hvaultRemoteService) CheckTokenValidity(ctx context.Context) error {
	token, err := s.GetVaultToken(ctx)
	if err != nil {
		// could not get token - re-authentication needed
		klog.Fatal("ERROR:token:check could not get token: ", err.Error())
		return err
	}
	if s.Debug {
		// beware - it prints out the token itself!!
		klog.Info("DEBUG:token: token_data received: ", token)
	}

	creation_ttl, _ := strconv.Atoi(fmt.Sprintf("%s", token.Data["creation_ttl"]))
	ttl, _ := strconv.Atoi(fmt.Sprintf("%s", token.Data["ttl"]))

	if ttl <= 0 || ttl > creation_ttl {
		// token has been tampered/reboot happened
		// also if you modify role's ttl with e.g.
		// vault cli like vault write auth/kubernetes/role/kleidi ttl=1h (meaning you want to renew it by hand)
		// it's okay if token is renewed with vault token renew -accessor ...
		klog.Fatal("ERROR:token: invalid ttl, re-login needed")
		return errors.New("ERROR:token invalid ttl, re-login needed")
	}
	// update the token if it reached it's validity periods about 2/3rd
	if ttl <= creation_ttl-int(float32(creation_ttl)*0.667) {
		// update the token
		if s.Debug {
			klog.Info("DEBUG:token: Updating the token!!!")
		}
		err = s.RenewOwnToken(ctx, creation_ttl)
		if err != nil {
			klog.Info("ERROR:token: could not renew token: ", err.Error())
			return err
		}
	}
	// no need for token update
	if s.Debug {
		klog.Info("DEBUG:token: No need for token update.")
	}
	return nil
}

func (s *hvaultRemoteService) RenewOwnToken(ctx context.Context, creation_ttl int) error {
	// renews with the original creation_ttl
	path := fmt.Sprintf("auth/token/renew-self")
	_, err := s.Client.Logical().WriteWithContext(ctx, path, map[string]any{"data": map[string]any{
		"ttl":       fmt.Sprintf("%d", creation_ttl),
		"renewable": "true"}})
	if err != nil {
		klog.Info("ERROR:token:path: Something went wrong with token update: \n", err.Error())
		return err
	}
	if s.Debug {
		klog.Info("DEBUG:token: Token update successful.")
	}
	return nil
}
