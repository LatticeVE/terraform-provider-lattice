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
| `lattice_kube_cluster` | Managed Talos Linux Kubernetes cluster |
| `lattice_security_group` | Security group with ingress/egress rules |
| `lattice_ipam_pool` | DHCP pool for automatic VM IP assignment |

## Data Sources

| Data Source | Description |
|---|---|
| `lattice_vm` | Look up a VM by ID or name |
| `lattice_vpc` | Look up a VPC by ID or name |
| `lattice_image` | Look up a VM image by distro/version/arch for use as a boot disk (AMI-style) |
| `lattice_kernel` | Look up a Firecracker kernel by distro, `distro_version`, or `version_glob` |
| `lattice_nodes` | List host nodes filtered by `arch` (`amd64`/`arm64`); returns capacity metrics |
| `lattice_kube_releases` | List available Talos releases |
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
data "lattice_kernel" "alpine" {
  distro         = "alpine"
  distro_version = "3.24.1"
}

resource "lattice_vm" "fc" {
  name      = "fc-01"
  vm_type   = "firecracker"
  kernel_id = data.lattice_kernel.alpine.id
  cpus      = 2
  memory_mb = 512
  nics      = [{ bridge = lattice_vpc.main.bridge }]
}
```

### Managed Kubernetes Cluster (LatticeKube)

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
  cp_count      = 3
  cp_vcpus      = 4
  cp_memory_mb  = 8192
  cp_disk_gb    = 50
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

Scale workers or upgrade versions with a plan/apply — no replacement needed:

```hcl
resource "lattice_kube_cluster" "prod" {
  # ...
  worker_count  = 5          # was 3
  talos_version = "v1.9.1"  # upgrade
  k8s_version   = "v1.32.2"
}
```

## Full Examples

See the [`examples/`](examples/) directory:

- [`lattice_vm_basic/`](examples/lattice_vm_basic/) — VM with VPC and DHCP
- [`lattice_vm_advanced/`](examples/lattice_vm_advanced/) — VPC, firewall, NAT, security group, storage
- [`lattice_firecracker/`](examples/lattice_firecracker/) — Firecracker microVM with kernel catalog lookup
- [`lattice_kube/`](examples/lattice_kube/) — Full LatticeKube cluster with storage backend

## Documentation

See [`docs/resources/`](docs/resources/) and [`docs/data-sources/`](docs/data-sources/) for full argument and attribute references.
