# terraform-provider-lattice

Terraform provider for [LatticeVE](../LatticeVE) — manages VMs, VPCs, storage, networking, and LatticeKube managed Kubernetes clusters.

## Requirements

- Terraform 1.5+
- LatticeVE controller reachable from where Terraform runs

## Provider Configuration

```hcl
terraform {
  required_providers {
    lattice = {
      source  = "latticeve/lattice"
      version = "~> 0.1"
    }
  }
}

provider "lattice" {
  endpoint = "https://lattice.local:8006"
  api_key  = var.lattice_api_key   # or set LATTICE_API_KEY env var
  insecure = true                  # skip TLS verification for self-signed certs
}
```

| Argument | Env var | Description |
|---|---|---|
| `endpoint` | `LATTICE_ENDPOINT` | Controller URL (required) |
| `api_key` | `LATTICE_API_KEY` | API key (optional if using session auth) |
| `insecure` | `LATTICE_INSECURE` | Skip TLS cert verification (default `true`) |

## Resources

| Resource | Description |
|---|---|
| `lattice_vm` | QEMU/KVM or Firecracker VM; `arch` for scheduler hint, `node` for hard pin, `image_id` for AMI-style boot disk |
| `lattice_vpc` | VPC with firewall rules and port forwards |
| `lattice_vpc_load_balancer` | VPC load balancer with TCP/HTTP/HTTPS frontends and weighted backends |
| `lattice_lb_certificate` | TLS certificate for HTTPS load balancers |
| `lattice_public_ip_pool` | Routable CIDR pool on a host NIC |
| `lattice_public_ip` | Allocate a public IP; optional static NAT |
| `lattice_storage_backend` | Register a storage backend (LVM, LINSTOR, NFS, Ceph, …) |
| `lattice_storage_volume` | Provision a block volume; grow-only resize |
| `lattice_kube_cluster` | Managed LatticeKube cluster — k3s control plane and workers on Firecracker microVMs |
| `lattice_k3s_kernel` | Import a Kubernetes-compatible Firecracker kernel from latticeve-k3s-images |
| `lattice_kernel_catalog_import` | Import a Kernel Catalog entry into the kernels table |
| `lattice_k3s_rootfs_image` | Import a pinned or latest k3s rootfs build from latticeve-k3s-images |
| `lattice_security_group` | Security group with ingress/egress rules |
| `lattice_vm_security_group` | Attach a security group to a VM |
| `lattice_ipam_pool` | DHCP pool for automatic VM IP assignment |
| `lattice_ipam_lease` | Static IP/MAC lease within an IPAM pool |
| `lattice_affinity_group` | Affinity or anti-affinity placement group |
| `lattice_vm_affinity_group` | Assign a VM to a placement group |

## Data Sources

| Data Source | Description |
|---|---|
| `lattice_vm` | Look up a VM by ID or name |
| `lattice_vpc` | Look up a VPC by ID or name |
| `lattice_image` | Look up a VM image by distro/version/arch for use as a boot disk (AMI-style) |
| `lattice_kernel` | Look up an already-imported Firecracker kernel by distro, `version`/`version_glob`, or `arch` |
| `lattice_kernel_catalog` | Browse kernels available to import (built-in entries plus Firecracker CI discovery) |
| `lattice_rootfs_image` | Look up a Firecracker rootfs image by name, arch, source, or version |
| `lattice_nodes` | List host nodes filtered by `arch` (`amd64`/`arm64`); returns capacity metrics |
| `lattice_kube_cluster` | Look up a cluster and retrieve endpoint, kubeconfig, image, and live node status |
| `lattice_public_ip_pools` | List all public IP pools |
| `lattice_storage_backends` | List all storage backends |

## Examples

### Basic QEMU VM in a VPC

```hcl
data "lattice_image" "debian" {
  name = "debian-12-generic-amd64"
}

resource "lattice_vpc" "main" {
  name = "main"
  cidr = "10.10.0.0/24"
}

resource "lattice_ipam_pool" "main" {
  name        = "main-pool"
  bridge      = lattice_vpc.main.bridge
  subnet      = "10.10.0.0/24"
  gateway     = "10.10.0.1"
  range_start = "10.10.0.10"
  range_end   = "10.10.0.200"
  dns         = ["1.1.1.1"]
}

resource "lattice_vm" "web" {
  name         = "web-01"
  cpus         = 2
  memory_mb    = 2048
  image_id     = data.lattice_image.debian.id
  boot_disk_gb = 20
  nics         = [{ bridge = lattice_vpc.main.bridge }]
}
```

### VPC Load Balancer with TLS

```hcl
resource "lattice_lb_certificate" "web" {
  name     = "web-cert"
  cert_pem = file("${path.module}/fullchain.pem")
  key_pem  = sensitive(file("${path.module}/privkey.pem"))
}

resource "lattice_vpc_load_balancer" "web" {
  vpc_id           = lattice_vpc.main.id
  name             = "web"
  port             = 443
  protocol         = "https"
  certificate_id   = lattice_lb_certificate.web.id
  backend_protocol = "http"

  backends = [
    {
      address = "10.10.0.10:8080"
      weight  = 100
    },
  ]
}
```

### ARM64 VM with arch-aware placement

```hcl
# Discover available arm64 nodes
data "lattice_nodes" "arm64" {
  arch = "arm64"
}

data "lattice_image" "ubuntu_arm" {
  distro  = "ubuntu"
  version = "26.04"
  arch    = "arm64"
}

resource "lattice_vm" "arm_worker" {
  name         = "arm-worker-01"
  cpus         = 8
  memory_mb    = 16384
  image_id     = data.lattice_image.ubuntu_arm.id
  boot_disk_gb = 50
  arch         = "arm64"   # scheduler picks any online arm64 node
  nics         = [{ bridge = lattice_vpc.main.bridge }]
}

output "placed_on" {
  value = lattice_vm.arm_worker.node  # resolved after apply
}
```

### Firecracker microVM

```hcl
data "lattice_kernel_catalog" "fc" {
  distro = "firecracker"
  arch   = "amd64"
}

resource "lattice_kernel_catalog_import" "fc" {
  entry_id = data.lattice_kernel_catalog.fc.id
}

resource "lattice_vm" "fc" {
  name      = "fc-01"
  vm_type   = "firecracker"
  kernel_id = lattice_kernel_catalog_import.fc.id
  cpus      = 2
  memory_mb = 512
  nics      = [{ bridge = lattice_vpc.main.bridge }]
}
```

### Managed Kubernetes Cluster (LatticeKube)

```hcl
resource "lattice_k3s_kernel" "kube" {
  arch = "amd64"
}

resource "lattice_k3s_rootfs_image" "release" {
  arch    = "amd64"
  version = "v1.36.2+k3s1-r23"
  lifecycle { create_before_destroy = true }
}

resource "lattice_public_ip_pool" "kube" {
  name      = "kube-pool"
  interface = "br0"
  cidr      = "10.0.7.128/28"
}

resource "lattice_kube_cluster" "prod" {
  name         = "prod"
  kernel_id    = lattice_k3s_kernel.kube.id
  rootfs_id    = lattice_k3s_rootfs_image.release.id
  pool_id      = lattice_public_ip_pool.kube.id
  cni          = "flannel"
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

Omit `kernel_id`/`rootfs_id` to use the controller's built-in k3s kernel and image instead. Scale workers with a plan/apply — no replacement needed:

```hcl
resource "lattice_kube_cluster" "prod" {
  # ...
  worker_count = 5  # was 3
}
```

To upgrade Kubernetes, change the rootfs `version` to the next supported release. The provider waits while LatticeVE snapshots etcd, rolls control planes, and drains/upgrades workers.

## API Coverage

The provider models persistent, declarative infrastructure: VMs, VPC policy and load balancers, public addressing, storage, Firecracker/k3s images, Kubernetes clusters, security-group relationships, IPAM leases, and affinity relationships. Controller operations such as console sessions, guest exec, migration internals, node drain commands, alert acknowledgement, and upgrade retry/force actions intentionally remain operational API/CLI workflows rather than Terraform resources.

## Full Examples

See the [`examples/`](examples/) directory:

- [`lattice_vm_basic/`](examples/lattice_vm_basic/) — VM with VPC and DHCP
- [`lattice_vm_advanced/`](examples/lattice_vm_advanced/) — VPC, firewall, NAT, security group, storage
- [`lattice_firecracker/`](examples/lattice_firecracker/) — Firecracker microVM with kernel catalog lookup
- [`lattice_kube/`](examples/lattice_kube/) — Full LatticeKube cluster with storage backend

## Documentation

See [`docs/resources/`](docs/resources/) and [`docs/data-sources/`](docs/data-sources/) for full argument and attribute references.
