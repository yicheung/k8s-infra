# K8s Infrastructure (Pulumi + Go) — Multi-cloud

Provisions **AWS EKS** or **Google GKE** from a single codebase. Set `k8s-infra:cloud` to `aws` or `gcp`. Same Kubernetes stack (Cilium, Gateway API, ArgoCD) runs on either cluster.

## Prerequisites

- [Pulumi CLI](https://www.pulumi.com/docs/install/)
- [Go](https://go.dev/dl/) 1.21+
- **AWS (EKS):** AWS credentials (`aws configure` or env vars)
- **GCP (GKE):** `gcloud auth application-default login` and project set

## Config: choose cloud

| Config key | Description | Default |
|------------|-------------|---------|
| `k8s-infra:cloud` | `aws` (EKS) or `gcp` (GKE) | `aws` |

### AWS (EKS)

| Config key | Description | Default |
|------------|-------------|---------|
| `aws:region` | AWS region | required |
| `k8s-infra:clusterName` | EKS cluster name | `k8s-infra-cluster` |
| `k8s-infra:clusterVersion` | Kubernetes version | `1.29` |
| `k8s-infra:nodeInstanceType` | Node instance type | `t3.medium` |
| `k8s-infra:nodeDesiredSize` | Desired node count | `2` |
| `k8s-infra:vpcCidr` | VPC CIDR | `10.0.0.0/16` |
| `k8s-infra:enableFargate` | Enable Fargate profile for `taskapp` namespace | `false` |

### GCP (GKE)

| Config key | Description | Default |
|------------|-------------|---------|
| `gcp:project` or `k8s-infra:gcpProject` | GCP project ID | required for GKE |
| `gcp:region` or `k8s-infra:gcpRegion` | GCP region (e.g. `us-central1`) | `us-central1` |
| `k8s-infra:clusterName` | GKE cluster name | `k8s-infra-cluster` |
| `k8s-infra:clusterVersion` | Kubernetes version | `1.29` |
| `k8s-infra:nodeMachineType` | Node machine type | `e2-medium` |
| `k8s-infra:nodeDesiredSize` | Initial node count | `2` |

## Stacks

- **dev (EKS):** `Pulumi.dev.yaml` — `cloud: aws`
- **gke (GKE):** `Pulumi.gke.yaml` — `cloud: gcp`; set `gcp:project` (or `k8s-infra:gcpProject`) to your project ID

Create and use the GKE stack:

```bash
pulumi stack init gke   # once
# Copy config from Pulumi.gke.yaml or set:
pulumi config set k8s-infra:cloud gcp
pulumi config set gcp:project YOUR_PROJECT_ID
pulumi config set gcp:region us-central1
pulumi up
```

## Build and deploy

```bash
cd pulumi
go mod tidy
pulumi preview
pulumi up
```

## Configure kubectl

**AWS EKS:**

```bash
pulumi stack output updateKubeconfigCommand
# Run the printed command, e.g.:
# aws eks update-kubeconfig --region eu-west-1 --name k8s-infra-cluster
```

**GCP GKE:**

```bash
pulumi stack output getCredentialsCommand
# Run the printed command, e.g.:
# gcloud container clusters get-credentials k8s-infra-cluster --region us-central1 --project my-project
```

## Outputs

| Output | AWS (EKS) | GCP (GKE) |
|--------|-----------|-----------|
| `cloud` | `aws` | `gcp` |
| `clusterName` | ✓ | ✓ |
| `clusterEndpoint` | ✓ | ✓ |
| `vpcId` | ✓ | — |
| `updateKubeconfigCommand` | ✓ | — |
| `getCredentialsCommand` | — | ✓ |

## After the cluster is up

1. Install **Gateway API CRDs** and **Cilium** (see main [README](../README.md) Quick Start).
2. Deploy platform (cert-manager, External Secrets Operator, ArgoCD, etc.) and apps — same for EKS and GKE. Secrets live in AWS Secrets Manager or GCP Secret Manager.

## Destroy

```bash
pulumi destroy
```
