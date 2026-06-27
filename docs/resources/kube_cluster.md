# lattice_kube_cluster

Provisions a managed Kubernetes cluster running Talos Linux on LatticeVE. Creation is asynchronous — the resource polls until the cluster reaches `ready` or `failed` status (30-minute timeout).

Worker count, Talos version, and Kubernetes version can be updated in-place. All other arguments require cluster replacement.

## Example Usage

```hcl
data "lattice_kube_releases" "all" {}

resource "lattice_public_ip_pool" "kube" {
  name      = "kube-pool"
  interface = "eth0"
  cidr      = "192.168.100.128/27"
}

resource "lattice_kube_cluster" "prod" {
  name          = "prod"
  talos_image   = "/var/lib/lattice/images/talos-v1.9.0-metal-amd64.raw"
  talos_version = data.lattice_kube_releases.all.releases[0].version
  k8s_version   = data.lattice_kube_releases.all.releases[0].k8s_version
  pool_id       = lattice_public_ip_pool.kube.id
  cni           = "cilium"
  lb_mode       = "cilium"

  cp_count     = 3
  cp_vcpus     = 4
  cp_memory_mb = 8192
  cp_disk_gb   = 50

  worker_count     = 3
  worker_vcpus     = 8
  worker_memory_mb = 16384
  worker_disk_gb   = 100
}

output "kubeconfig" {
  value     = lattice_kube_cluster.prod.kubeconfig
  sensitive = true
}
```

## Scale-out Example

```hcl
resource "lattice_kube_cluster" "prod" {
  # ... other arguments unchanged ...
  worker_count = 5  # was 3 — adding 2 workers
}
```

## Upgrade Example

```hcl
resource "lattice_kube_cluster" "prod" {
  # ... other arguments unchanged ...
  talos_version = "v1.9.1"
  k8s_version   = "v1.32.2"
}
```

Version downgrades are rejected by the API.

## Argument Reference

- `name` (Required, Forces new resource) — Cluster name.
- `talos_image` (Required, Forces new resource) — Path to the Talos metal disk image on the LatticeVE host.
- `talos_version` (Optional, Computed) — Talos version tag, e.g. `v1.9.0`. Upgradeable.
- `k8s_version` (Optional, Computed) — Kubernetes version, e.g. `v1.32.0`. Upgradeable.
- `cni` (Optional, Computed, Forces new resource) — CNI plugin: `flannel`, `cilium`, or `none`.
- `lb_mode` (Optional, Computed, Forces new resource) — Load-balancer mode: `ccm`, `metallb`, or `cilium`.
- `pool_id` (Optional, Forces new resource) — Public IP pool ID for the control-plane floating IP.
- `cp_count` (Optional, Computed, Forces new resource) — Control-plane node count (odd numbers recommended).
- `worker_count` (Optional, Computed) — Worker node count. **Updatable** — increase to scale out, decrease to scale in.
- `cp_vcpus` (Optional, Computed, Forces new resource) — vCPUs per control-plane node.
- `cp_memory_mb` (Optional, Computed, Forces new resource) — Memory per control-plane node in MiB.
- `cp_disk_gb` (Optional, Computed, Forces new resource) — Boot disk per control-plane node in GiB.
- `worker_vcpus` (Optional, Computed, Forces new resource) — vCPUs per worker node.
- `worker_memory_mb` (Optional, Computed, Forces new resource) — Memory per worker node in MiB.
- `worker_disk_gb` (Optional, Computed, Forces new resource) — Boot disk per worker node in GiB.

## Attribute Reference

- `id` — Cluster UUID.
- `status` — Cluster lifecycle: `provisioning`, `ready`, `failed`, `deleting`.
- `endpoint` — Kubernetes API server URL.
- `public_ip` — Allocated public IP for the control plane.
- `vpc_id` — VPC UUID created for this cluster.
- `vpc_cidr` — CIDR assigned to the cluster VPC.
- `kubeconfig` (Sensitive) — kubeconfig YAML for `kubectl` access.
- `talosconfig` (Sensitive) — talosconfig YAML for `talosctl` access.
- `nodes` — List of cluster nodes, each with `id`, `vm_id`, `role`, `ip`, `status`.
