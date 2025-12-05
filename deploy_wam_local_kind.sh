#!/usr/bin/env zsh
set -euo pipefail

# Config
CLUSTER_NAME=${CLUSTER_NAME:-sample}
LOAD_TO_KIND=${LOAD_TO_KIND:-true}     # set to 'false' to skip kind load

ALPINE_BASE_IMAGE=${ALPINE_BASE_IMAGE:-alpine}
ARCH=${ARCH:-amd64}
RELEASE_VERSION=${RELEASE_VERSION:-v0.0.0}
NAMESPACE=${NAMESPACE:-ul}

WAM_SCHEDULER_IMAGE_NAME="wam-scheduler"
WAM_SCHEDULER_IMAGE_TAG="latest"
WAM_IMAGE_NAME="wam"
WAM_IMAGE_TAG="latest"

WAM_CHARTS_DIR="./deploy/wam"
WAM_SCHEDULER_CHARTS_DIR="./deploy/wam-scheduler"

echo "Set the kubectl context to $CLUSTER_NAME cluster"
kubectl cluster-info --context "kind-${CLUSTER_NAME}" || {
  echo "Context kind-${CLUSTER_NAME} not found. Proceeding; kind load will fail if cluster isn't available."
}
kubectl config use-context "kind-${CLUSTER_NAME}" || true

#Build Scheduler image
echo "Building Scheduler image..."
docker build --platform=linux/arm64 \
  -t $WAM_SCHEDULER_IMAGE_NAME:$WAM_SCHEDULER_IMAGE_TAG \
  -f ./wam-scheduler/build/scheduler/Dockerfile ./ \
  --build-arg ALPINE_BASE_IMAGE=$ALPINE_BASE_IMAGE \
  --build-arg ARCH=$ARCH \
  --build-arg RELEASE_VERSION=$RELEASE_VERSION      

# Load WAM-scheduler image into kind
if [[ "${LOAD_TO_KIND}" == "true" ]]; then
  echo "Load Scheduler image to Kind cluster named '$CLUSTER_NAME'"
  kind load docker-image --name "$CLUSTER_NAME" $WAM_SCHEDULER_IMAGE_NAME:$WAM_SCHEDULER_IMAGE_TAG
fi

# Build WAM image
echo "Building WAM image..."
docker build -t $WAM_IMAGE_NAME:$WAM_IMAGE_TAG -f ./wam/Dockerfile .

# Load WAM image into Kind
if [[ "${LOAD_TO_KIND}" == "true" ]]; then
  echo "Loading WAM image into Kind cluster named '$CLUSTER_NAME'..."
  kind load docker-image --name "$CLUSTER_NAME" $WAM_IMAGE_NAME:$WAM_IMAGE_TAG
fi

# Deploy Redis via helm
echo "Deploying Redis via Helm..."
helm upgrade --install --create-namespace \
  --namespace $NAMESPACE wam-redis deploy/redis

# Deploy WAM service via helm
echo "Deploying WAM via Helm..."
helm upgrade --install --create-namespace \
  --namespace $NAMESPACE \
  wam-app $WAM_CHARTS_DIR \
  --set image.repository=$WAM_IMAGE_NAME \
  --set image.tag=$WAM_IMAGE_TAG \
  --set image.pullPolicy=IfNotPresent

# Deploy WAM Scheduler via helm
echo "Deploying WAM Scheduler via Helm..."
helm upgrade --install --create-namespace \
  --namespace $NAMESPACE \
  wam-scheduler $WAM_SCHEDULER_CHARTS_DIR \
  --set image.repository=$WAM_SCHEDULER_IMAGE_NAME \
  --set image.tag=$WAM_SCHEDULER_IMAGE_TAG \
  --set image.pullPolicy=IfNotPresent

echo "Done."