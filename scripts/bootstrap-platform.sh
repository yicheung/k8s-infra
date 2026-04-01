#!/usr/bin/env bash
# Bootstrap EKS (AWS) or GKE (GCP): Pulumi cluster + Cilium + Gateway API + platform stack.
# Prerequisites: Pulumi, Go, kubectl, Helm; cloud credentials (aws configure or gcloud auth).
#
# Usage:
#   ./scripts/bootstrap-platform.sh aws              # stack dev → EKS + platform
#   ./scripts/bootstrap-platform.sh gcp              # stack gke → GKE + platform
#   ./scripts/bootstrap-platform.sh aws cluster      # only Pulumi + kubeconfig
#   ./scripts/bootstrap-platform.sh aws platform     # only Helm/kubectl (kubectl must work)
#
# GKE: ensure stack `gke` exists and gcp:project is set (see pulumi/README.md).

set -euo pipefail

CLOUD="${1:-}"
STEP="${2:-all}"

usage() {
  echo "Usage: $0 <aws|gcp> [cluster|platform|all]" >&2
  echo "  aws  — Pulumi stack: dev (EKS)" >&2
  echo "  gcp  — Pulumi stack: gke (GKE)" >&2
  echo "  cluster  — only create cluster and configure kubectl" >&2
  echo "  platform — only install Cilium, cert-manager, ESO, Prometheus, Argo CD" >&2
  echo "  all      — default: cluster + platform" >&2
  exit 1
}

case "${CLOUD}" in
  aws)
    STACK="dev"
    KUBECONFIG_OUTPUT="updateKubeconfigCommand"
    ;;
  gcp)
    STACK="gke"
    KUBECONFIG_OUTPUT="getCredentialsCommand"
    ;;
  *) usage ;;
esac

case "${STEP}" in
  cluster | platform | all) ;;
  *) usage ;;
esac

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${ROOT}/pulumi"

run_cluster() {
  echo "==> Pulumi stack: ${STACK} (${CLOUD})"
  go mod tidy
  pulumi stack select "${STACK}"
  pulumi up

  echo "==> kubectl credentials"
  CMD="$(pulumi stack output "${KUBECONFIG_OUTPUT}")"
  if [[ -z "${CMD}" ]]; then
    echo "ERROR: empty stack output ${KUBECONFIG_OUTPUT}" >&2
    exit 1
  fi
  eval "${CMD}"
  kubectl cluster-info
}

run_platform() {
  echo "==> Helm repos"
  helm repo add cilium https://helm.cilium.io/ 2>/dev/null || true
  helm repo add external-secrets https://charts.external-secrets.io 2>/dev/null || true
  helm repo add prometheus-community https://prometheus-community.github.io/helm-charts 2>/dev/null || true
  helm repo update

  echo "==> Cilium (Gateway API)"
  helm upgrade --install cilium cilium/cilium --namespace kube-system \
    --set gatewayAPI.enabled=true

  echo "==> Gateway API CRDs (standard channel)"
  kubectl apply -f https://github.com/kubernetes-sigs/gateway-api/releases/download/v1.0.0/standard-install.yaml

  echo "==> cert-manager"
  kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.13.0/cert-manager.yaml

  echo "==> External Secrets Operator"
  helm upgrade --install external-secrets external-secrets/external-secrets \
    -n external-secrets-system --create-namespace

  echo "==> kube-prometheus-stack"
  helm upgrade --install prometheus prometheus-community/kube-prometheus-stack \
    -n monitoring --create-namespace

  echo "==> Argo CD"
  kubectl create namespace argocd 2>/dev/null || true
  kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml

  echo "==> Done. Next: cloud secrets + kubectl apply -f argocd/applications/taskapp.yaml (see README)"
}

if [[ "${STEP}" == "cluster" || "${STEP}" == "all" ]]; then
  run_cluster
fi

if [[ "${STEP}" == "platform" || "${STEP}" == "all" ]]; then
  run_platform
fi
