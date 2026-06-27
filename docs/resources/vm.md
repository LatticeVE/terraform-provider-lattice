# lattice_vm

Manages a LatticeVE virtual machine (QEMU/KVM or Firecracker).

## Example Usage

### QEMU VM launched from an image (AMI-style)

```hcl
data "lattice_image" "debian" {
  name = "debian-12-generic-amd64"
}

resource "lattice_vm" "web" {
  name         = "web-01"
  cpus         = 4
  memory_mb    = 8192
  image_id     = data.lattice_image.debian.id
  boot_disk_gb = 40
  nics         = [{ bridge = lattice_vpc.main.bridge }]
}
```

### Firecracker microVM

```hcl
data "lattice_kernel" "alpine" {
  distro         = "alpine"
  distro_version = "3.24.1"
}

resource "lattice_vm" "fc" {
  name           = "fc-01"
  vm_type        = "firecracker"
  kernel_id      = data.lattice_kernel.alpine.id
  kernel_cmdline = "console=ttyS0 reboot=k panic=1 pci=off"
  cpus           = 2
  memory_mb      = 512
  nics           = [{ bridge = lattice_vpc.main.bridge }]
}
```

### ARM64 VM with arch-based placement

```hcl
# Discover arm64 nodes
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
  arch         = "arm64"
  nics         = [{ bridge = lattice_vpc.main.bridge }]
}
```

## Argument Reference

- `name` (Required) — VM name. Must be unique within the controller.
- `cpus` (Required) — Number of virtual CPUs.
- `memory_mb` (Required) — Memory in MiB.
- `boot_disk_gb` (Optional) — Boot disk size in GiB. The backing image is cloned and resized to this size. Defaults to the image size if omitted.
- `image_id` (Optional, Forces replacement) — UUID of a `lattice_image` data source. The image is cloned as the VM's boot disk, equivalent to launching from an AMI. Mutually exclusive with `disk_path`.
- `iso_path` (Optional) — Path to an ISO file on the host to attach as a CD-ROM drive. Mutually exclusive with `image_id`.
- `vm_type` (Optional) — Hypervisor type. One of `qemu` (default) or `firecracker`.
- `kernel_id` (Optional) — UUID of a Firecracker kernel from the `lattice_kernel` catalog. Required when `vm_type = "firecracker"`.
- `kernel_cmdline` (Optional) — Kernel command-line arguments passed to the Firecracker guest kernel. Only valid when `vm_type = "firecracker"`.
- `arch` (Optional, Computed, Forces replacement) — CPU architecture required for VM placement: `amd64` or `arm64`. The scheduler selects a node whose CPU matches. Computed from the assigned node when not set. Use the `lattice_nodes` data source to discover available architectures.
- `node` (Optional, Computed, Forces replacement) — Name of the specific host node to run this VM on. Pins placement to a node, bypassing the arch-based scheduler. Computed to reflect actual placement after creation.
- `force_destroy` (Optional, Forces replacement) — If true, Terraform destroy requests a force-delete. Defaults to false; normal destroy requests LatticeVE to stop the VM before deleting it.
- `nics` (Optional) — List of network interface objects. Each object accepts:
  - `bridge` (Required) — Bridge device to attach the NIC to (e.g. a VPC bridge).

## Attribute Reference

- `id` — VM UUID assigned by the controller.
- `status` — VM lifecycle status (`stopped`, `running`, `paused`, etc.).
- `disk_path` — Absolute host path to the managed boot disk image.
- `arch` — Resolved CPU architecture of the node where the VM was placed.
- `node` — Name of the host node where the VM is running.
