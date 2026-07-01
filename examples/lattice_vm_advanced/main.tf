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
  type    = string
  default = "https://lattice.local:8006"
}

variable "lattice_api_key" {
  type      = string
  sensitive = true
  default   = ""
}

data "lattice_image" "debian" {
  distro  = "debian"
  version = "12"
}

# VPC with firewall and port forwarding
resource "lattice_vpc" "app" {
  name           = "app-vpc"
  cidr           = "10.10.0.0/24"
  default_action = "drop"

  firewall_rules = [
    {
      direction = "ingress"
      proto     = "tcp"
      port      = "22"
      cidr      = "10.0.0.0/8"
      action    = "accept"
      desc      = "SSH from corp"
    },
    {
      direction = "ingress"
      proto     = "tcp"
      port      = "443"
      cidr      = "0.0.0.0/0"
      action    = "accept"
      desc      = "HTTPS public"
    },
  ]

  port_forwards = [
    {
      proto     = "tcp"
      ext_port  = 8443
      dest_ip   = "10.10.0.10"
      dest_port = 443
      desc      = "app HTTPS"
    },
  ]
}

# IPAM pool for automatic VM IP assignment
resource "lattice_ipam_pool" "app" {
  name        = "app-pool"
  bridge      = lattice_vpc.app.bridge
  subnet      = "10.10.0.0/24"
  gateway     = "10.10.0.1"
  range_start = "10.10.0.10"
  range_end   = "10.10.0.200"
  dns         = ["1.1.1.1", "8.8.8.8"]
}

# Public IP pool; reserve this CIDR inside br0's connected subnet.
resource "lattice_public_ip_pool" "app" {
  name      = "app-public"
  interface = "br0"
  cidr      = "192.168.200.64/26"
}

# Allocate and NAT a public IP to the app VM
resource "lattice_public_ip" "app" {
  pool_id     = lattice_public_ip_pool.app.id
  description = "app-01 public IP"
  private_ip  = "10.10.0.10"
}

# Security group for the app tier
resource "lattice_security_group" "app" {
  name        = "app-sg"
  description = "App-tier security group"

  rules = [
    {
      direction = "ingress"
      protocol  = "tcp"
      port_from = 443
      port_to   = 443
      cidr      = "0.0.0.0/0"
      action    = "accept"
      priority  = 100
    },
    {
      direction = "ingress"
      protocol  = "tcp"
      port_from = 22
      port_to   = 22
      cidr      = "10.0.0.0/8"
      action    = "accept"
      priority  = 90
    },
    {
      direction = "egress"
      protocol  = "all"
      port_from = 0
      port_to   = 0
      cidr      = "0.0.0.0/0"
      action    = "accept"
      priority  = 1
    },
  ]
}

# Storage backend (LVM for local performance)
resource "lattice_storage_backend" "lvm" {
  name = "lvm-local"
  type = "lvm"
  config = {
    vg_name = "data"
  }
}

# Data volume for the app
resource "lattice_storage_volume" "app_data" {
  name       = "app-data"
  size_gb    = 100
  backend_id = lattice_storage_backend.lvm.id
}

# The VM itself
resource "lattice_vm" "app" {
  name         = "app-01"
  image_id     = data.lattice_image.debian.id
  cpus         = 4
  memory_mb    = 8192
  boot_disk_gb = 40
}

resource "lattice_vm_security_group" "app" {
  vm_id             = lattice_vm.app.id
  security_group_id = lattice_security_group.app.id
}

output "public_ip" {
  value = lattice_public_ip.app.ip
}

output "vm_id" {
  value = lattice_vm.app.id
}
