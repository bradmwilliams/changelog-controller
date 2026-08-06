#!/usr/bin/env bash
set -euo pipefail

REGISTRY="registry.apps.engineering.openshift.org"
IMAGE="${REGISTRY}/changelog-controller/changelog-controller:latest"
AUTHFILE="/tmp/oc-registry-auth.json"
NAMESPACE="changelog-controller"
DOCKERFILE="images/changelog-controller/Dockerfile.dev"

cd "$(git rev-parse --show-toplevel)"

echo "==> Cross-compiling binary for linux/amd64..."
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o changelog-controller ./cmd/changelog-controller

echo "==> Building container image..."
podman build --platform linux/amd64 --authfile "${AUTHFILE}" -t "${IMAGE}" -f "${DOCKERFILE}" .

echo "==> Pushing image to registry..."
podman push --authfile "${AUTHFILE}" "${IMAGE}"

echo "==> Restarting deployment..."
oc rollout restart deployment/changelog-controller -n "${NAMESPACE}"

echo "==> Waiting for rollout..."
oc rollout status deployment/changelog-controller -n "${NAMESPACE}" --timeout=120s

echo "==> Cleaning up local binary..."
rm -f changelog-controller

echo "==> Done! Deployment updated."
