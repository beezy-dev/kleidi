#!/bin/bash
#############################################################################
# Script Name  :   env4kvault.sh                                               
# Description  :   Provide a view of the Kubernetes environment                                                                              
# Args         :   
# Author       :   romdalf aka Rom Adams
# Issues       :   Issues&PR https://github.com/beezy-dev/kleidi
#############################################################################

set -euo pipefail

echo
echo -e "Latest tested kubernetes environment for kleidi-kms-plugin"

echo
echo -e "  -> Cleaning any existing vault test env"
#killall -9 vault ||true
podman rm vault --force

echo
echo -e "  -> Cleaning any existing kind test env" 
kind delete cluster --name kleidi-vault-prd

echo
echo -e "  -> Cleaning vault-encryption-config.yaml"
cp manifests/vault-encryption-config-bkp.yaml manifests/vault-encryption-config.yaml

echo
echo -e "  -> Starting HashiCorp Vault"
# nohup vault server -dev -dev-root-token-id=kleidi-demo -dev-listen-address=0.0.0.0:8200 2> /dev/null &
podman run --cap-add=IPC_LOCK -e 'VAULT_DEV_ROOT_TOKEN_ID=kleidi-demo' -e 'VAULT_LOG_LEVEL=debug' -d --name=vault --network kind --ip 10.89.0.10 -p 8200:8200 hashicorp/vault
echo -e "  -> Sleeping for 10 seconds"
sleep 10

echo -e "  -> Exporting HashiCorp Vault parameters"
export VAULT_ADDR=http://127.0.0.1:8200
export VAULT_TOKEN="kleidi-demo"
export VAULT_SKIP_VERITY="true"

echo
echo -e "  -> Enabling HashiCorp Vault transit engine"
vault secrets enable transit 

echo
echo -e "  -> Creating kleidi key in HashiCorp Vault transit engine"
vault write -f transit/keys/kleidi

echo
echo -e "  -> Applying kleidi policy in HashiCorp Vault"
vault policy write kleidi vault/vault-policy.hcl 

echo
echo -e "  -> Starting k8s instance with Kind"
kind create cluster --config manifests/kind-vault.yaml
echo -e "  -> Sleeping for 10 seconds"
sleep 10

echo
echo -ne "  -> Checking k8s deployed version (>=1.29)...."
MINORVERSION=`kubectl version -o json |jq -r '.serverVersion.minor'`
NODEVERSION=`kubectl version -o json |jq -r '.serverVersion.gitVersion'`
if [[ ${MINORVERSION} -lt "29" ]]
then 
    echo -ne "NOK\n"
    echo -e "  $/!\ kleidi-kms-plugin$ requires kubernetes >=1.29 - current ${NODEVERSION}" 
    exit
fi 
echo -ne ".OK (${NODEVERSION})\n"

echo
echo -e "  -> Creating a pre kleidi deployment Secret"
kubectl create secret generic prekleidi -n default --from-literal=mykey=mydata

echo
echo -e "  -> Creating 1001 pre kleidi deployment Secrets"
for i in {0..1000}; do kubectl create secret generic prekleidi$i -n default --from-literal=mykey=mydata; done

echo
echo -e "  -> Creating kleidi k8s ServiceAccount/SA Secret/RBAC"
kubectl apply -f manifests/vault-sa.yaml

echo 
echo -e "  -> Enable k8s auth in HashiCorp Vault"
vault auth enable kubernetes
echo -e "  -> Sleeping for 5 seconds"
sleep 5

echo
echo -e "  -> Exporting k8s token, cert, and k8s cluster info HashiCorp Vault k8s auth"
echo -e "     -> export kleidi-vault-auth secret token"
export TOKEN=$(kubectl get secret -n kube-system kleidi-vault-auth -o go-template='{{ .data.token }}' | base64 --decode) 
echo -e "     -> export k8s root CA" 
export CERT=$(kubectl get cm kube-root-ca.crt -o jsonpath="{['data']['ca\.crt']}")
echo -e "     -> export k8s certificate issuer"
#export K8SPORT=$(kubectl config view --raw --minify --flatten --output 'jsonpath={.clusters[].cluster.server}')
export K8SISSU=$(podman inspect -f '{{.NetworkSettings.Networks.kind.IPAddress}}' kleidi-vault-prd-control-plane)
export K8SHOST=https://"$K8SISSU":6443
echo -e "  -> Sleeping for 5 seconds"
sleep 5

echo
echo -e "  -> Write k8s auth config in HashiCorp Vault"
vault write auth/kubernetes/config token_reviewer_jwt="${TOKEN}" kubernetes_host="${K8SHOST}" kubernetes_ca_cert="${CERT}"

echo
echo -e "  -> Create k8s auth role in HashiCorp Vault with kleidi ServiceAccount"
vault write auth/kubernetes/role/kleidi bound_service_account_names=kleidi-vault-auth bound_service_account_namespaces=kube-system policies=kleidi ttl=24h

echo
echo -e "  -> Deploy kleidi static pod with HashiCorp Vault integration"
kubectl apply -f manifests/vault-pod-kleidi-kms.yaml
echo -e "  -> Sleeping for 60 seconds to allow pull image"
sleep 60 

KLEIDI=`kubectl -n kube-system get pod kleidi-kms-plugin --no-headers -ocustom-columns=status:.status.phase`
if [[ ${KLEIDI} == "Running" ]]
then 
    echo -e "  -> kleidi is running"
else
    echo -e "  -> kleidi is not running"
    exit
fi 

echo
echo -e "  -> Update vault-encryption-config.yaml with KMS provider"
cp manifests/vault-encryption-config-with_kms.yaml manifests/vault-encryption-config.yaml
sleep 5

echo
echo -e "  -> Trigger Kind k8s API server restart"
kubectl delete -n kube-system pod/kube-apiserver-kleidi-vault-prd-control-plane
echo -e "  -> Sleeping for 30 seconds to allow kube-apiserver to restart"
sleep 30

echo 
echo -e "  -> Checking a pre kleidi deployment Secret"
# kubectl -n kube-system exec etcd-kleidi-vault-prd-control-plane -- sh -c "ETCDCTL_ENDPOINTS='https://127.0.0.1:2379' ETCDCTL_CACERT='/etc/kubernetes/pki/etcd/ca.crt' ETCDCTL_CERT='/etc/kubernetes/pki/etcd/server.crt' ETCDCTL_KEY='/etc/kubernetes/pki/etcd/server.key' ETCDCTL_API=3 etcdctl get /registry/secrets/default/prekleidi" | hexdump -C

if kubectl -n kube-system exec etcd-kleidi-vault-prd-control-plane -- sh -c "ETCDCTL_ENDPOINTS='https://127.0.0.1:2379' ETCDCTL_CACERT='/etc/kubernetes/pki/etcd/ca.crt' ETCDCTL_CERT='/etc/kubernetes/pki/etcd/server.crt' ETCDCTL_KEY='/etc/kubernetes/pki/etcd/server.key' ETCDCTL_API=3 etcdctl get /registry/secrets/default/prekleidi" | hexdump -C | grep mydata;
then 
    echo -e "  unencrypted prekleidi Secret object found :)" 
else 
    echo -e "  /!\ no unencrypted prekleidi Secret object found!"
    exit
fi
echo -e "  -> Sleeping for 10 seconds to allow kube-apiserver to restart"
sleep 10 

echo 
echo -e "  -> Performing replace of pre kleidi deployment Secret"
kubectl get secret prekleidi -o json | kubectl replace -f -

echo -e "  -> Checking a pre kleidi deployment Secret replace encryption"
# kubectl -n kube-system exec etcd-kleidi-vault-prd-control-plane -- sh -c "ETCDCTL_ENDPOINTS='https://127.0.0.1:2379' ETCDCTL_CACERT='/etc/kubernetes/pki/etcd/ca.crt' ETCDCTL_CERT='/etc/kubernetes/pki/etcd/server.crt' ETCDCTL_KEY='/etc/kubernetes/pki/etcd/server.key' ETCDCTL_API=3 etcdctl get /registry/secrets/default/prekleidi" | hexdump -C

if kubectl -n kube-system exec etcd-kleidi-vault-prd-control-plane -- sh -c "ETCDCTL_ENDPOINTS='https://127.0.0.1:2379' ETCDCTL_CACERT='/etc/kubernetes/pki/etcd/ca.crt' ETCDCTL_CERT='/etc/kubernetes/pki/etcd/server.crt' ETCDCTL_KEY='/etc/kubernetes/pki/etcd/server.key' ETCDCTL_API=3 etcdctl get /registry/secrets/default/prekleidi" | hexdump -C |grep kms;
then
    echo -e "  encrypted prekleidi Secret object found :)" 
else
    echo -e "  /!\ no encrypted prekleidi Secret object found :("
    echo -e "  -> Sleeping for 10 seconds to allow kube-apiserver to restart"
    sleep 10
    echo -e "  -> Performing replace again to avoid API server restart issue"
    kubectl get secret prekleidi -o json | kubectl replace -f -
    kubectl get secret prekleidi -o json | kubectl replace -f - 
    if kubectl -n kube-system exec etcd-kleidi-vault-prd-control-plane -- sh -c "ETCDCTL_ENDPOINTS='https://127.0.0.1:2379' ETCDCTL_CACERT='/etc/kubernetes/pki/etcd/ca.crt' ETCDCTL_CERT='/etc/kubernetes/pki/etcd/server.crt' ETCDCTL_KEY='/etc/kubernetes/pki/etcd/server.key' ETCDCTL_API=3 etcdctl get /registry/secrets/default/prekleidi" | hexdump -C |grep kms;
    then
	echo -e "  encrypted prekleidi Secret object found :)" 
    else
    	echo -e "  /!\ no encrypted prekleidi Secret object found :("
    fi
fi
echo -e "  -> Sleeping for 10 seconds to allow kube-apiserver to restart"
sleep 10


echo
echo -e "  -> Creating a post kleidi deployment Secret"
kubectl create secret generic postkleidi -n default --from-literal=mykey=mydata

echo 
echo -e "  -> Checking a post kleidi deployment Secret"
# kubectl -n kube-system exec etcd-kleidi-vault-prd-control-plane -- sh -c "ETCDCTL_ENDPOINTS='https://127.0.0.1:2379' ETCDCTL_CACERT='/etc/kubernetes/pki/etcd/ca.crt' ETCDCTL_CERT='/etc/kubernetes/pki/etcd/server.crt' ETCDCTL_KEY='/etc/kubernetes/pki/etcd/server.key' ETCDCTL_API=3 etcdctl get /registry/secrets/default/postkleidi" | hexdump -C

if kubectl -n kube-system exec etcd-kleidi-vault-prd-control-plane -- sh -c "ETCDCTL_ENDPOINTS='https://127.0.0.1:2379' ETCDCTL_CACERT='/etc/kubernetes/pki/etcd/ca.crt' ETCDCTL_CERT='/etc/kubernetes/pki/etcd/server.crt' ETCDCTL_KEY='/etc/kubernetes/pki/etcd/server.key' ETCDCTL_API=3 etcdctl get /registry/secrets/default/postkleidi" | hexdump -C | grep kms;
then 
    echo -e "  encrypted postkleidi Secret object found :)" 
else 
    echo -e "  /!\ no encrypted postkleidi Secret object found!"
    exit
fi 
echo -e "  -> Sleeping for 10 seconds to allow kube-apiserver to restart"
sleep 10

echo
echo -e "  -> Replacing 1001 pre kleidi deployment Secrets"
for i in {0..1000}; do kubectl get secret prekleidi$i -o json | kubectl replace -f -; done

echo -e "  -> Checking 1001 pre kleidi deployment Secrets encryption"
for i in {0..1000}
do
        if kubectl -n kube-system exec etcd-kleidi-vault-prd-control-plane -- sh -c "ETCDCTL_ENDPOINTS='https://127.0.0.1:2379' ETCDCTL_CACERT='/etc/kubernetes/pki/etcd/ca.crt' ETCDCTL_CERT='/etc/kubernetes/pki/etcd/server.crt' ETCDCTL_KEY='/etc/kubernetes/pki/etcd/server.key' ETCDCTL_API=3 etcdctl get /registry/secrets/default/prekleidi$i" | hexdump -C |grep kms;
        then
                echo -e "  encrypted prekleidi Secret object number $i found :)"
        else
                echo -e "  /!\ no encrypted prekleidi Secret object $i found :("
                exit
        fi
done

echo
echo -e "  -> Creating 1001 secrets"
for i in {0..1000}; do kubectl create secret generic postkleidi$i -n default --from-literal=mykey=mydata; done

echo -e "  -> Checking 1001 secrets encryption"
for i in {0..1000}
do
	if kubectl -n kube-system exec etcd-kleidi-vault-prd-control-plane -- sh -c "ETCDCTL_ENDPOINTS='https://127.0.0.1:2379' ETCDCTL_CACERT='/etc/kubernetes/pki/etcd/ca.crt' ETCDCTL_CERT='/etc/kubernetes/pki/etcd/server.crt' ETCDCTL_KEY='/etc/kubernetes/pki/etcd/server.key' ETCDCTL_API=3 etcdctl get /registry/secrets/default/postkleidi$i" | hexdump -C |grep kms;
	then
		echo -e "  encrypted postkleidi Secret object number $i found :)"
	else
		echo -e "  /!\ no encrypted postkleidi Secret object $i found :("
		exit
	fi
done
echo -e "  -> Sleeping for 10"
sleep 10


echo
echo -e "  -> Rotating key"
vault write -f transit/keys/kleidi/rotate
echo -e "  -> Sleeping for 10"
sleep 10

echo
echo -e "  -> Replacing 1001 pre kleidi deployment Secrets with new rotated key"
for i in {0..1000}; do kubectl get secret prekleidi$i -o json | kubectl replace -f -; done

echo -e "  -> Checking 1001 pre kleidi deployment Secrets encryption with new rotated key"
for i in {0..1000}
do
        if kubectl -n kube-system exec etcd-kleidi-vault-prd-control-plane -- sh -c "ETCDCTL_ENDPOINTS='https://127.0.0.1:2379' ETCDCTL_CACERT='/etc/kubernetes/pki/etcd/ca.crt' ETCDCTL_CERT='/etc/kubernetes/pki/etcd/server.crt' ETCDCTL_KEY='/etc/kubernetes/pki/etcd/server.key' ETCDCTL_API=3 etcdctl get /registry/secrets/default/prekleidi$i" | hexdump -C |grep "2_";
        then
                echo -e "  encrypted prekleidi Secret object number $i rotated :)"
        else
                echo -e "  /!\ no rotated prekleidi Secret object $i found :("
                exit
        fi
done

echo -e "  -> Checking 1001 postkleidi Secrets with old encryption key"
for i in {0..1000}
do
        if kubectl -n kube-system exec etcd-kleidi-vault-prd-control-plane -- sh -c "ETCDCTL_ENDPOINTS='https://127.0.0.1:2379' ETCDCTL_CACERT='/etc/kubernetes/pki/etcd/ca.crt' ETCDCTL_CERT='/etc/kubernetes/pki/etcd/server.crt' ETCDCTL_KEY='/etc/kubernetes/pki/etcd/server.key' ETCDCTL_API=3 etcdctl get /registry/secrets/default/postkleidi$i" | hexdump -C |grep "1_";
        then
                echo -e "  encrypted postkleidi Secret object number $i found :)"
        else
                echo -e "  /!\ no encrypted postkleidi Secret object $i found :("
                exit
        fi
done
echo -e "  -> Sleeping for 10"
sleep 10
