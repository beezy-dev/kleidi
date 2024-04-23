package providers

import (
	"log"
)

func TmpPlaceholder() {
	log.Fatalln("/!\\ implementation in progress - stay tune!")
}

// var (
// 	// Default SRK handle
// 	srkHandle tpmutil.Handle = 0x81000001

// 	// Default SRK key template
// 	srkTemplate = tpm2.Public{
// 		Type:       tpm2.AlgECC,
// 		NameAlg:    tpm2.AlgSHA256,
// 		Attributes: tpm2.FlagStorageDefault | tpm2.FlagNoDA,
// 		ECCParameters: &tpm2.ECCParams{
// 			Symmetric: &tpm2.SymScheme{
// 				Alg:     tpm2.AlgAES,
// 				KeyBits: 128,
// 				Mode:    tpm2.AlgCFB,
// 			},
// 			CurveID: tpm2.CurveNISTP256,
// 			Point: tpm2.ECPoint{
// 				XRaw: make([]byte, 32),
// 				YRaw: make([]byte, 32),
// 			},
// 		},
// 	}

// 	// Our Key Handle
// 	keyHandle tpmutil.Handle = 0x81010004

// 	// ECC Encrypt/Decrypt key template
// 	eccKeyTemplate = tpm2.Public{
// 		Type:       tpm2.AlgECC,
// 		NameAlg:    tpm2.AlgSHA256,
// 		Attributes: tpm2.FlagStorageDefault & ^tpm2.FlagRestricted,
// 		ECCParameters: &tpm2.ECCParams{
// 			CurveID: tpm2.CurveNISTP256,
// 			Point: tpm2.ECPoint{
// 				XRaw: make([]byte, 32),
// 				YRaw: make([]byte, 32),
// 			},
// 		},
// 	}
// )

// TODO:
// check if a TPM entry exists
func tpmHasKey() {
	// --> TPM entry exists, start service
	// --> not TPM entry, generate and store a key in TPM

}

// start a service
func NewTPMRemoteService() {

	// validate config

	// validate keyID

	// verify key

	// GCM

	// remoteService start

}

//to encrypt/decrypt using key in TPM

// check status

func generateKey() {

	// // use TPM as a source of randomness to generate a 32 bytes block
	// key := make([]byte, 32)

	// aes.BlockSize(32)
	// _, err := rand.Read(key)
	// cipher, err := aes.NewCipher(key)
	// if err != nil {
	// 	fmt.Println(err)
	// }
	// fmt.Printf("%v", cipher)

}

func StoreKeyTPM() {

}
