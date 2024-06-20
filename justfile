default:
  just --list


build_and_push_wam_scheduler:
    #!/usr/bin/env bash
    set -euxo pipefail
    cd wam-scheduler || exit
    ./hack/update-codegen.sh
    make local-image
    docker tag localhost:5000/scheduler-plugins/kube-scheduler:latest k3d-registry.localhost:50000/wam-scheduler:latest
    docker push k3d-registry.localhost:50000/wam-scheduler:latest

build_and_push_wam:
    docker build -t k3d-registry.localhost:50000/wam:latest wam
    docker push k3d-registry.localhost:50000/wam:latest

build_and_push: build_and_push_wam_scheduler build_and_push_wam
