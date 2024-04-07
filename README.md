# kleidi KMS provider plugin for Kubernetes

## Current feature
* KMSv2 
* PKCS#11 interface with [SoftHSM](https://www.opendnssec.org/softhsm/).   
  **Note: it is intended to be used for PoC only, not for production use.**

## Why 1.29 or later?
Stability!   
Any prior release marked KMSv2 as non-stable. Here is the extract from the [Kubernetes documentation](https://kubernetes.io/docs/tasks/administer-cluster/kms-provider/#before-you-begin):  
*The version of Kubernetes that you need depends on which KMS API version you have selected. Kubernetes recommends using KMS v2.*   
* *If you selected KMS API v2, you should use Kubernetes v1.29 (if you are running a different version of Kubernetes that also supports the v2 KMS API, switch to the documentation for that version of Kubernetes).*
* *If you selected KMS API v1 to support clusters before version v1.27 or if you have a legacy KMS plugin that only supports KMS v1, any supported Kubernetes version will work. This API is deprecated as of Kubernetes v1.28. Kubernetes does not recommend the use of this API.*

## Future feature

* production-grade SoftHSM implementation. 
* (v)TPM integration.
* HashiCorp Vault Community Edition/openbao integration. 


# Why a KMS provider plugin for Kubernetes? 

This is related to security exposure and how credential handling practices differ between application and infrastructure management with [physical/virtual] machines and with a container platform like Kubernetes. 

More to be said and understood with [Kubernetes Secrets Handbook](https://www.amazon.com/Kubernetes-Secrets-Handbook-production-grade-management/dp/180512322X)

## Physical/Virtual machine world
The entire IT organization is segmented into knowledge domains such as networking, storage, computing, and applications in the legacy world. 
When an application team asks for a virtual machine:
* The VMware Team has its credentials, which the Linux team cannot access.
* The Linux team will configure, maintain, and support the operating system and not share their credentials with any other team. 
* The application team will deploy their application, which might be connected to a database; the DBA will provide credentials.   
This quick overview can be enriched with all other layers like storage, backup, networking, monitoring, etc.
None will cross-share their credentials.

## Container platform world
Within Kubernetes, the states and configurations of every component, from computing to networking to applications and more, are stored within the ```etcd``` key-value datastore. 

Even if cloud-native applications can interact directly with a KMS provider like Vault, application and platform credentials are still stored within the cluster. This might also include the token to connect with the KMS provider.

All data fields are encoded in base64 but not encrypted. 

## Security exposures

The following diagram takes a 10,000-feet overview to explore the security exposures leading to a potential secret leaking/spilling: 

![kleidi security exposures](docs/images/kledi-security_exposure.drawio.png)

* The secret comes from an external source and needs to be injected.  
* The base64 encoded secret will be ingested via the API server. 
* If a Kubernetes KMS provider plugin exists, the API server encrypts the data field using an envelope encryption scheme. 
* The secret and encrypted data filed will be stored within the ```etcd``` key-value datastore. 
* The ```etcd``` key-value datastore file is saved on a local volume on the control plane node filesystem. 

What are the exposures:
* The secret comes from an external source. It requires a base64-encoded payload. This transformation is a first-level exposure of the data field at file and console levels.
* A common mistake is committing the secret YAML definition to a Git repository. 
* If no KMS provider plugin exists, the API server stores the base64-encoded secret within the ```etcd``` key-value datastore. 
* If a KMS provider plugin exists, the API server encrypts the payload and stores it within the ```etcd``` key-value datastore.
* When using the KMS provider plugin (and for any applications), non-encrypted credentials are stored within Kubernetes to provide access to the KMS provider. 
* The ```etcd``` key-value datastore is stored on the control plane filesystem. Encrypting the filesystem helps secure the datastore file from being read, except if the node has been compromised with root access.
* Lastly, if the API server is compromised, any protective measures are useless since the API server will decrypt secrets for the attacker.  
Thanks to Red Hat colleagues Francois Duthilleul and Frederic Herrmann for spending time analyzing the gaps.

# Implementation

## kleidi v0.1 

kleidi has bootstrapped a code base from the [Kunernetes mock KMS plugin](https://github.com/kubernetes/kms/tree/master/internal/plugins/_mock). This provides a PKCS#11 interface for a local software HSM like [SoftHSM](https://www.opendnssec.org/softhsm/).

The code provides the following:   
* KMSv2 support tested with Kubernetes 1.29 and onwards. 
* PCKS#11 interface to SoftHSM.
* DaemonSet deployment.
* Logging subsystem. 
* Plugin configuration.
* HashiCorp Vault and TPM package module placeholders.

Based on a gRPC architecture requirement from the Kubernetes project, kleidi lives close to the API server on the master node(s).   
kleidi depends on a custom ```initContainer``` to streamline the bootstrap of both SoftHSM and PCKS#11 interface using two volumes:   
* ```/opt/kleidi/``` to store the ```config.json``` 
* ```/var/lib(64)/softhsm/``` to set up the HSM token 

With successful ```initContainer```, the ```kleidi-kms-plugin``` container starts and accesses three volumes:   
* ```/opt/kleidi/``` to access the ```config.json``` 
* ```/var/lib(64)/softhsm/``` to access the token 
* ```/tmp/kleidi``` to create the gRPC socket 

![kleidiv0.1](docs/images/kleidiv0.1.drawio.png)

***This version is a PoC and should never be used in production-grade environments.***

## Deployment

---TODO---
The current implementation has been tested on:   
* Kind
* RKE2 

# kleidi R&D
Considering the security exposures described in this README, an in-platform solution leveraging the (v)TPM chipset is currently designed and tested.

# Origin of kleidi

Initially, [romdalf](https://github.com/romdalf) founded [Trousseau](https://trousseau.io) in 2019 and released a production-grade KMSv1 provider plugin during his tenure at Ondat. 
With the Kubernetes project moving to KMSv2 stable at 1.29 and KMSv1 being deprecated, a decision needed to be made regarding the plugin's rewriting. Considering this technical requirement and Akamai's acquisition of Ondat.io, which sponsored Trousseau's development, the best course of action was to deprecate Trousseau. 

![](https://beezy.dev/images/DALL-E-kleid%C3%AD_comic_strip.png)

