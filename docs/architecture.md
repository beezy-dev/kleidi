# Implementation

## kleidi  
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

## Why 1.29 or later?
***Stability!***   

Any prior release marked KMSv2 as non-stable. Here is the extract from the [Kubernetes documentation](https://kubernetes.io/docs/tasks/administer-cluster/kms-provider/#before-you-begin):  
*The version of Kubernetes that you need depends on which KMS API version you have selected. Kubernetes recommends using KMS v2.*   
* *If you selected KMS API v2, you should use Kubernetes v1.29 (if you are running a different version of Kubernetes that also supports the v2 KMS API, switch to the documentation for that version of Kubernetes).*
* *If you selected KMS API v1 to support clusters before version v1.27 or if you have a legacy KMS plugin that only supports KMS v1, any supported Kubernetes version will work. This API is deprecated as of Kubernetes v1.28. Kubernetes does not recommend the use of this API.*