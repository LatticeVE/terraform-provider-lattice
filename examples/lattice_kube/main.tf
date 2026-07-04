terraform {
  required_providers {
    lattice = {
      source  = "latticeve/lattice"
      version = "~> 0.1"
    }
  }
}

provider "lattice" {
  endpoint = var.lattice_endpoint
  api_key  = var.lattice_api_key
  insecure = true
}

variable "lattice_endpoint" {
  description = "LatticeVE controller URL"
  type        = string
  default     = "https://lattice.local:8006"
}

variable "lattice_api_key" {
  description = "LatticeVE API key"
  type        = string
  sensitive   = true
  default     = ""
}

variable "public_bridge" {
  description = "External Linux bridge reported by LatticeVE (for example br0 or vmbr0)"
  type        = string
  default     = "br0"
}

variable "public_pool_cidr" {
  description = "Reserved CIDR inside the external bridge subnet; exclude it from upstream DHCP"
  type        = string
}

# Import the latest Kubernetes-compatible kernel and k3s rootfs. Pin `version`
# in production when you need fully reproducible cluster builds.
resource "lattice_k3s_kernel" "latest" {
  arch = "amd64"
}

resource "lattice_k3s_rootfs_image" "latest" {
  arch = "amd64"

  lifecycle {
    create_before_destroy = true
  }
}

# Optional public IP pool backed by a routable subnet on the host NIC.
# LatticeVE always creates or uses a VPC for cluster nodes; this pool only
# exposes the cluster API and Kubernetes LoadBalancer services externally.
resource "lattice_public_ip_pool" "kube" {
  name      = "kube-pool"
  interface = var.public_bridge
  cidr      = var.public_pool_cidr
}

# Storage backend (LINSTOR for replicated volumes)
resource "lattice_storage_backend" "linstor" {
  name = "linstor-default"
  type = "linstor"
  config = {
    controller = "linstor://10.0.0.1:3370"
    pool_name  = "thinpool"
  }
}

# Managed LatticeKube cluster (k3s on Firecracker)
resource "lattice_kube_cluster" "prod" {
  name      = "prod"
  kernel_id = lattice_k3s_kernel.latest.id
  rootfs_id = lattice_k3s_rootfs_image.latest.id
  storage   = lattice_storage_backend.linstor.name
  # vpc_id  = lattice_vpc.existing.id # optional; omitted creates a managed VPC
  pool_id   = lattice_public_ip_pool.kube.id

  cni     = "flannel"
  lb_mode = "ccm"
  # Defaults to true. Disable only if you intentionally do not want Kubernetes
  # Metrics Server for LatticeVE workload CPU/memory views.
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

output "kube_endpoint" {
  description = "Kubernetes API server URL"
  value       = lattice_kube_cluster.prod.endpoint
}

# Human kubeconfigs are short-lived and role-scoped. Download one from the
# LatticeVE UI after apply; API-key credentials are not written to state.

output "cluster_images" {
  value = {
    kernel = lattice_k3s_kernel.latest.version
    rootfs = lattice_k3s_rootfs_image.latest.version
  }
}
