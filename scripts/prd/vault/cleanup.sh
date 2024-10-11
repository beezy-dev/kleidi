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
