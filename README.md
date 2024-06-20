# Workload Actions Manager (WAM)

## Requirements

- [Docker](https://docs.docker.com/get-docker/)
- [Kubectl](https://kubernetes.io/docs/tasks/tools/#kubectl)
- [K3d](https://k3d.io/v5.2.2/#installation)
- [Just](https://github.com/casey/just?tab=readme-ov-file#installation)
- [Helm](https://helm.sh/docs/intro/install/)

## Build

todo

## Create a Cluster

```bash
grep k3d-registry.localhost /etc/hosts || echo "127.0.0.1 k3d-registry.localhost" | sudo tee -a /etc/hosts
k3d cluster create --config deploy/k3d.yaml --api-port localhost:5443
```

## Deploy

```bash
helm install wam-redis deploy/redis
kubectl apply -f deploy/wam-scheduler
kubectl apply -f deploy/wam
kubectl apply -f deploy/test
kubectl port-forward wam_pod_name 3000:3000
```

## Test Create Action

```bash
curl -X POST -H "Content-Type: application/json" \
  -d '{"method":"action.Create","params":[{"workload": {"namespace": "default", "apiVersion": "apps/v1", "kind": "Deployment", "name": "test"}, "node": {"name": "k3d-aces-agent-7"}}], "id":"1"}' \
  http://localhost:3000/rpc
```

## Clean up

``` bash
k3d cluster stop aces && k3d cluster delete aces
```

## To Read

- https://www.awelm.com/posts/kube-scheduler/
- https://cast.ai/blog/custom-kube-scheduler-why-and-how-to-set-it-up-in-kubernetes/
- https://scheduler-plugins.sigs.k8s.io/docs/user-guide/installation/
- https://kubernetes.io/docs/concepts/scheduling-eviction/scheduling-framework/
- https://medium.com/@juliorenner123/k8s-creating-a-kube-scheduler-plugin-8a826c486a1
