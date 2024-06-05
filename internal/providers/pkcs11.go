/*
Forked from the KMSv2 mockup example
Source: https://github.com/kubernetes/kms.git
Apache 2.0 License
*/

package providers

import (
	"context"
	"crypto/cipher"
	"crypto/rand"
	"fmt"

	crypot11 "github.com/ThalesIgnite/crypto11"
	"k8s.io/kms/pkg/service"
)

const (
	annotationKey = "v2.kleidi.beezy.dev"
)

var _ service.Service = &pkcs11RemoteService{}

type pkcs11RemoteService struct {
	keyID string
	aead  cipher.AEAD
}

// NewPKCS11RemoteService creates a new PKCS11 remote service with SoftHSMv2 configuration file and keyID
func NewPKCS11RemoteService(configFilePath, keyID string) (service.Service, error) {
	ctx, err := crypot11.ConfigureFromFile(configFilePath)
	if err != nil {
		return nil, fmt.Errorf("/!\\ %v", err)
	}

	if len(keyID) == 0 {
		return nil, fmt.Errorf("/!\\ invalid keyID")
	}

	remoteService := &pkcs11RemoteService{
		keyID: keyID,
	}

	key, err := ctx.FindKey(nil, []byte(keyID))
	if err != nil {
		return nil, err
	}

	if key == nil {
		return nil, fmt.Errorf("/!\\ key not found")
	}

	if remoteService.aead, err = key.NewGCM(); err != nil {
		return nil, err
	}

	return remoteService, nil
}

func (s *pkcs11RemoteService) Encrypt(ctx context.Context, uid string, plaintext []byte) (*service.EncryptResponse, error) {
	nonceSize := s.aead.NonceSize()
	result := make([]byte, nonceSize+s.aead.Overhead()+len(plaintext))

	n, err := rand.Read(result[:nonceSize])
	if err != nil {
		return nil, err
	}

	if n != nonceSize {
		return nil, fmt.Errorf("/!\\ unable to read sufficient random bytes")
	}

	cipherText := s.aead.Seal(result[nonceSize:nonceSize], result[:nonceSize], plaintext, []byte(s.keyID))

	return &service.EncryptResponse{
		Ciphertext: result[:nonceSize+len(cipherText)],
		KeyID:      s.keyID,
		Annotations: map[string][]byte{
			annotationKey: []byte("1"),
		},
	}, nil
}

func (s *pkcs11RemoteService) Decrypt(ctx context.Context, uid string, req *service.DecryptRequest) ([]byte, error) {

	if len(req.Annotations) != 1 {
		return nil, fmt.Errorf("/!\\ invalid annotations")
	}

	if v, ok := req.Annotations[annotationKey]; !ok || string(v) != "1" {
		return nil, fmt.Errorf("/!\\ invalid version in annotations")
	}

	if req.KeyID != s.keyID {
		return nil, fmt.Errorf("/!\\ invalid keyID")
	}

	nonceSize := s.aead.NonceSize()

	data := req.Ciphertext
	if len(data) < nonceSize {
		return nil, fmt.Errorf("/!\\ stored data was shorter than the required size")
	}

	return s.aead.Open(nil, data[:nonceSize], data[nonceSize:], []byte(s.keyID))
}

func (s *pkcs11RemoteService) Status(ctx context.Context) (*service.StatusResponse, error) {
	return &service.StatusResponse{
		Version: "v2",
		Healthz: "ok",
		KeyID:   s.keyID,
	}, nil
}
