package providers

import (
	"crypto/cipher"
	"fmt"

	"github.com/beezy-dev/kleidi/utils"
	"k8s.io/kms/pkg/service"
)

// func HvaultPlaceholder() {
// 	log.Fatalln("/!\\ implementation in progress - stay tune!")
// }

// var _ service.Service = &hvaultRemoteService{}

type hvaultRemoteService struct {
	keyID string
	aead  cipher.AEAD
}

func NewHVaultRemoteService(configFilePath, provider, keyID string) (service.Service, error) {
	ctx, err := utils.ConfigureFromFile(configFilePath, provider)
	if err != nil {
		return nil, fmt.Errorf("/!\\ %v", err)
	}

	if len(keyID) == 0 {
		return nil, fmt.Errorf("/!\\ invalid keyID")
	}

	remoteService := &hvaultRemoteService{
		keyID: keyID,
	}

	key, err := ctx.FindKey(nil, []byte(keyID))
}
