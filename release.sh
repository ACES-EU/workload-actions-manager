#!/bin/bash
set -euxo pipefail

version="0.7.0"

docker build --platform linux/amd64 --push -t ghcr.io/vidbregar/workload-actions-manager/wam:$version -f wam/Dockerfile .

cd wam-scheduler || exit
./hack/update-codegen.sh
ARCH=amd64 make local-image
cd - || exit
docker tag localhost:5000/scheduler-plugins/kube-scheduler:latest ghcr.io/vidbregar/workload-actions-manager/wam-scheduler:$version
docker push ghcr.io/vidbregar/workload-actions-manager/wam-scheduler:$version
