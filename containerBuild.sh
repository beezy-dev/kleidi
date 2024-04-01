#!/bin/bash
#############################################################################
# Script Name  :   containerBuild.sh                                                                                           
# Description  :   Build container images with Git commit as a tag                                                                              
# Args         :   
# Author       :   rom@beezy.dev
# Issues       :   Issues&PR https://github.com/beezy-dev/kleidi.git
#############################################################################

set -euo pipefail

# Define some colours for later
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[1;34m'
NC='\033[0m' # No Color


# Define variables 
VERSION=$(git log -1 --pretty=%h)
GITREPO="https://github.com/beezy-dev/kleidi.git" 
CONTREG="ghcr.io/beezy-dev/kleidi-kms-plugin" 
INITREG="ghcr.io/beezy-dev/kleidi-kms-init" 
BUILDDT=$(date '+%F_%H:%M:%S' )

STR="'$*'" 

# echo 
# echo -e "${NC}Running gosec with report ${BLUE}results.sarif${NC}."
# gosec -no-fail -fmt sarif -out results.sarif ./...

echo 
echo -e "${NC}Git commit with message ${BLUE}$STR${NC}."
git add --all && git commit -m "$STR" 

echo -e "${NC}Git push to ${BLUE}$GITREPO${NC}." 
git push

echo -e "${NC}Building kleidi container image ${BLUE}$CONTREG:$VERSION${NC} on ${BLUE}$BUILDDT${NC}."  
podman build -f Containerfile-kleidi-kms -t "$CONTREG:$VERSION" -t "$CONTREG:latest" --build-arg VERSION="$VERSION"

echo -e "${NC}Container pushed to push to ${BLUE}$CONTREG${NC} with tags ${BLUE}$VERSION${NC} and ${BLUE}latest${NC}." 
podman push $CONTREG:$VERSION
podman push $CONTREG:latest

echo -e "${NC}Building kleidi init container image ${BLUE}$INITREG:$VERSION${NC} on ${BLUE}$BUILDDT${NC}."  
podman build -f platform/kleidi-init/Containerfile -t "$INITREG:$VERSION" -t "$INITREG:latest" --build-arg VERSION="$VERSION"
podman push $INITREG:$VERSION
podman push $INITREG:latest