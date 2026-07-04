# lattice_kube_cluster

Looks up an existing LatticeKube cluster by `id` or `name` and exposes its endpoint, image versions, VPC/OIDC ownership state, Metrics Server setting, node counts, and live per-node kubelet/upgrade status. Every cluster has a VPC: either an existing `vpc_id` selected at creation time or a LatticeVE-managed VPC. A public IP is present only when the cluster was created with a public IP pool. Human kubeconfigs are short-lived and intentionally unavailable to provider API keys.

```hcl
data "lattice_kube_cluster" "prod" {
  name = "prod"
}

output "endpoint" {
  value = data.lattice_kube_cluster.prod.endpoint
}
```
