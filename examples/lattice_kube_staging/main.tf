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
  insecure = var.lattice_insecure
}

locals {
  cluster_name = "${var.resource_prefix}-cluster"
}

resource "lattice_k3s_kernel" "kube" {
  arch    = var.arch
  version = var.kernel_version
}

resource "lattice_k3s_rootfs_image" "kube" {
  arch    = var.arch
  version = var.rootfs_version

  lifecycle {
    create_before_destroy = true
  }
}

resource "lattice_public_ip_pool" "kube" {
  count = var.public_pool_cidr == "" ? 0 : 1

  name      = "${var.resource_prefix}-public"
  interface = var.public_bridge
  cidr      = var.public_pool_cidr
}

resource "lattice_kube_cluster" "kube" {
  name      = local.cluster_name
  runtime   = "firecracker"
  kernel_id = lattice_k3s_kernel.kube.id
  rootfs_id = lattice_k3s_rootfs_image.kube.id

  # Omitted when public_pool_cidr is empty. This validates the VPC-only cluster
  # path by default; set public_pool_cidr to also test external endpoint/CCM IPs.
  pool_id = var.public_pool_cidr == "" ? null : lattice_public_ip_pool.kube[0].id

  cni            = "flannel"
  lb_mode        = "ccm"
  metrics_server = true

  cp_count     = var.cp_count
  cp_vcpus     = var.cp_vcpus
  cp_memory_mb = var.cp_memory_mb
  cp_disk_gb   = var.cp_disk_gb

  worker_count     = var.worker_count
  worker_vcpus     = var.worker_vcpus
  worker_memory_mb = var.worker_memory_mb
  worker_disk_gb   = var.worker_disk_gb
}

output "cluster_id" {
  value = lattice_kube_cluster.kube.id
}

output "cluster_name" {
  value = lattice_kube_cluster.kube.name
}

output "cluster_status" {
  value = lattice_kube_cluster.kube.status
}

output "cluster_endpoint" {
  value = lattice_kube_cluster.kube.endpoint
}

output "cluster_vpc" {
  value = {
    id      = lattice_kube_cluster.kube.vpc_id
    cidr    = lattice_kube_cluster.kube.vpc_cidr
    managed = lattice_kube_cluster.kube.vpc_managed
  }
}

output "cluster_public_ip" {
  value = lattice_kube_cluster.kube.public_ip
}

output "kernel" {
  value = {
    id      = lattice_k3s_kernel.kube.id
    version = lattice_k3s_kernel.kube.version
    managed = lattice_k3s_kernel.kube.managed
  }
}

output "rootfs" {
  value = {
    id      = lattice_k3s_rootfs_image.kube.id
    version = lattice_k3s_rootfs_image.kube.version
  }
}
