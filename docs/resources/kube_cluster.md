# lattice_kube_cluster

Provisions a LatticeKube cluster — a k3s control plane and workers running as Firecracker microVMs on LatticeVE. Creation is asynchronous — the resource polls until the cluster reaches `ready` or `failed` status (30-minute timeout).

Worker count and control-plane count can be scaled in place. Control-plane counts must remain `1`, `3`, or `5`, and only scale-out is supported. Kubernetes upgrades are performed by changing `rootfs_id` to a newer, upgrade-compatible `lattice_k3s_rootfs_image`; the provider waits for the controller's rolling control-plane and worker upgrade to finish.

## Example Usage

```hcl
resource "lattice_k3s_kernel" "kube" {
  arch = "amd64"
  # version = "6.1.175" # optional production pin
}

resource "lattice_k3s_rootfs_image" "release" {
  arch    = "amd64"
  version = "v1.36.2+k3s1-r23"

  lifecycle {
    create_before_destroy = true
  }
}

resource "lattice_public_ip_pool" "kube" {
  name      = "kube-pool"
  interface = "br0"
  cidr      = "10.0.7.128/28"
}

resource "lattice_kube_cluster" "prod" {
  name      = "prod"
  kernel_id = lattice_k3s_kernel.kube.id
  rootfs_id = lattice_k3s_rootfs_image.release.id
  pool_id   = lattice_public_ip_pool.kube.id
  cni       = "flannel"
  lb_mode   = "ccm"
  metrics_server = true

  cp_count     = 3
  cp_vcpus     = 4
  cp_memory_mb = 8192
  cp_disk_gb   = 50

  worker_count     = 3
  worker_vcpus     = 8
  worker_memory_mb = 16384
  worker_disk_gb   = 100
}

```

The provider API key cannot retrieve a human kubeconfig. Download a short-lived,
role-scoped kubeconfig from the LatticeVE UI after apply.

The pool CIDR must be reserved inside the selected external bridge's connected IPv4 subnet and excluded from upstream DHCP. The cluster allocates and manages its own API endpoint address from `pool_id`; do not create a separate `lattice_public_ip` for it.

## Scale-out Example

```hcl
resource "lattice_kube_cluster" "prod" {
  # ... other arguments unchanged ...
  worker_count = 5  # was 3 — adding 2 workers
}
```

Control-plane scale-out uses the same resource:

```hcl
resource "lattice_kube_cluster" "prod" {
  # ... other arguments unchanged ...
  cp_count = 3 # supported transitions: 1 -> 3 -> 5
}
```

## Upgrade Example

Change the pinned rootfs image to the next supported Kubernetes patch or minor release and apply:

```hcl
resource "lattice_k3s_rootfs_image" "release" {
  arch    = "amd64"
  version = "v1.36.2+k3s1-r23"

  lifecycle {
    create_before_destroy = true
  }
}

resource "lattice_kube_cluster" "prod" {
  # ...
  rootfs_id = lattice_k3s_rootfs_image.release.id
}
```

LatticeVE validates the upgrade path, snapshots etcd, upgrades HA control planes one at a time with an etcd/API health check between nodes, then drains, upgrades, and uncordons workers. Terraform waits for `ready`; `failed` and `upgrade_blocked` are returned as apply errors while preserving the cluster for operator recovery.

## Argument Reference

- `name` (Required, Forces new resource) — Cluster name.
- `runtime` (Optional, Computed, Forces new resource) — VM backend for cluster nodes. Only `firecracker` is supported.
- `kernel_id` (Required, Forces new resource) — Kubernetes-compatible Firecracker kernel UUID, normally from `lattice_k3s_kernel`.
- `rootfs_id` (Optional, Computed) — Rootfs image UUID from `lattice_rootfs_image` or `lattice_k3s_rootfs_image`. Changing it invokes the safe in-place upgrade/revision workflow.
- `storage` (Optional, Computed, Forces new resource) — Named storage backend for cluster VM disks. Empty uses the default backend.
- `k8s_version` (Optional, Computed) — Kubernetes version, e.g. `v1.32.0`. Inferred from the rootfs image's name/description when omitted.
- `cni` (Optional, Computed, Forces new resource) — CNI plugin: `flannel`, `cilium`, or `none`.
- `lb_mode` (Optional, Computed, Forces new resource) — Load-balancer mode: `ccm`, `metallb`, or `cilium`.
- `metrics_server` (Optional, Computed, Forces new resource) — Enables the bundled Kubernetes Metrics Server for live CPU/memory in LatticeVE workload views. Defaults to `true`; set `false` to bootstrap the cluster with `--disable=metrics-server`.
- `pool_id` (Optional, Forces new resource) — Public IP pool ID for the control-plane floating IP.
- `vpc_id` (Optional, Computed, Forces new resource) — Existing VPC UUID; when omitted LatticeVE creates a managed VPC.
- `root_password_hash` (Optional, Sensitive, Forces new resource) — crypt(3) root password hash for cluster nodes.
- `ssh_authorized_keys` (Optional, Forces new resource) — public SSH keys installed on cluster nodes.
- `cp_count` (Optional, Computed) — Control-plane node count. Must be 1, 3, or 5. Scale-out is in place; scale-down is rejected.
- `worker_count` (Optional, Computed) — Worker node count. **Updatable** — increase to scale out, decrease to scale in.
- `cp_vcpus` (Optional, Computed, Forces new resource) — vCPUs per control-plane node.
- `cp_memory_mb` (Optional, Computed, Forces new resource) — Memory per control-plane node in MiB.
- `cp_disk_gb` (Optional, Computed, Forces new resource) — Boot disk per control-plane node in GiB.
- `worker_vcpus` (Optional, Computed, Forces new resource) — vCPUs per worker node.
- `worker_memory_mb` (Optional, Computed, Forces new resource) — Memory per worker node in MiB.
- `worker_disk_gb` (Optional, Computed, Forces new resource) — Boot disk per worker node in GiB.

## Attribute Reference

- `id` — Cluster UUID.
- `kernel_version` — Linux kernel version used by cluster nodes.
- `status` — Cluster lifecycle: `provisioning`, `ready`, `failed`, `deleting`.
- `endpoint` — Kubernetes API server URL.
- `public_ip` — Allocated public IP for the control plane.
- `vpc_id` — Cluster VPC UUID.
- `vpc_cidr` — CIDR assigned to the cluster VPC.
- `vpc_managed` — Whether LatticeVE owns and deletes the cluster VPC.
- `oidc_enabled` — Whether role-scoped Kubernetes credentials are enabled.
- `metrics_server` — Whether the cluster was bootstrapped with the bundled Metrics Server enabled.
- `kubeconfig` (Deprecated, always null) — human credentials are intentionally excluded from Terraform state.
- `nodes` — Cluster nodes with `id`, `vm_id`, `name`, `role`, `ip`, `status`, live `kubelet_version`, and any node-specific `upgrade_error`.
