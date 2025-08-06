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
	"strings"
	"time"

	"github.com/hashicorp/vault/api"
	"k8s.io/kms/pkg/service"
	"go.uber.org/zap"
)

const (
	retrySleep = 150*time.Millisecond
)

var _ service.Service = &hvaultRemoteService{}

type hvaultRemoteService struct {
	*api.Client
	ClientAuthMethod api.AuthMethod

	LatestKeyID string
	Namespace   string `json:"namespace"`
	Transitkey  string `json:"transitkey"`
	Vaultrole   string `json:"vaultrole"`
	Address     string `json:"address"`
	AuthPath    string `json:"authpath"`
	TransitPath string `json:"transitpath"`
	AuthMethod  string `json:"authmethod"`
}

func fatalOrErr(err error) error {
	// it can happen that token gets ivalidated - shutdown in these cases
	// for others it just "flows through"
	if strings.Contains(err.Error(), "invalid token") {
		zap.L().Fatal("EXIT:token: invalid token, restarting: " + err.Error())
		return err
	}
	return err
}

func readConfig(configFilePath string) *hvaultRemoteService {
	data, err := os.ReadFile(configFilePath)
	if err != nil {
		zap.L().Fatal("EXIT:ctx: failed to read vault config file with error: " + err.Error())
	}
	vaultService := &hvaultRemoteService{}
	err = json.Unmarshal(([]byte(data)), &vaultService)
	if err != nil {
		zap.L().Fatal("EXIT:ctx: invalid JSON config file: " + err.Error())
	}
	if vaultService.TransitPath == "" {
		vaultService.TransitPath = "transit"
	}
	return vaultService
}

func setupClient(cfg *api.Config, ns string, method string, roleName string, mountPath string) (*api.Client, api.AuthMethod) {	
	authMethod, err := createAuthMethod(method, roleName, mountPath)
	if err != nil {
		zap.L().Fatal("EXIT:client: failed to create auth method: " + err.Error())
	}
	client, err := api.NewClient(cfg)
	if err != nil {
		zap.L().Fatal("EXIT:client: failed to initialize Vault client with error: " + err.Error())
	}
	client.SetNamespace(ns)
	authInfo, err := client.Auth().Login(context.Background(), authMethod)
	if err != nil {
		zap.L().Fatal("EXIT:authInfo: unable to log in with error:" + err.Error())
	}
	if authInfo == nil {
		zap.L().Fatal("EXIT:authInfo: no auth info was returned after login")
	}
	return client, authMethod
}

func NewVaultClientRemoteService(configFilePath string) (service.Service, error) {
	vaultService := readConfig(configFilePath)
	vaultconfig := api.DefaultConfig()
	vaultconfig.Address = vaultService.Address

	zap.L().Debug("Config loaded:", zap.String("Vault address", vaultService.Address),
		zap.String("Transit key name", vaultService.Transitkey),
		zap.String("Vault role", vaultService.Vaultrole),
		zap.String("Vault namespace", vaultService.Namespace),
		zap.String("Auth method", vaultService.AuthMethod),
		zap.String("Auth mount path", vaultService.AuthPath),
		zap.String("Transit engine mount path", vaultService.TransitPath),
	)
	// setup client with selected auth method
	vaultService.Client, vaultService.ClientAuthMethod = setupClient(vaultconfig,
		vaultService.Namespace,
		vaultService.AuthMethod, 
		vaultService.Vaultrole, 
		vaultService.AuthPath)

	// obtain latest version of the transit key and create a key ID for it
	key, err := vaultService.GetTransitKey(context.Background())
	if err != nil {
		zap.L().Fatal("ERROR:key: unable to find transit key, restarting: " + err.Error())
	}
	vaultService.LatestKeyID = createLatestTransitKeyId(key)
	zap.L().Info("Received key ID on startup: " + vaultService.LatestKeyID)

	// initial token check
	err = vaultService.CheckTokenValidity(context.Background())
	if err != nil {
		zap.L().Fatal("EXIT:token: could not check token validity: " + err.Error())
	}

	return vaultService, nil
}

func (s *hvaultRemoteService) Encrypt(ctx context.Context, uid string, plaintext []byte) (*service.EncryptResponse, error) {
	zap.L().Debug("Received encrypt request with UID: " + uid)
	enresult, err := s.encrypt(ctx, plaintext)
	if err != nil {
		zap.L().Error("enresult: invalid response")
		return nil, errors.New("Invalid response")
	}

	return &service.EncryptResponse{
		Ciphertext: enresult,
		KeyID:      s.LatestKeyID,
		Annotations: map[string][]byte{
			annotationKey: []byte("1"),
		},
	}, nil
}

func (s *hvaultRemoteService) encrypt(ctx context.Context, plaintext []byte) ([]byte, error) {
	enckeypath := fmt.Sprintf("%s/encrypt/%s", s.TransitPath, s.Transitkey)
	encodepayload := map[string]interface{}{
		"plaintext": base64.StdEncoding.EncodeToString(plaintext),
	}
	encrypt, err := retryVaultOp(s, ctx, 3, retrySleep, func()(*api.Secret, error){
		return s.Client.Logical().WriteWithContext(ctx, enckeypath, encodepayload)
	})
	if err != nil {
		zap.L().Error("encrypt: error: " + err.Error())
		return nil, fatalOrErr(err)
	}
	enresult, ok := encrypt.Data["ciphertext"].(string)
	if !ok {
		zap.L().Error("enresult: invalid response")
		return nil, errors.New("Invalid response")
	}
	return []byte(enresult), nil
}

func (s *hvaultRemoteService) Decrypt(ctx context.Context, uid string, req *service.DecryptRequest) ([]byte, error) {
	zap.L().Debug("Received decrypt request with UID: " + uid)
	if len(req.Annotations) != 1 {
		zap.L().Error("len:annotations: " + fmt.Sprintf("%v", req.Annotations))
		return nil, fmt.Errorf("/!\\ invalid annotations")
	}
	if v, ok := req.Annotations[annotationKey]; !ok || string(v) != "1" {
		return nil, fmt.Errorf("/!\\ invalid version in annotations")
	}
	return s.decrypt(ctx, req.Ciphertext)
}

func (s *hvaultRemoteService) decrypt(ctx context.Context, ciphertext []byte) ([]byte, error) {
	decryptkeypath := fmt.Sprintf("%s/decrypt/%s", s.TransitPath, s.Transitkey)
	encryptedPayload := map[string]interface{}{
		"ciphertext": string(ciphertext),
	}
	encryptedResponse, err := retryVaultOp(s, ctx, 3, retrySleep, func()(*api.Secret, error){
		return s.Logical().WriteWithContext(ctx, decryptkeypath, encryptedPayload)
	})
	if err != nil {
		zap.L().Error("encryptedResponse: with error: " + err.Error())
		return nil, fatalOrErr(err)
	}
	response, ok := encryptedResponse.Data["plaintext"].(string)
	if !ok {
		zap.L().Error("response: invalid response")
		return nil, errors.New("response: invalid response")
	}
	decodepayload, err := base64.StdEncoding.DecodeString(response)
	if err != nil {
		zap.L().Error("decodepayload: with error: " + err.Error())
		return nil, err
	}
	return decodepayload, nil
}

func (s *hvaultRemoteService) Status(ctx context.Context) (*service.StatusResponse, error) {
	// get transit key, obtain the latest version of the transit key
	key, err := s.GetTransitKey(ctx)
	if err != nil {
		zap.L().Error("ERROR:key: unable to find transit key: " + err.Error())
		return s.createStatusResponse(healthNOK), err
	}
	// extract the latest and create key id for it
	s.LatestKeyID = createLatestTransitKeyId(key)
	zap.L().Debug("Key ID updated to: " + s.LatestKeyID)
	// do healthcheck
	err = s.Health(ctx)
	if err != nil {
		zap.L().Error("ERROR:Status: unhealthy: " + err.Error())
		return s.createStatusResponse(healthNOK), err
	}
	// all OK
	return s.createStatusResponse(healthOK), nil
}

func (s *hvaultRemoteService) Health(ctx context.Context) error {
	// check if it has valid token lease (Vault)
	err := s.CheckTokenValidity(ctx)
	if err != nil {
		return err
	}
	// check Encryption as Service functionality (transit)
	enc, err := s.encrypt(ctx, []byte(healthy))
	if err != nil {
		zap.L().Error("Health: encrypt failed: " + err.Error())
		return err
	}
	dec, err := s.decrypt(ctx, []byte(enc))
	if err != nil {
		return errors.New("Health: decrypt failed: " + err.Error())
	}
	// decrypted plaintext does not match
	if healthy != string(dec) {
		return errors.New("Health check failed: decrypt does not match")
	}
	zap.L().Info("Health: Health check OK")
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
	// retry read 3x with 150 millisec delay between them
	key, err := retryVaultOp(s, ctx, 3, retrySleep, func()(*api.Secret, error){
		return s.Client.Logical().ReadWithContext(ctx, fmt.Sprintf("%s/keys/%s", s.TransitPath, s.Transitkey))
	})
	if err != nil {
		return nil, fatalOrErr(err)
	}
	zap.L().Debug("Got transit key: " + fmt.Sprintf("%v", map[string]interface{}{
		"latest_version":         key.Data["latest_version"],
		"min_available_version":  key.Data["min_available_version"],
		"min_encryption_version": key.Data["min_encryption_version"],
		"min_decryption_version": key.Data["min_decryption_version"],
		"auto_rotate_period":     key.Data["auto_rotate_period"]}))
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

	token, err := retryVaultOp(s, ctx, 3, retrySleep, func()(*api.Secret, error){
		return s.Client.Logical().ReadWithContext(ctx, path)
	})
	if err != nil {
		return nil, fatalOrErr(err)
	}
	return token, nil
}

func (s *hvaultRemoteService) CheckTokenValidity(ctx context.Context) error {
	token, err := s.GetVaultToken(ctx)
	if err != nil {
		zap.L().Error("Token: could not get token: " + err.Error())
		return err
	}

	creation_ttl, _ := strconv.Atoi(fmt.Sprintf("%s", token.Data["creation_ttl"]))
	ttl, _ := strconv.Atoi(fmt.Sprintf("%s", token.Data["ttl"]))

	zap.L().Debug("Token: " + fmt.Sprintf("%v", map[string]interface{}{
		"creation_ttl":     creation_ttl,
		"issue_time":       token.Data["issue_time"],
		"expire_time":      token.Data["expire_time"],
		"explicit_max_ttl": token.Data["explicit_max_ttl"],
		"ttl":              ttl,
	}))

	if ttl <= 0 || ttl > creation_ttl {
		// token has been tampered with
		// also happens if you've modify role's ttl by hand
		// To wait (return Error) or not to wait (Fatal)?
		zap.L().Fatal("EXIT:token: invalid ttl, re-login needed")
	}
	// renew the token if it reached it's validity periods about 2/3rd
	if float32(ttl) <= float32(creation_ttl)-(float32(creation_ttl)*0.667) {
		// renew the token
		zap.L().Debug("Token near expiry, renewing the token.")
		err = s.RenewOwnToken(ctx, creation_ttl)
		if err != nil {
			zap.L().Error("Token renew failed: " + err.Error())
			return errors.New("Token renew failed.")
		}
		zap.L().Info("Token renew successful.")
		return nil
	}
	// no need for token renew
	zap.L().Debug("No need for token renew.")
	return nil
}

func (s *hvaultRemoteService) RenewOwnToken(ctx context.Context, creation_ttl int) error {
	// renews with the original creation_ttl
	path := fmt.Sprintf("auth/token/renew-self")
	_, err := retryVaultOp(s, ctx, 3, retrySleep, func()(*api.Secret, error){
		return s.Client.Logical().WriteWithContext(ctx, path, 
			map[string]any{"data": map[string]any{
				"ttl":       fmt.Sprintf("%d", creation_ttl),
				"renewable": "true"}})
	})
	if err != nil {
		return fatalOrErr(err)
	}
	return nil
}

func retryVaultOp[T any](s *hvaultRemoteService, ctx context.Context, amount int, sleepTime time.Duration, f func()(T, error)) (result T, err error) {
	// Retries operation f() "amount", times, with "sleepTime" in between them.
	// If operation cannot be performed due to e.g. expired login, try to login in again and retry.
	// Applicable wherever read/write call to Vault is performed.
	for i := 0; i < amount; i++ {
		select {
		case <-ctx.Done():
			return result, ctx.Err()
		default:
			result, err = f()
			if err != nil {
				if strings.Contains(err.Error(), "invalid token") {
					// re-login
					authInfo, err := s.Client.Auth().Login(ctx, s.ClientAuthMethod)
					if err != nil {
						zap.L().Error("Error: Could not relogin: " + err.Error())
					}
					if authInfo == nil {
						zap.L().Error("Error: Relogin received empty auth info")
					}
					// relogin OK
				} // other error that cannot be solved by relogin: try calling f() again
			} else {
				// no error, no need to retry
				zap.L().Debug("Operation succeded on attempt " + fmt.Sprintf("%d", i+1))
				return result, nil
			}
			time.Sleep(sleepTime)
		}
	}
	return result, err
}
