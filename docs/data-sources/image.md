# lattice_image (Data Source)

Look up a LatticeVE image to use as the boot disk for a `lattice_vm`. Works like an AWS AMI lookup — the image is cloned for the VM at creation time. At least one filter must be set. If multiple images match, the most recently imported one is returned.

## Example Usage

```hcl
# Debian 12 (amd64 is the default when arch is omitted)
data "lattice_image" "debian" {
  distro  = "debian"
  version = "12"
}

# Ubuntu 26.04 LTS, arm64
data "lattice_image" "ubuntu_arm" {
  distro  = "ubuntu"
  version = "26.04"
  arch    = "arm64"
}

# Alpine 3.24
data "lattice_image" "alpine" {
  distro  = "alpine"
  version = "3.24"
}

# Look up by exact UUID (skips all other filters)
data "lattice_image" "pinned" {
  id = "a1b2c3d4-..."
}

resource "lattice_vm" "web" {
  name         = "web-01"
  image_id     = data.lattice_image.debian.id
  cpus         = 2
  memory_mb    = 2048
  boot_disk_gb = 20
  nics         = [{ bridge = lattice_vpc.web.bridge }]
}
```

## Argument Reference

At least one filter must be set. When `id` is set it short-circuits all other filters.

- `id` (Optional) — Exact image UUID.
- `name` (Optional) — Exact image name.
- `distro` (Optional) — Distro name: `debian`, `ubuntu`, `alpine`, `fedora`, `rocky`, etc.
- `version` (Optional) — Distro version string, e.g. `12`, `26.04`, `3.24`.
- `arch` (Optional) — Architecture: `amd64` (default when omitted) or `arm64`.

## Attribute Reference

- `id` — Image UUID. Pass this to `lattice_vm.image_id`.
- `name` — Image name as registered in the catalog.
- `distro` — Distro of the selected image.
- `version` — Version of the selected image.
- `arch` — Architecture of the selected image.
- `format` — Disk format: `qcow2` or `raw`.
- `size_bytes` — Image size in bytes.
- `description` — Image description.
- `created_at` — ISO 8601 import timestamp.

## Notes

Images are managed via `GET /images`, `POST /images/import` (URL fetch + convert), and `DELETE /images/{id}` on the LatticeVE API. LatticeVE ships a curated catalog of cloud images for Debian, Ubuntu, Alpine, Fedora, and Rocky Linux. Custom images can be imported from any URL that serves a raw or qcow2 disk image.
