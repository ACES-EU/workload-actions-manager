default:
  just --list

build_and_push_scheduler:
  docker build -t k3d-registry.localhost:50000/wam-scheduler:latest wam-scheduler
  docker push k3d-registry.localhost:50000/wam-scheduler:latest

