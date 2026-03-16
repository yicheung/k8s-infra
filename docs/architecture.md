# Architecture: Cilium, Gateway API, EKS/GKE

This document describes the modern stack used in k8s-infra: **managed Kubernetes (EKS or GKE)**, **Cilium** (CNI + Gateway API), and **Gateway API** for ingress.

## Overview

| Layer | Technology | Role |
|-------|------------|------|
| **Cloud** | AWS EKS or Google GKE | Managed control plane and node lifecycle |
| **Networking** | Cilium | CNI (eBPF), network policies, Gateway API implementation |
| **Ingress** | Gateway API (HTTPRoute, Gateway) | Standard L7 routing; Cilium implements the controller |
| **Load balancing** | Cloud provider | EKS: ALB/NLB from Cilium Gateway; GKE: GCP Load Balancer |

Legacy components **not** used:

- **MetalLB** – Not needed; cloud LBs are used.
- **Legacy Ingress controllers** – Replaced by **Gateway API** (Cilium as provider).
- **Calico** – Replaced by **Cilium** (CNI + policies).
- **kubeadm / self-managed nodes** – Replaced by **EKS / GKE**.

## Data flow

1. **Pulumi** provisions EKS (or GKE): VPC, subnets, cluster, managed node group.
2. **Cilium** is installed (Helm) with `gatewayAPI.enabled=true`; **Gateway API CRDs** are applied.
3. A **Gateway** (e.g. `taskapp-gateway`) and **HTTPRoute**(s) define how traffic reaches services.
4. Cilium programs the cloud load balancer (e.g. AWS ALB) and routes traffic to the correct Services.

## Gateway API in this repo

- **Gateway** (`kubernetes/apps/gateway.yaml`): `gatewayClassName: cilium`, HTTP listener on port 80.
- **HTTPRoute** (`kubernetes/apps/httproute.yaml`): Attached to that Gateway; `/api` → backend:8000, `/` → frontend:80.

No legacy Ingress resources or ingressClassName-specific controllers; everything uses Gateway API resources.

## Multi-cloud (EKS and GKE)

One Pulumi program in `pulumi/main.go` supports both clouds. Set **`k8s-infra:cloud`** to:

- **`aws`** – Provisions VPC + EKS cluster + node group. Output: `updateKubeconfigCommand` → run `aws eks update-kubeconfig ...`.
- **`gcp`** – Provisions GKE cluster + node pool. Requires `gcp:project` (or `k8s-infra:gcpProject`). Output: `getCredentialsCommand` → run `gcloud container clusters get-credentials ...`.

Use stack **dev** for EKS (`Pulumi.dev.yaml`) or **gke** for GKE (`Pulumi.gke.yaml`). Same Kubernetes manifests (Cilium, Gateway API, ArgoCD) apply to either cluster.

## Security and secrets

- **Secrets** are stored in the cloud provider: **AWS Secrets Manager** (EKS) or **GCP Secret Manager** (GKE). No HashiCorp Vault.
- **External Secrets Operator** syncs from AWS/GCP into Kubernetes Secrets (e.g. `taskapp-secrets`). Use IRSA (EKS) or Workload Identity (GKE) for production.
- **Network policies** can be expressed with Cilium CiliumNetworkPolicy or standard NetworkPolicy.

## Observability

- **Prometheus** and **Grafana** for metrics and dashboards.
- Cilium provides Hubble for flow observability (optional).
