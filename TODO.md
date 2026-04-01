# To-Do: Research & Test

## Service mesh options (besides Istio)

- [ ] **AWS App Mesh** – Research and test on EKS (managed, Envoy-based).
- [ ] **Linkerd** – Research and test on EKS and/or GKE (lightweight, mTLS + observability).
- [ ] **Consul Connect** – Research and test on EKS and/or GKE (HashiCorp, discovery + mesh).
- [ ] **Cilium Service Mesh** – Research and test on EKS and/or GKE (eBPF + Envoy; aligns with existing Cilium CNI).
- [ ] **Anthos Service Mesh (ASM)** – Research and test on GKE (managed Istio).
- [ ] **Traffic Director** – Research and test on GCP (managed L4/L7 traffic control; often used with mesh).

## Node autoscaling

- [ ] **Karpenter (EKS)** – Research and test on AWS EKS (provision nodes for unschedulable pods; instance selection, consolidation).
- [ ] **Karpenter (GKE)** – Research and test on GKE using Karpenter’s GKE provider.
- [ ] **GKE Autopilot** – Evaluate for fully managed nodes (no node-pool management).
- [ ] **GKE Cluster Autoscaler** – Research and test standard GKE node-pool autoscaling.

## Policy & security

- [ ] **Kyverno** or **OPA Gatekeeper** – Policy-as-code (admission control, image validation, compliance).
- [ ] **Trivy** / **Trivy Operator** – Image and manifest scanning in CI or in-cluster.
- [ ] **Cilium Tetragon** – eBPF-based runtime security and observability (complements Cilium CNI); compare overlap with **Falco** ([`argocd/applications/falco.yaml`](argocd/applications/falco.yaml)).

## Observability (beyond Prometheus/Grafana)

- [ ] **Cilium Hubble** – Enable flow observability/UI aligned with existing Cilium CNI (complements metrics-only views).
- [ ] **OpenTelemetry** – Unified traces, metrics, logs; OTLP collectors and instrumentation.
- [ ] **Loki** or **structured logging** – Centralized log aggregation and querying.

## Serverless & event-driven

- [ ] **KEDA** – Event-driven autoscaling (scale deployments from queues, HTTP, cron, etc.; scale to zero).

## Platform & developer experience

- [ ] **GKE feature parity** – Research and implement GKE equivalents for EKS features (e.g. Fargate ↔ GKE Autopilot, IRSA ↔ Workload Identity, ALB ↔ GCLB).

- [ ] **Backstage** or **internal developer portal** – Service catalog, docs, and self-service for teams.
- [ ] **Kubecost** / **OpenCost** – Cost visibility and allocation per namespace, deployment, or label.

## Production foundation (DNS, TLS, backup)

- [ ] **ExternalDNS** – Sync Gateway API / HTTPRoute hostnames to **Route53** (EKS) or **Cloud DNS** (GKE) for stable production hostnames.
- [ ] **HTTPS on Gateway API** – **cert-manager** `Certificate` resources + TLS listeners on the Gateway (and DNS validation); extend beyond HTTP-only [`gateway.yaml`](kubernetes/apps/gateway.yaml).
- [ ] **Velero** – Cluster backup and restore (namespaces, PVs where applicable); validate DR story on EKS and GKE.

## Resilience & chaos

- [ ] **Chaos Mesh** or **Litmus** – Chaos engineering (pod kill, network delay, node failure) for resilience testing.
