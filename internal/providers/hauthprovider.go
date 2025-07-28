package providers

import (
	"errors"
	hvaultapi "github.com/hashicorp/vault/api"
	k8sauth   "github.com/hashicorp/vault/api/auth/kubernetes"
	certauth  "github.com/hashicorp/vault/api/auth/cert"
)

func getK8sAuth(roleName string, mountPath string) (hvaultapi.AuthMethod, error) {
	return k8sauth.NewKubernetesAuth(
		roleName,
		k8sauth.WithMountPath(mountPath))
}

func getCertAuth(roleName string, mountPath string) (hvaultapi.AuthMethod, error) {
	return certauth.NewCertAuth(
		certauth.WithRole(roleName),
		certauth.WithMountPath(mountPath))
}

func createAuthMethod(method string, roleName string, mountPath string) (hvaultapi.AuthMethod, error) {
	switch method {
	case "k8s":
		return getK8sAuth(roleName, mountPath)
	case "cert":
		return getCertAuth(roleName, mountPath)
	default:
		return nil, errors.New("Unsupported auth method: " + method)
	}
}