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
  insecure = true
}

# Look up Debian 12 from the image catalog (like an AMI lookup in AWS)
data "lattice_image" "debian" {
  distro  = "debian"
  version = "12"
}

# Isolated private network for the VM
resource "lattice_vpc" "web" {
  name           = "web-vpc"
  cidr           = "10.20.0.0/24"
  default_action = "accept"
}

# DHCP pool so the VM gets an IP automatically
resource "lattice_ipam_pool" "web" {
  name        = "web-pool"
  bridge      = lattice_vpc.web.bridge
  subnet      = "10.20.0.0/24"
  gateway     = "10.20.0.1"
  range_start = "10.20.0.10"
  range_end   = "10.20.0.200"
  dns         = ["1.1.1.1", "8.8.8.8"]
}

# Basic QEMU VM — boots from the Debian 12 image (cloned, like launching from an AMI)
resource "lattice_vm" "web" {
  name         = "web-01"
  image_id     = data.lattice_image.debian.id
  cpus         = 2
  memory_mb    = 2048
  boot_disk_gb = 20

  nics = [
    {
      bridge = lattice_vpc.web.bridge
    },
  ]

  cloud_init = {
    user_data = <<-EOF
      #cloud-config
      hostname: web-01
      users:
        - name: admin
          sudo: ALL=(ALL) NOPASSWD:ALL
          ssh_authorized_keys:
            - ssh-ed25519 AAAA... your-key-here
    EOF
    meta_data = "instance-id: web-01\nlocal-hostname: web-01\n"
  }
}

output "vm_id" {
  value = lattice_vm.web.id
}

output "vpc_bridge" {
  value = lattice_vpc.web.bridge
}
