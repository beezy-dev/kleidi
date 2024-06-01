# kleidi KMS provider plugin for Kubernetes

## Origin of kleidi

<img align="right" src="https://beezy.dev/images/DALL-E-kleid%C3%AD_comic_strip.png" width="25%">

Initially, [romdalf](https://github.com/romdalf) founded [Trousseau](https://trousseau.io) in 2019 and released a production-grade KMSv1 provider plugin during his tenure at Ondat.  

With the Kubernetes project moving to KMSv2 stable at 1.29 and KMSv1 being deprecated since 1.27, a decision needed to be made regarding rewriting the plugin, leading to the creation of kleidi.

The origin is Greek, and the meaning is "key". (Source: [Wikipedia](https://en.wiktionary.org/wiki/%CE%BA%CE%BB%CE%B5%CE%B9%CE%B4%CE%AF))

<br clear="left"/>
<br clear="left"/>

## Current state
* KMSv2 with Kubernetes 1.29 and onwards.
* PKCS#11 interface to [SoftHSM](https://www.opendnssec.org/softhsm/) deployed on the control plane nodes.   
  **Note: it is intended to be used for PoC only, not for production use.**

## Why 1.29 or later?
***Stability!***   

Any prior release marked KMSv2 as non-stable. Here is the extract from the [Kubernetes documentation](https://kubernetes.io/docs/tasks/administer-cluster/kms-provider/#before-you-begin):  
*The version of Kubernetes that you need depends on which KMS API version you have selected. Kubernetes recommends using KMS v2.*   
* *If you selected KMS API v2, you should use Kubernetes v1.29 (if you are running a different version of Kubernetes that also supports the v2 KMS API, switch to the documentation for that version of Kubernetes).*
* *If you selected KMS API v1 to support clusters before version v1.27 or if you have a legacy KMS plugin that only supports KMS v1, any supported Kubernetes version will work. This API is deprecated as of Kubernetes v1.28. Kubernetes does not recommend the use of this API.*

## Future state  
* production-grade SoftHSM implementation. 
* HashiCorp Vault Community Edition/openbao integration.
* (v)TPM integration (see R&D)

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

|Exposure | Risk | Mitigation |
|---------|------|------------|
|The secret comes from an external source. It requires a base64-encoded payload. | This transformation is a first-level exposure of the data field at file and console levels. | Work an injection mechanism from the password manager or KMS |  
| A common mistake is committing the secret YAML definition to a Git repository. | The credentials are exposed for life and will need to be rotated. | Don't include any YAML manifest with sensitive data in a Git repository even for testing purposes. Using a tool like SOPS can help prevent such scenario | 
| If no KMS provider plugin exists. | The API server stores the base64-encoded secret within the ```etcd``` key-value datastore. | Application secrets might benefit from an external KMS. Platform secrets will require a data encryption at rest option provided by Kubernetes. |
| If a KMS provider plugin exists. | The encryption Key or credentials to access the KMS or HSM are exposed in clear text. | Set up a mTLS authentication if possible. |
| The ```etcd``` key-value datastore is stored on the control plane filesystem. | The datastore file can be accessed if the node is compromised. | Encrypting the filesystem helps secure the datastore file from being read, except if the node has been compromised with root access. |
| The API server is the Kubernetes heart and soul. | If the appropriate RBAC or the API server is compromised, all protective measures will be useless since the API server will decrypt all sensitive data fields. | RBAC and masking the API server if possible | 

Thanks to Red Hat colleagues Francois Duthilleul and Frederic Herrmann for spending time analyzing the gaps.

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

***This version is a PoC and should never be used in production-grade environments.***

# Dev/Test Deployments

## SoftHSM

### Deployment
This implementation includes the initialization of a SoftHSM with a PKCS#11 interface on the control plane.    

Notes:   
- At the current stage, only the SoftHSM token (unretrievable) and PKCS#11 libraries are deployed locally to provide:
  - soft dependencies for the API server
  - data persistence for the token
- The ```pod``` definition should only be used with ```kind``` or similar. For production usage, the ```daemonset``` should be considered.

This can be achieved using the ```kind``` configuration for development/testing purposes:

```
kind create cluster --config configuration/k8s/kind/kind-softhsm.yaml
```

Expected output:
``` 
enabling experimental podman provider
Creating cluster "kleidi-softhsm" ...
 ✓ Ensuring node image (kindest/node:v1.29.2) 🖼
 ✓ Preparing nodes 📦 📦  
 ✓ Writing configuration 📜 
 ✓ Starting control-plane 🕹️ 
 ✓ Installing CNI 🔌 
 ✓ Installing StorageClass 💾 
 ✓ Joining worker nodes 🚜 
Set kubectl context to "kind-kleidi-softhsm"
You can now use your cluster with:

kubectl cluster-info --context kind-kleidi-softhsm

Have a nice day! 👋
```

This configuration is available in ```configuration/k8s/kind/kind-softhsm.yaml```:

```YAML
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
name: kleidi-softhsm
nodes:
- role: control-plane
  image: kindest/node:v1.29.2@sha256:51a1434a5397193442f0be2a297b488b6c919ce8a3931be0ce822606ea5ca245
  extraMounts:
    - containerPath: /etc/kubernetes/encryption-config.yaml
      hostPath: configuration/k8s/encryption/encryption-config.yaml
      readOnly: true
      propagation: None 
    - containerPath: /etc/kubernetes/manifests/kube-kms.yaml
      hostPath: configuration/k8s/deploy/pod-kleidi-kms.yaml
      readOnly: true
      propagation: None
    - containerPath: /opt/softhsm/config.json
      hostPath: configuration/softhsm/config.json
      readOnly: true
      propagation: None
  kubeadmConfigPatches:
  - |
    kind: ClusterConfiguration
    apiServer:
      extraArgs:
        encryption-provider-config: "/etc/kubernetes/encryption-config.yaml"
        encryption-provider-config-automatic-reload: "true"
        v: "5"
      extraVolumes:
        - name: encryption-config
          hostPath: /etc/kubernetes/encryption-config.yaml
          mountPath: /etc/kubernetes/encryption-config.yaml
          readOnly: true
          pathType: File
        - name: softhsm
          hostPath: /opt/softhsm/config.json
          mountPath: /opt/softhsm/config.json
          readOnly: false
          pathType: File
        - name: kleidi-socket
          hostPath: /tmp/kleidi
          mountPath: /tmp/kleidi
      scheduler:
        extraArgs:
          v: "5"
      controllerManager:
          v: "5"
- role: worker
  image: kindest/node:v1.29.2@sha256:51a1434a5397193442f0be2a297b488b6c919ce8a3931be0ce822606ea5ca245
```

At the bootstrap, a kleidi ```pod``` will be started to provide the API server with access to the gRPC socket in ```/tmp/kleidi```. If the ```pod``` creation fails, the API server will fail to start, along with the cluster create task. 

The kleidi ```pod``` definition available in ```configuration/k8s/deploy/softhsm-pod-kleidi-kms.yaml```: 
```YAML
apiVersion: v1
kind: Pod
metadata:
  name: kleidi-kms-plugin
  namespace: kube-system
  labels:
    tier: control-plane
    component: kleidi-kms-plugin
spec:
  hostNetwork: true
  initContainers:
  - args:
    - |
      #!/bin/sh
      set -e
      set -x

      # if token exists, skip initialization
      if [ $(ls -1 /var/lib/softhsm/tokens | wc -l) -ge 1 ]; then
        echo "Skipping initialization of softhsm"
        exit 0
      fi

      mkdir -p /var/lib/softhsm/tokens
      
      TOKEN_LABEL=$(jq -r '.tokenLabel' /opt/softhsm/config.json)
      PIN=$(jq -r '.pin' /opt/softhsm/config.json)
      MODULE_PATH=$(jq -r '.path' /opt/softhsm/config.json)

      softhsm2-util --init-token --free --label $TOKEN_LABEL --pin $PIN --so-pin $PIN
      pkcs11-tool --module $MODULE_PATH --keygen --key-type aes:32 --pin $PIN --token-label $TOKEN_LABEL --label kleidi-kms-plugin

      softhsm2-util --show-slots

      ls -al /var/lib/softhsm/tokens
    command:
    - /bin/sh
    - -c
    image: ghcr.io/beezy-dev/kleidi-kms-init:latest
    imagePullPolicy: Always
    name: kleidi-kms-init
    volumeMounts:
    - mountPath: /var/lib/softhsm/tokens
      name: softhsm-tokens
    - mountPath: /opt/softhsm/config.json
      name: softhsm-config
  containers:
    - name: kleidi-kms-plugin
      image: ghcr.io/beezy-dev/kleidi-kms-plugin:latest
      imagePullPolicy: Always
      resources:
        limites:
          cpu: 300m
          memory: 256Mi
      volumeMounts:
        - name: sock
          mountPath: /tmp/kleidi
        - name: softhsm-config
          mountPath: /opt/softhsm/config.json
        - name: softhsm-tokens
          mountPath: /var/lib/softhsm/tokens
  volumes:
    - name: sock
      hostPath:
        path: /tmp/kleidi
        type: DirectoryOrCreate
    - name: softhsm-config
      hostPath:
        path: /opt/softhsm/config.json
        type: File
    - name: softhsm-tokens
      hostPath:
        path: /var/lib/softhsm/tokens
        type: DirectoryOrCreate
```
 

Note that the successful creation of the ```kind``` equals the successful deployment of ```kleidi``` as we can verified this with the following command:

```
kubectl get all -A
```

Expected output
```
NAMESPACE            NAME                                                       READY   STATUS    RESTARTS      AGE
kube-system          pod/coredns-76f75df574-fdwx5                               1/1     Running   0             63s
kube-system          pod/coredns-76f75df574-xjw7k                               1/1     Running   0             63s
kube-system          pod/etcd-kleidi-softhsm-control-plane                      1/1     Running   0             106s
kube-system          pod/kindnet-7dqdj                                          1/1     Running   0             57s
kube-system          pod/kindnet-dv2pl                                          1/1     Running   0             63s
kube-system          pod/kleidi-kms-plugin-kleidi-softhsm-control-plane         1/1     Running   0             105s
kube-system          pod/kube-apiserver-kleidi-softhsm-control-plane            1/1     Running   0             104s
kube-system          pod/kube-controller-manager-kleidi-softhsm-control-plane   1/1     Running   1 (97s ago)   104s
kube-system          pod/kube-proxy-ds4pp                                       1/1     Running   0             57s
kube-system          pod/kube-proxy-qf48v                                       1/1     Running   0             63s
kube-system          pod/kube-scheduler-kleidi-softhsm-control-plane            1/1     Running   0             105s
local-path-storage   pod/local-path-provisioner-7577fdbbfb-7r9lk                1/1     Running   0             63s

NAMESPACE     NAME                 TYPE        CLUSTER-IP   EXTERNAL-IP   PORT(S)                  AGE
default       service/kubernetes   ClusterIP   10.96.0.1    <none>        443/TCP                  91s
kube-system   service/kube-dns     ClusterIP   10.96.0.10   <none>        53/UDP,53/TCP,9153/TCP   89s

NAMESPACE     NAME                        DESIRED   CURRENT   READY   UP-TO-DATE   AVAILABLE   NODE SELECTOR            AGE
kube-system   daemonset.apps/kindnet      2         2         2       2            2           kubernetes.io/os=linux   88s
kube-system   daemonset.apps/kube-proxy   2         2         2       2            2           kubernetes.io/os=linux   89s

NAMESPACE            NAME                                     READY   UP-TO-DATE   AVAILABLE   AGE
kube-system          deployment.apps/coredns                  2/2     2            2           89s
local-path-storage   deployment.apps/local-path-provisioner   1/1     1            1           88s

NAMESPACE            NAME                                                DESIRED   CURRENT   READY   AGE
kube-system          replicaset.apps/coredns-76f75df574                  2         2         2       63s
local-path-storage   replicaset.apps/local-path-provisioner-7577fdbbfb   1         1         1       63s
```

### Encryption/Decryption Test
A test secret can be created using the following command:

```
kubectl create secret generic encrypted-secret -n default --from-literal=mykey=mydata
```
Expected output:
```
secret/encrypted-secret created
```

The payload encryption can be verified with the following command:

``` 
kubectl -n kube-system exec etcd-kleidi-softhsm-control-plane -- sh -c "ETCDCTL_ENDPOINTS='https://127.0.0.1:2379' ETCDCTL_CACERT='/etc/kubernetes/pki/etcd/ca.crt' ETCDCTL_CERT='/etc/kubernetes/pki/etcd/server.crt' ETCDCTL_KEY='/etc/kubernetes/pki/etcd/server.key' ETCDCTL_API=3 etcdctl get /registry/secrets/default/encrypted-secret" | hexdump -C
```

Expected output:

```
00000000  2f 72 65 67 69 73 74 72  79 2f 73 65 63 72 65 74  |/registry/secret|
00000010  73 2f 64 65 66 61 75 6c  74 2f 65 6e 63 72 79 70  |s/default/encryp|
00000020  74 65 64 2d 73 65 63 72  65 74 0a 6b 38 73 3a 65  |ted-secret.k8s:e|
00000030  6e 63 3a 6b 6d 73 3a 76  32 3a 6b 6c 65 69 64 69  |nc:kms:v2:kleidi|
00000040  2d 6b 6d 73 2d 70 6c 75  67 69 6e 3a 0a a9 02 83  |-kms-plugin:....|
00000050  f0 a4 9c 7e b2 ea 23 c4  55 c5 ff 3e 34 42 56 9b  |...~..#.U..>4BV.|
00000060  ba f9 6e 52 81 46 39 e8  46 1f 1e 90 b8 11 88 bf  |..nR.F9.F.......|
00000070  dc 11 2c 57 25 73 72 c5  22 19 3d e1 51 51 c4 90  |..,W%sr.".=.QQ..|
00000080  3b 2a 3e 96 71 f1 6a 40  7b 49 dd c8 ab 9f 67 6a  |;*>.q.j@{I....gj|
00000090  ab 02 a8 aa d1 29 d5 ec  e6 9d 2f 1a bc e5 a8 53  |.....)..../....S|
000000a0  6d c3 fc 77 f1 ce 4f 86  2d 8d 25 13 ba f7 13 4c  |m..w..O.-.%....L|
000000b0  b2 7a e8 67 3b ef 18 cf  75 82 7d bf 0d 66 31 9c  |.z.g;...u.}..f1.|
000000c0  29 f6 16 43 ac 45 c4 b7  f3 fd f7 39 cc d8 48 0d  |)..C.E.....9..H.|
000000d0  92 c7 cd d2 3d 91 69 6f  a7 01 4f b5 0f f6 2f 6d  |....=.io..O.../m|
000000e0  87 e5 bd 11 64 f2 1b 7c  bd c0 c9 39 af 92 d3 c8  |....d..|...9....|
000000f0  a6 24 f7 f5 84 a2 bb ba  35 cc e8 4d d6 18 e3 aa  |.$......5..M....|
00000100  a0 69 f8 ce d6 d2 62 cc  2d da d9 9d 59 5d 88 04  |.i....b.-...Y]..|
00000110  36 18 49 40 97 18 e0 a1  98 85 0e 03 94 b2 d7 a1  |6.I@............|
00000120  c7 22 5b 48 59 57 64 f1  0f d6 fa fc df a6 74 23  |."[HYWd.......t#|
00000130  c6 e5 22 64 31 71 0b c0  0d d7 16 88 63 20 c9 ad  |.."d1q......c ..|
00000140  42 8a 06 98 99 82 47 c4  c4 b5 2e c8 f5 48 5e 5c  |B.....G......H^\|
00000150  be 8a 82 a7 1d c8 38 cc  ca 85 a4 56 81 b5 5a db  |......8....V..Z.|
00000160  6d 78 9f 9c b6 33 de 82  b4 ed bd 1a 1e a6 9e fd  |mx...3..........|
00000170  b6 48 6a 57 4c 64 34 66  12 11 6b 6c 65 69 64 69  |.HjWLd4f..kleidi|
00000180  2d 6b 6d 73 2d 70 6c 75  67 69 6e 1a 40 92 aa b3  |-kms-plugin.@...|
00000190  a4 20 fe b6 47 e0 14 96  90 a6 47 8d 6c 05 c1 1e  |. ..G.....G.l...|
000001a0  d5 fe fb 8a 53 fb 9b 82  9e 0b 4c fe 1e 69 39 08  |....S.....L..i9.|
000001b0  10 8e a3 bd 57 36 18 d2  40 d6 24 03 5d d9 4c ef  |....W6..@.$.].L.|
000001c0  67 f8 a5 a5 d1 45 13 54  fe 97 1f ce 48 22 21 0a  |g....E.T....H"!.|
000001d0  1c 76 65 72 73 69 6f 6e  2e 65 6e 63 72 79 70 74  |.version.encrypt|
000001e0  69 6f 6e 2e 72 65 6d 6f  74 65 2e 69 6f 12 01 31  |ion.remote.io..1|
000001f0  28 01 0a                                          |(..|
000001f3
```

**The above extract shows an encrypted payload with the header ```enc:kms:v2:kleidi-kms-plugin:```.**

## HashiCorp Vault/OpenBao

### Deployment

This implementation includes the initialization of an external HashiCorp Vault with a Transit Key Engine.     
Download the HashiCorp Vault binary or install it with ```brew```, then run the following command:

```
vault server -dev -dev-root-token-id=kleidi-demo -config=configuration/vault/vault-config.json
```
Expected output:
```
...
2024-05-01T10:39:48.750+0200 [INFO]  core: successful mount: namespace="" path=secret/ type=kv version=""
WARNING! dev mode is enabled! In this mode, Vault runs entirely in-memory
and starts unsealed with a single unseal key. The root token is already
authenticated to the CLI, so you can immediately begin using Vault.

You may need to set the following environment variables:

    $ export VAULT_ADDR='http://127.0.0.1:8200'

The unseal key and root token are displayed below in case you want to
seal/unseal the Vault or re-authenticate.

Unseal Key: Ps0mP3INZPeExviBYRy6ZxPhAMEOlPYJNQEEGJzi7sQ=
Root Token: kleidi-demo

Development mode should NOT be used in production installations!

2024-05-01T10:45:21.061+0200 [INFO]  core: successful mount: namespace="" path=transit/ type=transit version=""
```

This start a vault instance with the root token ```kleid-demo``` and storing data on the local filesystem in ```/tmp/vault``` as defined in the configuration file ```configuration/vault/vault-config.json```:

```JSON
{
  "backend": {
    "file": {
      "path": "/tmp/vault/data"
    }
  }
}
```

Then export the Vault address (if not, it will default and fail on HTTPS):
```
export VAULT_ADDR='http://127.0.0.1:8200'
```

Check if the ```vault``` CLI can access the Vault service:
```
vault status
``` 

Expected Output:
```
Key             Value
---             -----
Seal Type       shamir
Initialized     true
Sealed          false
Total Shares    1
Threshold       1
Version         1.14.4
Build Date      2023-09-22T21:29:05Z
Storage Type    file
Cluster Name    vault-cluster-f4e224eb
Cluster ID      e2706241-a8e3-e7f8-f4ca-2f2f77bb7c60
HA Enabled      false
```

Enable the Vault Transit Engine
```
vault secrets enable transit
```
Expected Output:
```
Success! Enabled the transit secrets engine at: transit/
```

Configure a key for kleidi
```
vault write -f transit/keys/kleidi
```

Expected Output:
```
Key                       Value
---                       -----
allow_plaintext_backup    false
auto_rotate_period        0s
deletion_allowed          false
derived                   false
exportable                false
imported_key              false
keys                      map[1:1714553158]
latest_version            1
min_available_version     0
min_decryption_version    1
min_encryption_version    0
name                      kleidi
supports_decryption       true
supports_derivation       true
supports_encryption       true
supports_signing          false
type                      aes256-gcm96
```

Create an ACL policy to control access the engine:
```
vault policy write kleidi configuration/vault/vault-policy.hcl 
```
Expected Output:
```
Success! Uploaded policy: kleidi
```

The policy is available in ```configuration/vault/vault-policy.hcl```:
```hcl
path "transit/encrypt/kleidi" {
   capabilities = [ "update" ]
}

path "transit/decrypt/kleidi" {
   capabilities = [ "update" ]
}

path "transit/keys/kleidi" {
   capabilities = [ "read" ]
}

path "auth/token/lookup-self" {
    capabilities = ["read"]
}
```

At this stage, we have a basic HashiCorp Vault dev/test environment and we can deploy a ```kind``` cluster:

```
kind create cluster --config configuration/k8s/kind/kind-vault.yaml
```

Expected output:
``` 
enabling experimental podman provider
Creating cluster "kleidi-vault" ...
 ✓ Ensuring node image (kindest/node:v1.29.2) 🖼
 ✓ Preparing nodes 📦 📦  
 ✓ Writing configuration 📜 
 ✓ Starting control-plane 🕹️ 
 ✓ Installing CNI 🔌 
 ✓ Installing StorageClass 💾 
 ✓ Joining worker nodes 🚜 
Set kubectl context to "kind-kleidi-vault"
You can now use your cluster with:

kubectl cluster-info --context kind-kleidi-vault

Thanks for using kind! 😊
``` 

This configuration is available in ```configuration/k8s/kind/kind-vault.yaml```:

```YAML
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
name: kleidi-vault
nodes:
- role: control-plane
  image: kindest/node:v1.29.2@sha256:51a1434a5397193442f0be2a297b488b6c919ce8a3931be0ce822606ea5ca245
  extraMounts:
    - containerPath: /etc/kubernetes/encryption-config.yaml
      hostPath: configuration/k8s/encryption/vault-encryption-config.yaml
      readOnly: true
      propagation: None 
    - containerPath: /etc/kubernetes/manifests/kube-kms.yaml
      hostPath: configuration/k8s/deploy/vault-pod-kleidi-kms.yaml
      readOnly: true
      propagation: None
  kubeadmConfigPatches:
  - |
    kind: ClusterConfiguration
    apiServer:
      extraArgs:
        encryption-provider-config: "/etc/kubernetes/encryption-config.yaml"
        encryption-provider-config-automatic-reload: "true"
        v: "5"
      extraVolumes:
        - name: encryption-config
          hostPath: /etc/kubernetes/encryption-config.yaml
          mountPath: /etc/kubernetes/encryption-config.yaml
          readOnly: true
          pathType: File
        - name: kleidi-socket
          hostPath: /tmp/kleidi
          mountPath: /tmp/kleidi
      scheduler:
        extraArgs:
          v: "5"
      controllerManager:
          v: "5"
- role: worker
  image: kindest/node:v1.29.2@sha256:51a1434a5397193442f0be2a297b488b6c919ce8a3931be0ce822606ea5ca245
```

At the bootstrap, a kleidi ```pod``` will be started to provide the API server with access to the gRPC socket in ```/tmp/kleidi```. Due to extra configuration to get the ```ServiceAccount``` to connect to HashiCorp, this ```pod``` will fail. A reconciliation loop will be implemented in the future.

Create the ```ServiceAccount``` including its token and RBAC for Kleidi and Vault:
```
kubectl apply -f configuration/k8s/deploy/vault-sa.yaml 
```
Expected output:
```
serviceaccount/kleidi-vault-auth created
secret/kleidi-vault-auth created
clusterrolebinding.rbac.authorization.k8s.io/role-tokenreview-binding created
```
 
The ```ServiceAccount``` definition from ```configuration/k8s/deploy/vault-sa.yaml```: 
```YAML
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: kleidi-vault-auth
  namespace: kube-system

---
apiVersion: v1
kind: Secret
metadata:
  namespace: kube-system
  name: kleidi-vault-auth
  annotations:
    kubernetes.io/service-account.name: "kleidi-vault-auth"
type: kubernetes.io/service-account-token

---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
   name: role-tokenreview-binding
   namespace: kube-system
roleRef:
   apiGroup: rbac.authorization.k8s.io
   kind: ClusterRole
   name: system:auth-delegator
subjects:
- kind: ServiceAccount
  name: kleidi-vault-auth
  namespace: kube-system
```

Pass the token to HashiCorp Vault using the Vault Kubernetes Auth mechanism:

```
vault auth enable kubernetes
```
Expected output:
```
Success! Enabled kubernetes auth method at: kubernetes/
```

Exporting both the token and the cluster root certificate:
```
token=$(kubectl get secret -n kube-system kleidi-vault-auth -o go-template='{{ .data.token }}' | base64 --decode)
cert=$(kubectl get cm kube-root-ca.crt -o jsonpath="{['data']['ca\.crt']}")
k8shost=$(kubectl config view --raw --minify --flatten --output 'jsonpath={.clusters[].cluster.server}')
```

Injecting the token and certificate to connect to HashiCorp Vault and the Transit engine:
```
vault write auth/kubernetes/config token_reviewer_jwt="${token}" kubernetes_host="${k8shost}" kubernetes_ca_cert="${cert}"
```
Expected output:
```
Success! Data written to: auth/kubernetes/config
```

Link the HashiCorp Vault policy with the ```ServiceAccount```:
``` 
vault write auth/kubernetes/role/kleidi bound_service_account_names=kleidi-vault-auth bound_service_account_namespaces=kube-system policies=kleidi ttl=24h
```
Expected output:
```
Success! Data written to: auth/kubernetes/role/kleidi
```

# Deployment

---TODO---
The current implementation has been tested on:   
* Kind
* RKE2 
Considering this technical requirement and Akamai's acquisition of Ondat.io, which sponsored Trousseau's development, the best course of action was to deprecate Trousseau. 

## SoftHSM

## HashiCorp Vault/openbao


## Todo Delinea/Thycotic integration 
