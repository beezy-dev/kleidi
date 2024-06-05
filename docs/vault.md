# HashiCorp Vault/OpenBao Implementation

## Vault Deployment

This implementation includes the initialization of an external HashiCorp Vault with a Transit Key Engine.     
Download the HashiCorp Vault binary or install it with ```brew```, then run the following command:

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
export VAULT_ADDR="http://<your_ip>:8200"
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

To provide a secure connectivity between ```kleidi``` running on kubernetes and ```vault```, a ```ServiceAccount``` with a token and RBAC is configured. 

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
The k8shost variable might output ```127.0.0.1``` which will result in a dial back failure from Vault when verifying the kubernetes authentication via the provided certificate. Correct value should be a FQDN or ```kubernetes.default.svc.cluster.local```. In the above case ```k8shost=https://kubernetes.default.svc.cluster.local:port``` is a valide option. 

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
      image: ghcr.io/beezy-dev/kleidi-kms-plugin:vault-1283a8e
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
The above provides the last tested images. It is advised not to change it. 

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

This should triggere a restart of the kubernetes API server.

## Encryption/Decryption Test

To validate there is now encryption, create a post-deployment secret:
```
kubectl create secret generic postkleidi-secret -n default --from-literal=mykey=mydata
```
Expected output:
```
secret/postkleidi-secret created
```
```
kubectl -n kube-system exec etcd-kleidi-vault-control-plane -- sh -c "ETCDCTL_ENDPOINTS='https://127.0.0.1:2379' ETCDCTL_CACERT='/etc/kubernetes/pki/etcd/ca.crt' ETCDCTL_CERT='/etc/kubernetes/pki/etcd/server.crt' ETCDCTL_KEY='/etc/kubernetes/pki/etcd/server.key' ETCDCTL_API=3 etcdctl get /registry/secrets/default/postkleidi-secret" | hexdump -C
```
Expected output:
```
00000000  2f 72 65 67 69 73 74 72  79 2f 73 65 63 72 65 74  |/registry/secret|
00000010  73 2f 64 65 66 61 75 6c  74 2f 70 6f 73 74 6b 6c  |s/default/postkl|
00000020  65 69 64 69 2d 73 65 63  72 65 74 0a 6b 38 73 3a  |eidi-secret.k8s:|
00000030  65 6e 63 3a 6b 6d 73 3a  76 32 3a 6b 6c 65 69 64  |enc:kms:v2:kleid|
00000040  69 2d 6b 6d 73 2d 70 6c  75 67 69 6e 3a 0a aa 02  |i-kms-plugin:...|
00000050  98 45 a9 2d 64 9c 71 0e  ad cb c5 56 22 8a 3a 0e  |.E.-d.q....V".:.|
00000060  e9 84 d3 ce 57 24 6b 99  c2 2d 6d 87 bf 67 37 2e  |....W$k..-m..g7.|
00000070  71 bd 0c a0 69 a6 56 ae  13 67 f3 fc f2 5c 81 66  |q...i.V..g...\.f|
00000080  2e 8f 62 fd ef ec 71 46  30 05 eb e4 a6 0d 54 fc  |..b...qF0.....T.|
00000090  a5 0b 6e 6b 4d 81 a3 ab  5a e6 0c ff 65 3c 0e 16  |..nkM...Z...e<..|
000000a0  c5 4e 6b 0e 3d b4 6e b9  b6 90 7d 53 2b 66 ba d9  |.Nk.=.n...}S+f..|
000000b0  f0 71 00 0e 8a 3a d4 44  d9 3b 78 7c b6 dc dd b0  |.q...:.D.;x|....|
000000c0  ea 53 04 c3 d5 31 f2 10  06 3a 39 e8 d1 8b 37 5d  |.S...1...:9...7]|
000000d0  25 fd f5 ee 00 98 e7 45  64 34 a3 3f f8 94 aa 9b  |%......Ed4.?....|
000000e0  ea 8e 5f 0b bf b3 84 e4  71 7e 57 b0 50 5a d3 58  |.._.....q~W.PZ.X|
000000f0  61 b4 77 71 9a 1f a1 e9  33 e3 b7 b1 e6 12 32 fd  |a.wq....3.....2.|
00000100  97 91 48 84 cc 27 a5 b3  cf 55 d9 45 f7 6f 0f 50  |..H..'...U.E.o.P|
00000110  06 4f ba 59 1b e2 2f 47  3e 9d c8 f7 8d c6 b2 ea  |.O.Y../G>.......|
00000120  7e 61 1f 91 c5 44 90 34  5a dc 8b 22 41 2a 8e f0  |~a...D.4Z.."A*..|
00000130  2e db 86 f5 c7 ea 23 07  56 10 d1 9a 89 07 23 58  |......#.V.....#X|
00000140  be cc 0f f3 d0 fd a1 d8  57 74 6a 24 7c 10 91 85  |........Wtj$|...|
00000150  ee 19 f7 ef fd ea 9f 1d  a6 93 d6 37 1c 90 6e d3  |...........7..n.|
00000160  e5 9d 84 90 96 18 78 af  3f 24 49 b9 81 2a fc e4  |......x.?$I..*..|
00000170  be b6 bb e4 4d 78 c2 bc  cb d4 12 11 6b 6c 65 69  |....Mx......klei|
00000180  64 69 2d 6b 6d 73 2d 70  6c 75 67 69 6e 1a 59 76  |di-kms-plugin.Yv|
00000190  61 75 6c 74 3a 76 31 3a  75 46 4a 52 47 38 68 44  |ault:v1:uFJRG8hD|
000001a0  36 52 4b 69 50 4c 6c 76  6d 52 44 36 75 6b 34 47  |6RKiPLlvmRD6uk4G|
000001b0  6b 4c 36 48 6b 57 73 78  4e 62 37 4c 47 36 69 33  |kL6HkWsxNb7LG6i3|
000001c0  4e 37 54 79 76 33 6b 47  79 4d 51 2b 31 6c 62 5a  |N7Tyv3kGyMQ+1lbZ|
000001d0  62 59 6a 6f 36 53 4b 67  45 55 64 38 56 68 47 37  |bYjo6SKgEUd8VhG7|
000001e0  78 67 54 66 30 74 63 76  22 18 0a 13 76 32 2e 6b  |xgTf0tcv"...v2.k|
000001f0  6c 65 69 64 69 2e 62 65  65 7a 79 2e 64 65 76 12  |leidi.beezy.dev.|
00000200  01 31 28 01 0a                                    |.1(..|
00000205
```

**The above extract shows an encrypted payload with the header ```enc:kms:v2:kleidi-kms-plugin:```.**

Then let's encrypt the pre-deployment secret too:

```
kubectl get secret prekleidi-secret -o json | /home/linuxbrew/.linuxbrew/bin/kubectl replace -f -
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
