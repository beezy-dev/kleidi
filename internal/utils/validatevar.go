package utils

import (
	"fmt"
	"net/url"
	"slices"
	"strings"

	"k8s.io/klog/v2"
)

func ValidateListenAddr(listenAddr string) (string, error) {

	// The only valid protocol defined for a Kubernetes KMS plugin is "unix".
	const (
		proto = "unix"
	)

	url, err := url.Parse(listenAddr)
	if err != nil {
		return url.Path, fmt.Errorf("/!\\ invalid listen-addr %q, error: %v", listenAddr, err)
	}

	if len(listenAddr) == 0 {
		return url.Path, fmt.Errorf("/!\\ can not be an empty string")
	}

	if url.Scheme != proto {
		return url.Scheme, fmt.Errorf("/!\\ proto can be different than unix")
	}

	if strings.HasPrefix(url.Path, "/@") {
		return strings.TrimPrefix(url.Path, "/"), nil
	}

	klog.InfoS("ListenAddr set as", "-listen", listenAddr)
	return url.Path, nil
}

func ValidateProvider(providerService string) (string, error) {

	providerServices := []string{"hvault", "softhsm", "tpm"}
	if !slices.Contains(providerServices, providerService) {
		return providerService, fmt.Errorf("/!\\ flag -provider is not supported. Only %v are valid options", providerServices)
	}

	klog.InfoS("Provider set as", "-provider", providerService)
	return providerService, nil
}

func ValidateConfigfile(providerConfigFile string) (string, error) {

	if len(providerConfigFile) == 0 {
		return providerConfigFile, fmt.Errorf("/!\\ can not be an empty string")
	}

	klog.InfoS("ConfigFile set as", "-configfile", providerConfigFile)
	return providerConfigFile, nil

}
