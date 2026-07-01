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

# Discover the latest amd64 Firecracker kernel in the Kernel Catalog and
# import it into the kernels table (id is reused, so the import resource's
# id is directly usable as kernel_id below).
data "lattice_kernel_catalog" "fc" {
  distro = "firecracker"
  arch   = "amd64"
}

resource "lattice_kernel_catalog_import" "fc" {
  entry_id = data.lattice_kernel_catalog.fc.id
}

# VPC — Firecracker VMs use the same bridge model as QEMU
resource "lattice_vpc" "fc" {
  name           = "fc-vpc"
  cidr           = "10.30.0.0/24"
  default_action = "drop"

  firewall_rules = [
    {
      direction = "ingress"
      proto     = "tcp"
      port      = "22"
      cidr      = "10.0.0.0/8"
      action    = "accept"
      desc      = "SSH from management"
    },
    {
      direction = "egress"
      proto     = "all"
      port      = ""
      cidr      = "0.0.0.0/0"
      action    = "accept"
      desc      = "Allow all outbound"
    },
  ]
}

# DHCP pool on the VPC bridge
resource "lattice_ipam_pool" "fc" {
  name        = "fc-pool"
  bridge      = lattice_vpc.fc.bridge
  subnet      = "10.30.0.0/24"
  gateway     = "10.30.0.1"
  range_start = "10.30.0.10"
  range_end   = "10.30.0.200"
  dns         = ["1.1.1.1", "8.8.8.8"]
}

# Public IP pool for NAT. The CIDR must be reserved inside br0's connected
# subnet and excluded from upstream DHCP.
resource "lattice_public_ip_pool" "fc" {
  name      = "fc-public"
  interface = "br0"
  cidr      = "192.168.50.64/26"
}

# Allocate a public IP and NAT it to the Firecracker VM
resource "lattice_public_ip" "fc" {
  pool_id     = lattice_public_ip_pool.fc.id
  description = "fc-01 public IP"
  private_ip  = "10.30.0.10"
}

# Firecracker microVM
#
# vm_type = "firecracker" selects the Firecracker backend.
# kernel_id references an already-imported kernel (GET /kernels). New kernels
# reach that table by uploading directly (POST /kernels) or importing a
# Kernel Catalog entry (POST /kernel-catalog/{id}/import, as above).
# Firecracker boots directly — no BIOS, no UEFI, no GRUB.
#
# The NIC bridges into the same VPC as a QEMU VM would. Firecracker uses
# a TAP device instead of a virtio-net-pci bus, but from the network
# perspective the VM is just another host on the bridge.
resource "lattice_vm" "fc" {
  name      = "fc-01"
  vm_type   = "firecracker"
  kernel_id = lattice_kernel_catalog_import.fc.id
  cpus      = 2
  memory_mb = 1024

  # Firecracker boots a raw disk image (not qcow2).
  # boot_disk_gb sets the size of the sparse raw file LatticeVE creates.
  boot_disk_gb = 10

  # Extra kernel args appended after the LatticeVE defaults.
  # LatticeVE already sets console=ttyS0, reboot=k, panic=1.
  kernel_cmdline = "ip=10.30.0.10::10.30.0.1:255.255.255.0::eth0:off"

  nics = [
    {
      bridge = lattice_vpc.fc.bridge
      model  = "virtio-net-pci"
    },
  ]
}

output "kernel_version" {
  value       = lattice_kernel_catalog_import.fc.version
  description = "Version of the imported Firecracker kernel"
}

output "fc_vm_id" {
  value = lattice_vm.fc.id
}

output "fc_public_ip" {
  value       = lattice_public_ip.fc.ip
  description = "Public IP NAT'd to the Firecracker VM"
}

output "fc_bridge" {
  value       = lattice_vpc.fc.bridge
  description = "Host bridge — Firecracker TAP device is plugged in here"
}
