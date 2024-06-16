# Workload Actions Manager (WAM)

## Deploy
- https://cast.ai/blog/custom-kube-scheduler-why-and-how-to-set-it-up-in-kubernetes/
- https://scheduler-plugins.sigs.k8s.io/docs/user-guide/installation/
- https://kubernetes.io/docs/concepts/scheduling-eviction/scheduling-framework/
- https://medium.com/@juliorenner123/k8s-creating-a-kube-scheduler-plugin-8a826c486a1

### Requirements

- [Docker](https://docs.docker.com/get-docker/)
- [Kubectl](https://kubernetes.io/docs/tasks/tools/#kubectl)
- [K3d](https://k3d.io/v5.2.2/#installation)

### Create a Cluster

[//]: # (todo: --volume to mount the kubeconfig to server node, then --k3s-arg '--kube-scheduler-arg=config=/var/lib/scheduler/scheduler-config.yaml')
```bash
grep k3d-registry.localhost /etc/hosts || echo "127.0.0.1 k3d-registry.localhost" | sudo tee -a /etc/hosts
k3d cluster create --config deploy/k3d.yaml
```

kubectl apply -f deploy/wam-scheduler
kubectl apply -f deploy/test

### Clean up

``` bash
k3d cluster stop aces && k3d cluster delete aces
```
