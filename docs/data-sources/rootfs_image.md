# lattice_rootfs_image (Data Source)

Looks up a Firecracker rootfs image from LatticeVE's general-purpose rootfs registry (`GET /rootfs-images`) — covers both manual uploads and images imported via the k3s auto-fetch flow (`source = "latticeve-k3s-images"`). At least one filter must be set. If multiple images match, the most recently created one is returned.

## Example Usage

```hcl
# Look up the latest manually-uploaded image by name
data "lattice_rootfs_image" "custom" {
  name = "my-custom-rootfs"
}

# Look up the latest k3s rootfs already imported for amd64
data "lattice_rootfs_image" "k3s" {
  source = "latticeve-k3s-images"
  arch   = "amd64"
}

resource "lattice_kube_cluster" "prod" {
  name      = "prod"
  rootfs_id = data.lattice_rootfs_image.k3s.id
  # ...
}
```

## Argument Reference

At least one of `id`, `name`, `arch`, `source`, or `version` must be set. `id` short-circuits all other filters.

- `id` (Optional) — Filter by exact image UUID.
- `name` (Optional) — Filter by exact image name.
- `arch` (Optional) — Filter by architecture: `amd64` or `arm64`.
- `source` (Optional) — Filter by import source, e.g. `latticeve-k3s-images`. Empty for manual uploads.
- `version` (Optional) — Filter by exact version (only set for images imported via the k3s auto-fetch flow).

## Attribute Reference

- `description` — Image description.
- `rootfs_path` — Host path to the rootfs image.
- `size_bytes` — Image size in bytes.
- `sha256` — SHA-256 checksum of the rootfs image.
- `created_at` — ISO 8601 timestamp when the image was uploaded or imported.
