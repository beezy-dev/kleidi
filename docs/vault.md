# HashiCorp Vault Implementation

## Automated deployment

The folder ```scripts/prd/vault``` includes a script ```env4vault.sh``` leveraging ```Podman``` and ```Kind``` to:
- clean any previous instances of a previous deployment
- deploy:
  - a HashiCorp Vault dev instance
  - a Kind instance running Kubernetes v1.29.2
  - the latest release of Kleidi
- configure:
  - the HashiCorp Vault kleidi and Kubernetes Auth configuration 
  - a kleidi-kms-plugin system critical pod
  - a kube-api restart with the ```encryption-config.yaml```configuration 
- validate:
  - by creating 1001 pre-deployment secrets
  - by creating 1001 post-deployment secrets resulting in encrypted data in etcd
  - by replacing the pre-deployment secrets with their encrypted version in etcd
  - by rotating the Vault key
  - by replacing 1001 pre-deployment encrypted secrets with the rotated key
- If any of the above fails, the script will exit with the related error.

This requires to have the following install:
- Podman
- Kind 
- HashiCorp Vault CLI 
- kubectl 

## Manual deployment

This implementation includes the initialization of an external HashiCorp Vault with a Transit Key Engine.     
Download the HashiCorp Vault binary or install it with ```brew```.

Run the following command to start of dev/test instance:
```
vault server -dev -dev-root-token-id=kleidi-demo --dev-listen-address=0.0.0.0:8200
```
Expected output:
```
...
2024-05-01T10:39:48.750+0200 [INFO]  core: successful mount: namespace="" path=secret/ type=kv version=""
WARNING! dev mode is enabled! In this mode, Vault runs entirely in-memory
and starts unsealed with a single unseal key. The root token is already
authenticated to the CLI, so you can immediately begin using Vault.

You may need to set the following environment variables:

    $ export VAULT_ADDR='http://0.0.0.0:8200'

The unseal key and root token are displayed below in case you want to
seal/unseal the Vault or re-authenticate.

Unseal Key: Ps0mP3INZPeExviBYRy6ZxPhAMEOlPYJNQEEGJzi7sQ=
Root Token: kleidi-demo

Development mode should NOT be used in production installations!

2024-05-01T10:45:21.061+0200 [INFO]  core: successful mount: namespace="" path=transit/ type=transit version=""
```

Then export the Vault address (if not, it will default and fail on HTTPS):
```
export VAULT_ADDR="http://0.0.0.0:8200"
export VAULT_TOKEN="kleidi-demo"
export VAULT_SKIP_VERIFY="true"
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

Create an ACL policy to control access to the Vault transit key engine:
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

path "auth/token/renew-self" {
    capabilities = ["update"]
}
```

## Kind Deployment

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
    - containerPath: /opt/kleidi/config.json
      hostPath: configuration/kleidi/vault-config.json
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
        - name: kleidi-config
          hostPath: /opt/kleidi/config.json
          mountPath: /opt/kleidi/config.json
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
```

To validate there is no encryption, create a pre-deployment secret:
```
kubectl create secret generic prekleidi-secret -n default --from-literal=mykey=mydata
```
Expected output:
```
secret/prekleidi-secret created
```
```
kubectl -n kube-system exec etcd-kleidi-vault-control-plane -- sh -c "ETCDCTL_ENDPOINTS='https://127.0.0.1:2379' ETCDCTL_CACERT='/etc/kubernetes/pki/etcd/ca.crt' ETCDCTL_CERT='/etc/kubernetes/pki/etcd/server.crt' ETCDCTL_KEY='/etc/kubernetes/pki/etcd/server.key' ETCDCTL_API=3 etcdctl get /registry/secrets/default/prekleidi-secret" | hexdump -C
```
Expected output:
```
00000000  2f 72 65 67 69 73 74 72  79 2f 73 65 63 72 65 74  |/registry/secret|
00000010  73 2f 64 65 66 61 75 6c  74 2f 70 72 65 6b 6c 65  |s/default/prekle|
00000020  69 64 69 2d 73 65 63 72  65 74 0a 6b 38 73 00 0a  |idi-secret.k8s..|
00000030  0c 0a 02 76 31 12 06 53  65 63 72 65 74 12 d4 01  |...v1..Secret...|
00000040  0a b8 01 0a 10 70 72 65  6b 6c 65 69 64 69 2d 73  |.....prekleidi-s|
00000050  65 63 72 65 74 12 00 1a  07 64 65 66 61 75 6c 74  |ecret....default|
00000060  22 00 2a 24 61 35 36 37  36 66 35 62 2d 35 66 31  |".*$a5676f5b-5f1|
00000070  33 2d 34 34 63 66 2d 61  66 63 37 2d 39 37 35 30  |3-44cf-afc7-9750|
00000080  31 30 31 31 30 36 35 31  32 00 38 00 42 08 08 93  |101106512.8.B...|
00000090  f5 80 b3 06 10 00 8a 01  62 0a 0e 6b 75 62 65 63  |........b..kubec|
000000a0  74 6c 2d 63 72 65 61 74  65 12 06 55 70 64 61 74  |tl-create..Updat|
000000b0  65 1a 02 76 31 22 08 08  93 f5 80 b3 06 10 00 32  |e..v1".........2|
000000c0  08 46 69 65 6c 64 73 56  31 3a 2e 0a 2c 7b 22 66  |.FieldsV1:..,{"f|
000000d0  3a 64 61 74 61 22 3a 7b  22 2e 22 3a 7b 7d 2c 22  |:data":{".":{},"|
000000e0  66 3a 6d 79 6b 65 79 22  3a 7b 7d 7d 2c 22 66 3a  |f:mykey":{}},"f:|
000000f0  74 79 70 65 22 3a 7b 7d  7d 42 00 12 0f 0a 05 6d  |type":{}}B.....m|
00000100  79 6b 65 79 12 06 6d 79  64 61 74 61 1a 06 4f 70  |ykey..mydata..Op|
00000110  61 71 75 65 1a 00 22 00  0a                       |aque.."..|
00000119
```

## kleidi Deployment

To provide secure connectivity between "kleidi" running on Kubernetes and "vault", a "ServiceAccount" with a token and RBAC is configured. 

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

***Warning*** 
The k8shost variable might output ```127.0.0.1```, resulting in a dial-back failure from Vault when verifying the Kubernetes authentication via the provided certificate. The correct value should be an FQDN, the IP address, or ``` Kubernetes.default.svc.cluster.local```. In the above case ```k8shost=https://kubernetes.default.svc.cluster.local:port``` is a valide option. 

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

Now, it's time to deploy ```kleidi```'s pod:

```
kubectl apply -f configuration/k8s/deploy/vault-pod-kleidi-kms.yaml
```
Expected output:
```
pod/kleidi-kms-plugin created
```

The definition from ```configuration/k8s/deploy/vault-pod-kleidi-kms.yaml```: 
```YAML
---
apiVersion: v1
kind: Pod
metadata:
  name: kleidi-kms-plugin
  namespace: kube-system
  labels:
    tier: control-plane
    component: kleidi-kms-plugin
spec:
  serviceAccount: kleidi-vault-auth
  hostNetwork: true
  containers:
    - name: kleidi-kms-plugin
      image: ghcr.io/beezy-dev/kleidi-kms-plugin:vault-781c292
      imagePullPolicy: Always
      args:
        - -provider=hvault
      resources:
        limits:
          cpu: 300m
          memory: 256Mi
      volumeMounts:
        - name: token
          mountPath: /var/run/secrets/kubernetes.io/serviceaccount/
        - name: sock
          mountPath: /tmp/kleidi
        - name: kleidi-config
          mountPath: /opt/kleidi/config.json
  volumes:
    - name: token
      projected:
        sources:
        - serviceAccountToken:
            path: token
            expirationSeconds: 7200
            audience: vault
    - name: sock
      hostPath:
        path: /tmp/kleidi
        type: DirectoryOrCreate
    - name: kleidi-config
      hostPath:
        path: /opt/kleidi/config.json
        type: File
```

***NOTE***
The above provides the last released images. Please don't change it. 

Then modify the original ```vault-encryption-config.yaml```:
```YAML
apiVersion: apiserver.config.k8s.io/v1
kind: EncryptionConfiguration
resources:
  - resources:
      - secrets
      - configmaps
    providers: 
      - identity: {}
```
with the following definition:
```YAML
apiVersion: apiserver.config.k8s.io/v1
kind: EncryptionConfiguration
resources:
  - resources:
      - secrets
      - configmaps
    providers: 
      - kms:
          apiVersion: v2
          name: kleidi-kms-plugin
          endpoint: unix:///tmp/kleidi/kleidi-kms-plugin.socket
          timeout: 5s    
      - identity: {}
``` 

This should trigger a restart of the Kubernetes API server.

## Encryption/Decryption Test

To validate there is now encryption, create a post-deployment secret:
```
kubectl create secret generic postkleidi -n default --from-literal=mykey=mydata
```
Expected output:
```
secret/postkleidi created
```
```
kubectl -n kube-system exec etcd-kleidi-vault-prd-control-plane -- sh -c "ETCDCTL_ENDPOINTS='https://127.0.0.1:2379' ETCDCTL_CACERT='/etc/kubernetes/pki/etcd/ca.crt' ETCDCTL_CERT='/etc/kubernetes/pki/etcd/server.crt' ETCDCTL_KEY='/etc/kubernetes/pki/etcd/server.key' ETCDCTL_API=3 etcdctl get /registry/secrets/default/postkleidi" | hexdump -C 
```
Expected output:
```
00000000  2f 72 65 67 69 73 74 72  79 2f 73 65 63 72 65 74  |/registry/secret|
00000010  73 2f 64 65 66 61 75 6c  74 2f 70 6f 73 74 6b 6c  |s/default/postkl|
00000020  65 69 64 69 0a 6b 38 73  3a 65 6e 63 3a 6b 6d 73  |eidi.k8s:enc:kms|
00000030  3a 76 32 3a 6b 6c 65 69  64 69 2d 6b 6d 73 2d 70  |:v2:kleidi-kms-p|
00000040  6c 75 67 69 6e 3a 0a a3  02 b0 90 f1 11 d7 38 da  |lugin:........8.|
00000050  dd 7e 50 86 52 6e 88 fe  46 78 9d 76 22 d8 f7 0b  |.~P.Rn..Fx.v"...|
00000060  f4 96 54 ad ba 9c 45 59  5f 39 be ee e2 14 83 18  |..T...EY_9......|
00000070  78 b3 d5 e1 d0 4f 9c 9a  47 2d 19 e6 56 93 26 82  |x....O..G-..V.&.|
00000080  8b b6 c4 eb ba f5 d0 b1  6b 22 88 55 75 99 25 4c  |........k".Uu.%L|
00000090  0e 43 45 87 a4 70 78 b4  26 15 ed c2 6e ad 03 c0  |.CE..px.&...n...|
000000a0  8b 10 56 05 a4 61 c0 41  d5 f9 1b 8b cd 27 d6 32  |..V..a.A.....'.2|
000000b0  71 e7 7c e0 87 fa c8 34  2c 27 26 21 68 a8 e0 0c  |q.|....4,'&!h...|
000000c0  70 5c 7b e1 5b 9f 4d ec  b4 b0 7b ce 01 d2 8f 80  |p\{.[.M...{.....|
000000d0  25 84 77 78 0c 21 73 48  dc 7c 50 66 6e 00 8b e0  |%.wx.!sH.|Pfn...|
000000e0  08 8f 5d 6c d6 2c 7e 46  e4 cb f9 6c f5 d8 72 00  |..]l.,~F...l..r.|
000000f0  44 dc 23 3f 6d cf 2e 38  b7 03 bd 03 54 30 3b a7  |D.#?m..8....T0;.|
00000100  ba ed 1b 5e 42 c1 47 10  68 79 87 64 31 43 73 87  |...^B.G.hy.d1Cs.|
00000110  e1 c4 ce a0 bc 5c 15 ae  b3 30 42 bf f5 fb b8 bc  |.....\...0B.....|
00000120  b2 0a bc 38 29 65 e3 8d  81 23 db 92 38 c2 e5 cb  |...8)e...#..8...|
00000130  4c 2d 24 3e df ba e6 01  ba 11 cc 16 17 0a 10 bb  |L-$>............|
00000140  98 2f 53 90 4a 1a 9e 90  9d 4c a0 34 19 04 91 9c  |./S.J....L.4....|
00000150  22 d4 ac 1d 14 01 1b 45  2f d4 ed e0 73 b6 cf a9  |"......E/...s...|
00000160  43 68 1f a3 5c 56 c6 5d  51 5c 75 4d 12 1e 6b 6c  |Ch..\V.]Q\uM..kl|
00000170  65 69 64 69 2d 6b 6d 73  2d 70 6c 75 67 69 6e 5f  |eidi-kms-plugin_|
00000180  31 5f 31 37 32 37 31 38  38 31 33 39 1a 59 76 61  |1_1727188139.Yva|
00000190  75 6c 74 3a 76 31 3a 36  42 7a 71 53 4d 50 2f 77  |ult:v1:6BzqSMP/w|
000001a0  4b 61 62 4e 6c 79 6d 53  72 59 5a 68 45 7a 55 6c  |KabNlymSrYZhEzUl|
000001b0  31 73 4b 76 4c 2f 78 69  63 7a 50 72 61 74 44 71  |1sKvL/xiczPratDq|
000001c0  4a 70 39 59 70 72 49 34  65 38 5a 56 51 63 43 30  |Jp9YprI4e8ZVQcC0|
000001d0  6a 52 4a 44 6c 45 66 4c  42 4c 6d 55 53 51 48 4f  |jRJDlEfLBLmUSQHO|
000001e0  48 43 2b 52 77 71 44 22  18 0a 13 76 32 2e 6b 6c  |HC+RwqD"...v2.kl|
000001f0  65 69 64 69 2e 62 65 65  7a 79 2e 64 65 76 12 01  |eidi.beezy.dev..|
00000200  31 28 01 0a                                       |1(..|
00000204
```

The above extract shows an encrypted payload with two crucial pieces of information:  
- the header ```enc:kms:v2:kleidi-kms-plugin:```.
- the key version ```1_1727188139``` 

If there is a rotation of the Vault transit key:
```
vault write -f transit/keys/kleidi/rotate
```

then replace the secrets:
```
kubectl get secret postkleidi -o json | kubectl replace -f -
secret/postkleidi replaced
```

then the following payload would expected:
```
kubectl -n kube-system exec etcd-kleidi-vault-prd-control-plane -- sh -c "ETCDCTL_ENDPOINTS='https://127.0.0.1:2379' ETCDCTL_CACERT='/etc/kubernetes/pki/etcd/ca.crt' ETCDCTL_CERT='/etc/kubernetes/pki/etcd/server.crt' ETCDCTL_KEY='/etc/kubernetes/pki/etcd/server.key' ETCDCTL_API=3 etcdctl get /registry/secrets/default/postkleidi" | hexdump -C 
```
``` 
00000000  2f 72 65 67 69 73 74 72  79 2f 73 65 63 72 65 74  |/registry/secret|
00000010  73 2f 64 65 66 61 75 6c  74 2f 70 6f 73 74 6b 6c  |s/default/postkl|
00000020  65 69 64 69 0a 6b 38 73  3a 65 6e 63 3a 6b 6d 73  |eidi.k8s:enc:kms|
00000030  3a 76 32 3a 6b 6c 65 69  64 69 2d 6b 6d 73 2d 70  |:v2:kleidi-kms-p|
00000040  6c 75 67 69 6e 3a 0a a3  02 53 67 55 93 db 8b 4e  |lugin:...SgU...N|
00000050  ad 83 d9 0e 89 e3 4c 14  f5 24 34 f7 1c c5 ea 37  |......L..$4....7|
00000060  22 8c 01 cd ec 3b 34 0c  f8 28 10 17 33 8a f1 a8  |"....;4..(..3...|
00000070  22 ab b8 9c 34 1d 54 f2  6d 75 ac 6c d5 d4 1e 9b  |"...4.T.mu.l....|
00000080  9d d0 ab ad 8d 6f 3e dd  7a 8d a7 3e f5 a9 6f a1  |.....o>.z..>..o.|
00000090  77 83 ba 1e 1e b0 25 73  64 c5 b4 91 b9 6e 46 21  |w.....%sd....nF!|
000000a0  fe 8d 4d c7 97 77 64 35  0e f3 96 20 93 12 e8 f2  |..M..wd5... ....|
000000b0  f6 0b 26 fc 61 7d f8 09  7c 08 c3 a5 4b 62 47 3e  |..&.a}..|...KbG>|
000000c0  69 59 83 0b c5 3a 9c 32  4c 0e e6 bb c5 38 5a be  |iY...:.2L....8Z.|
000000d0  77 8d 0e 5c 87 95 e8 27  65 0b b7 e1 37 d6 4a ed  |w..\...'e...7.J.|
000000e0  8b 7c 7b 33 e5 71 e3 20  a5 3b 28 8f 9c 89 73 e9  |.|{3.q. .;(...s.|
000000f0  9a 21 6b 3c 1b 25 c7 61  b1 81 5f 55 59 93 53 ec  |.!k<.%.a.._UY.S.|
00000100  d1 2e ca 8b d9 c8 1a d5  2c a3 4f 6e 52 1d 26 28  |........,.OnR.&(|
00000110  70 5f 04 c1 45 58 30 2f  14 b9 a0 7e 1a 50 6a 71  |p_..EX0/...~.Pjq|
00000120  98 64 14 23 df b7 15 41  6e e1 66 88 9d 72 c4 d9  |.d.#...An.f..r..|
00000130  c7 30 63 65 6c e2 23 e4  5f 88 da c4 50 40 cc ce  |.0cel.#._...P@..|
00000140  4e 96 91 54 64 07 97 54  63 b3 93 fe dc ae f9 6b  |N..Td..Tc......k|
00000150  11 62 50 70 8a 37 ca 7c  78 5e ac 1f d4 53 2d ba  |.bPp.7.|x^...S-.|
00000160  13 a6 92 be ed 48 aa 54  06 a7 a3 ee 12 1e 6b 6c  |.....H.T......kl|
00000170  65 69 64 69 2d 6b 6d 73  2d 70 6c 75 67 69 6e 5f  |eidi-kms-plugin_|
00000180  32 5f 31 37 32 37 31 38  38 37 30 34 1a 59 76 61  |2_1727188704.Yva|
00000190  75 6c 74 3a 76 32 3a 6f  4b 51 46 55 48 57 44 52  |ult:v2:oKQFUHWDR|
000001a0  52 31 30 41 4a 4f 51 74  6b 74 49 55 57 32 4f 69  |R10AJOQtktIUW2Oi|
000001b0  55 56 73 56 57 59 65 6a  79 77 44 56 53 45 52 46  |UVsVWYejywDVSERF|
000001c0  53 41 70 6e 75 76 6b 42  5a 75 65 34 44 61 78 43  |SApnuvkBZue4DaxC|
000001d0  78 45 50 7a 4e 6d 77 48  39 63 75 72 61 4f 4b 34  |xEPzNmwH9curaOK4|
000001e0  6b 6c 6a 7a 77 45 6a 22  18 0a 13 76 32 2e 6b 6c  |kljzwEj"...v2.kl|
000001f0  65 69 64 69 2e 62 65 65  7a 79 2e 64 65 76 12 01  |eidi.beezy.dev..|
00000200  31 28 01 0a                                       |1(..|
00000204
``` 

The version has been updated with ```2_1727188704``` and we can verify the key version in Vault:
```
vault read transit/keys/kleidi
```
```
Key                       Value
---                       -----
allow_plaintext_backup    false
auto_rotate_period        0s
deletion_allowed          false
derived                   false
exportable                false
imported_key              false
keys                      map[1:1727188139 2:1727188704]
latest_version            2
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


Then let's encrypt the pre-deployment secret too:

```
kubectl get secret prekleidi-secret -o json | kubectl replace -f -
``` 
Expected output:
```
secret/prekleidi-secret replaced
```
Query ```etcd```:
``` 
kubectl -n kube-system exec etcd-kleidi-vault-control-plane -- sh -c "ETCDCTL_ENDPOINTS='https://127.0.0.1:2379' ETCDCTL_CACERT='/etc/kubernetes/pki/etcd/ca.crt' ETCDCTL_CERT='/etc/kubernetes/pki/etcd/server.crt' ETCDCTL_KEY='/etc/kubernetes/pki/etcd/server.key' ETCDCTL_API=3 etcdctl get /registry/secrets/default/prekleidi-secret" | hexdump -C
```
Expected output:
``` 
00000000  2f 72 65 67 69 73 74 72  79 2f 73 65 63 72 65 74  |/registry/secret|
00000010  73 2f 64 65 66 61 75 6c  74 2f 70 72 65 6b 6c 65  |s/default/prekle|
00000020  69 64 69 2d 73 65 63 72  65 74 0a 6b 38 73 3a 65  |idi-secret.k8s:e|
00000030  6e 63 3a 6b 6d 73 3a 76  32 3a 6b 6c 65 69 64 69  |nc:kms:v2:kleidi|
00000040  2d 6b 6d 73 2d 70 6c 75  67 69 6e 3a 0a a9 02 5f  |-kms-plugin:..._|
00000050  0e 09 ea 2b 5d 17 e8 d2  ef 1d 3d 41 7d 7f dc d6  |...+].....=A}...|
00000060  b1 2e c5 63 1c c7 f1 97  80 f3 d7 29 9a 9e 68 a8  |...c.......)..h.|
00000070  ff 56 1d cf 07 84 1c 2e  b6 fe 94 83 11 34 d9 4b  |.V...........4.K|
00000080  c9 94 1d d0 ce 38 08 ec  cf 67 76 f3 9a ea 8a dd  |.....8...gv.....|
00000090  4e 07 0e 11 9e 50 88 be  c7 26 cb ef 8f 58 c3 9c  |N....P...&...X..|
000000a0  bb 6a db 03 83 47 af ec  d1 95 6c b5 24 c0 c3 c9  |.j...G....l.$...|
000000b0  ce 6f 68 4f 64 40 8d df  05 3f 83 96 11 d0 72 9c  |.ohOd@...?....r.|
000000c0  b6 c7 7a 7c fd c2 a8 af  12 26 30 08 fa 53 54 71  |..z|.....&0..STq|
000000d0  7b df 9f c0 06 7c f2 81  a9 dc 96 49 fa bc 7e cb  |{....|.....I..~.|
000000e0  4c b7 e2 df d0 de 2e 14  9b 28 34 ef f5 8c 4f 1e  |L........(4...O.|
000000f0  f4 b5 33 9d f1 12 0c 5f  38 80 fd ab b8 54 22 59  |..3...._8....T"Y|
00000100  ac 85 9d 1a 74 6b 4d 93  f1 9c 6e 62 39 b1 1a 9e  |....tkM...nb9...|
00000110  ad a1 32 8e 22 88 f6 12  95 f4 38 a1 ce db 96 1e  |..2.".....8.....|
00000120  56 cf aa ee 31 28 f2 51  2b 2e 86 4d 36 a9 f0 b5  |V...1(.Q+..M6...|
00000130  1f 1d db d7 b3 df e0 37  01 36 2b f8 93 29 24 d8  |.......7.6+..)$.|
00000140  45 5b ab 51 6d 64 1b c2  1f c7 2d f4 5e 2d af 3d  |E[.Qmd....-.^-.=|
00000150  83 d4 33 dd e9 e6 77 ea  9e 84 16 0a 43 54 c8 24  |..3...w.....CT.$|
00000160  ef b9 b4 1d 9d a3 f1 a0  e0 ed d3 fa e7 b1 9e 37  |...............7|
00000170  bd c2 b5 f7 f8 e6 78 0d  12 11 6b 6c 65 69 64 69  |......x...kleidi|
00000180  2d 6b 6d 73 2d 70 6c 75  67 69 6e 1a 59 76 61 75  |-kms-plugin.Yvau|
00000190  6c 74 3a 76 31 3a 75 46  4a 52 47 38 68 44 36 52  |lt:v1:uFJRG8hD6R|
000001a0  4b 69 50 4c 6c 76 6d 52  44 36 75 6b 34 47 6b 4c  |KiPLlvmRD6uk4GkL|
000001b0  36 48 6b 57 73 78 4e 62  37 4c 47 36 69 33 4e 37  |6HkWsxNb7LG6i3N7|
000001c0  54 79 76 33 6b 47 79 4d  51 2b 31 6c 62 5a 62 59  |Tyv3kGyMQ+1lbZbY|
000001d0  6a 6f 36 53 4b 67 45 55  64 38 56 68 47 37 78 67  |jo6SKgEUd8VhG7xg|
000001e0  54 66 30 74 63 76 22 18  0a 13 76 32 2e 6b 6c 65  |Tf0tcv"...v2.kle|
000001f0  69 64 69 2e 62 65 65 7a  79 2e 64 65 76 12 01 31  |idi.beezy.dev..1|
00000200  28 01 0a                                          |(..|
00000203
```
