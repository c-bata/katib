#!/bin/bash

set -ex

set -o errexit
set -o nounset
set -o pipefail

REGISTRY="gcr.io/kubeflow-images-public"
TAG="latest"
PREFIX="katib/v1alpha3"
CMD_PREFIX="cmd"
MACHINE_ARCH=`uname -m`

SCRIPT_ROOT=$(dirname ${BASH_SOURCE})/../..

cd ${SCRIPT_ROOT}

usage() { echo "Usage: $0 [-t <tag>] [-r <registry>] [-p <prefix>]" 1>&2; exit 1; }

while getopts ":t::r::p:" opt; do
    case $opt in
        t)
            TAG=${OPTARG}
            ;;
        r)
            REGISTRY=${OPTARG}
            ;;
        p)
            PREFIX=${OPTARG}
            ;;
        *)
            usage
            ;;
    esac
done
echo "Registry: ${REGISTRY}, tag: ${TAG}, prefix: ${PREFIX}"

docker build -t ${REGISTRY}/${PREFIX}/suggestion-goptuna:${TAG} -f ${CMD_PREFIX}/suggestion/goptuna/v1alpha3/Dockerfile .
docker tag ${REGISTRY}/${PREFIX}/suggestion-goptuna:${TAG} asia.gcr.io/cyberagent-263/katib-v1alpha3-suggestion-goptuna
gcloud docker -- push asia.gcr.io/cyberagent-263/katib-v1alpha3-suggestion-goptuna

kubectl apply -f manifests/v1alpha3/katib-controller/katib-config.yaml
