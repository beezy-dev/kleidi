# kleidi KMS Provider Plugin for Kubernetes

Kleidi is based on the [mock KMS plugin](https://github.com/kubernetes/kms/tree/master/internal/plugins/_mock) implemented by the Kubernetes SIG-Auth for API testing purposes.

The current implementation of Kleidi provides the followings features:

* support **only** the KMSv2 marked as stable from Kubernetes version 1.29 and onwards. 
* support of the original PKCS#11 interface backed by backed by [SoftHSM](https://www.opendnssec.org/softhsm/). It is intended to be used for testing only and not for production use.
* support HashiCorp Vault Community Edition. 

