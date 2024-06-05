# SoftHSM Implementation

## Deployment
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

## Encryption/Decryption Test
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