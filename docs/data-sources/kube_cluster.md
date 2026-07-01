# lattice_kube_cluster

Looks up an existing LatticeKube cluster by `id` or `name` and exposes its endpoint, sensitive kubeconfig, image versions, node counts, and live per-node kubelet/upgrade status.

```hcl
data "lattice_kube_cluster" "prod" {
  name = "prod"
}

output "endpoint" {
  value = data.lattice_kube_cluster.prod.endpoint
}
```
