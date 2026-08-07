#!/usr/bin/env bash
set -euo pipefail

REGISTRY="registry.apps.engineering.openshift.org"
IMAGE="${REGISTRY}/changelog-controller/changelog-client:latest"
AUTHFILE="/tmp/oc-registry-auth.json"
NAMESPACE="changelog-controller"
DOCKERFILE="images/changelog-client/Dockerfile.dev"

cd "$(git rev-parse --show-toplevel)"

echo "==> Cross-compiling binary for linux/amd64..."
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o changelog-client ./cmd/changelog-client

echo "==> Building container image..."
podman build --platform linux/amd64 --authfile "${AUTHFILE}" -t "${IMAGE}" -f "${DOCKERFILE}" .

echo "==> Pushing image to registry..."
podman push --authfile "${AUTHFILE}" "${IMAGE}"

echo "==> Applying manifests..."
oc apply -f manifests/changelog-client/

echo "==> Restarting deployment..."
oc rollout restart deployment/changelog-client -n "${NAMESPACE}"

echo "==> Waiting for rollout..."
oc rollout status deployment/changelog-client -n "${NAMESPACE}" --timeout=120s

echo "==> Cleaning up local binary..."
rm -f changelog-client

echo "==> Done! Deployment updated."
