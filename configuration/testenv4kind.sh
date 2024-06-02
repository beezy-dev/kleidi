#!/bin/bash
#############################################################################
# Script Name  :   00_k8s_en.sh                                               
# Description  :   Provide a view of the Kubernetes environment                                                                              
# Args         :   
# Author       :   romdalf aka Rom Adams
# Issues       :   Issues&PR https://github.com/beezy-dev/kleidi
#############################################################################

set -euo pipefail

# Define some colours for later
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[1;34m'
NC='\033[0m' # No Color

echo 
echo -e "${NC}Verify kubernetes environment to deploy ${BLUE}kleidi-kms-plugin${NC}"
echo -ne "  Checking node count (minimum 1)......................."
NODECOUNT=`kubectl get nodes -o name | wc -l`
if [ ${NODECOUNT} -lt 1 ]
then 
    echo -ne "${RED}NOK${NC}\n" 
    exit
fi 
echo -ne ".${GREEN}OK${NC} (${RED}${NODECOUNT}${NC})\n"

echo -ne "  Checking kubernetes version (>=1.29).................."
MINORVERSION=`kubectl version -o json |jq -r '.serverVersion.minor'`
NODEVERSION=`kubectl version -o json |jq -r '.serverVersion.gitVersion'`
if [[ ${MINORVERSION} -lt "29" ]]
then 
    echo -ne "${RED}NOK${NC}\n"
    echo -e "  ${RED}/!\ ${NC}${BLUE}kleidi-kms-plugin${NC} requires kubernetes >=1.29 - current ${RED}${NODEVERSION}${NC}${NC}" 
    exit
fi 
echo -ne ".${GREEN}OK${NC} (${RED}${NODEVERSION}${NC})\n"

echo -ne "  Checking for existing ${BLUE}kleidi-kms-plugin${NC} deployment...."
KLEIDICOUNT=`kubectl get pod -n kube-system -o name |grep kleidi-kms |wc -l`
if [ ${KLEIDICOUNT} -lt 1 ]
then
    echo -ne "${RED}NOK${NC}\n"
    echo -e "  ${RED}/!\ ${NC}${BLUE}kleidi-kms-plugin${NC} deployed on this kubernetes cluster.${NC}"
    exit
fi 
echo -ne ".${GREEN}OK${NC} (${RED}${KLEIDICOUNT}${NC})\n"
echo 