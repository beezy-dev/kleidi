#!/bin/bash
#############################################################################
# Script Name  :   00_k8s_en.sh                                               
# Description  :   Provide a view of the Kubernetes environment                                                                              
# Args         :   
# Author       :   romdalf aka Rom Adams
# Issues       :   Issues&PR https://github.com/beezy-dev/kleidi
#############################################################################

set -euo pipefail


echo
echo -e "Test kubernetes environment for kleidi-kms-plugin"

echo -e "  -> Cleaning any existing vault test env"
killall -9 vault ||true
echo -e "  -> Cleaning any existing kind test env" 
kind delete cluster --quiet --name kleidi-vault

# echo -e "  -> Starting HashiCorp Vault"
# nohup vault server -dev -dev-root-token-id=kleidi-demo -dev-listen-address=0.0.0.0:8200 2> /dev/null &
# sleep 3 

# export VAULT_ADDR=http://192.168.172.243:8200
# export VAULT_TOKEN="kleidi-demo"
# export VAULT_SKIP_VERITY="true"
# echo -e "  -> Enabling vault transit engine"
# vault secrets enable transit 

# echo -e "  -> Enabling vault transit engine"
# vault write -f transit/keys/kleidi

# echo -e "  -> Applying vault policy"
# vault policy write kleidi configuration/vault/vault-policy.hcl 

# echo -e "  -> Starting kind kubernetes instance for vault testing"
# kind create cluster --quiet --config configuration/k8s/kind/kind-vault.yaml
# sleep 3

# echo -ne "  -> Checking kubernetes version (>=1.29).."
# MINORVERSION=`kubectl version -o json |jq -r '.serverVersion.minor'`
# NODEVERSION=`kubectl version -o json |jq -r '.serverVersion.gitVersion'`
# if [[ ${MINORVERSION} -lt "29" ]]
# then 
#     echo -ne "NOK\n"
#     echo -e "  $/!\ kleidi-kms-plugin$ requires kubernetes >=1.29 - current ${NODEVERSION}" 
#     exit
# fi 
# echo -ne ".OK (${NODEVERSION})\n"

# echo -e "  -> Creating kubernetes SA/TOKEN/RBAC"
# kubectl apply -f configuration/k8s/deploy/vault-sa.yaml 1> /dev/null

# echo -e "  -> Enable vault kubernetes authentication"
# vault auth enable kubernetes

# echo -e "  -> Exporting token, cert, and k8s cluster info"
# TOKEN=$(kubectl get secret -n kube-system kleidi-vault-auth -o go-template='{{ .data.token }}' | base64 --decode)
# CERT=$(kubectl get cm kube-root-ca.crt -o jsonpath="{['data']['ca\.crt']}")
# K8SHOST=$(kubectl config view --raw --minify --flatten --output 'jsonpath={.clusters[].cluster.server}')

# echo -e "  -> Write vault config for kubernetes authentication"
# vault write auth/kubernetes/config token_reviewer_jwt="${TOKEN}" kubernetes_host="${K8SHOST}" kubernetes_ca_cert="${CERT}"

# echo -e "  -> Link vault config and policy for kubernetes authentication"
# vault write auth/kubernetes/role/kleidi bound_service_account_names=kleidi-vault-auth bound_service_account_namespaces=kube-system policies=kleidi ttl=24h

# echo -e "  -> Deploy kleidi kms plugin for vault"
# kubectl apply -f configuration/k8s/deploy/vault-pod-kleidi-kms.yaml


# echo -e "  Cleaning any existing vault test env"
# killall -9 vault
# echo -e "  Cleaning any existing kind test env" 
# kind delete cluster --quiet --name kleidi-vault

