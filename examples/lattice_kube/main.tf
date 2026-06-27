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

# Look up available Talos releases to pin a specific version
data "lattice_kube_releases" "all" {}

locals {
  # Pick the latest stable Talos release
  talos_version = data.lattice_kube_releases.all.releases[0].version
  k8s_version   = data.lattice_kube_releases.all.releases[0].k8s_version
}

# Public IP pool backed by a routable subnet on the host NIC
resource "lattice_public_ip_pool" "kube" {
  name      = "kube-pool"
  interface = "eth0"
  cidr      = "192.168.100.128/27"
}

# Allocate a floating IP for the control-plane endpoint
resource "lattice_public_ip" "cp_endpoint" {
  pool_id     = lattice_public_ip_pool.kube.id
  description = "k8s control-plane endpoint"
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

# Managed Kubernetes cluster
resource "lattice_kube_cluster" "prod" {
  name          = "prod"
  talos_image   = "/var/lib/lattice/images/talos-v1.9.0-metal-amd64.raw"
  talos_version = local.talos_version
  k8s_version   = local.k8s_version

  pool_id = lattice_public_ip_pool.kube.id

  cni     = "cilium"
  lb_mode = "cilium"

  cp_count  = 3
  cp_vcpus  = 4
  cp_memory_mb = 8192
  cp_disk_gb   = 50

  worker_count     = 3
  worker_vcpus     = 8
  worker_memory_mb = 16384
  worker_disk_gb   = 100
}

# LINSTOR-backed storage volume mounted into a VM
resource "lattice_storage_volume" "data" {
  name       = "prod-data-0"
  size_gb    = 200
  backend_id = lattice_storage_backend.linstor.id
}

output "kube_endpoint" {
  description = "Kubernetes API server URL"
  value       = lattice_kube_cluster.prod.endpoint
}

output "kubeconfig" {
  description = "Kubeconfig for kubectl access"
  value       = lattice_kube_cluster.prod.kubeconfig
  sensitive   = true
}

output "talosconfig" {
  description = "Talosconfig for talosctl access"
  value       = lattice_kube_cluster.prod.talosconfig
  sensitive   = true
}

output "public_ip" {
  description = "Allocated control-plane public IP"
  value       = lattice_public_ip.cp_endpoint.ip
}
