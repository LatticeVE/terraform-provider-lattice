# lattice_k3s_rootfs_image

Imports a pinned or latest k3s rootfs build for a given architecture from [`latticeve-k3s-images`](https://github.com/LatticeVE/latticeve-k3s-images)' GitHub releases. The resulting `id` can be used as `rootfs_id` in `lattice_kube_cluster`.

Import happens once at creation time. Set `version` for reproducible clusters and change it to replace the image during an upgrade. Keep the old image until the cluster update completes:

```hcl
lifecycle {
  create_before_destroy = true
}
```

## Example Usage

```hcl
resource "lattice_k3s_rootfs_image" "release" {
  arch    = "amd64"
  version = "v1.36.2+k3s1-r23"
  lifecycle { create_before_destroy = true }
}

resource "lattice_kube_cluster" "prod" {
  name      = "prod"
  rootfs_id = lattice_k3s_rootfs_image.release.id
  # ...
}
```

## Argument Reference

- `arch` (Required, Forces new resource) — Architecture to import: `amd64` or `arm64`.
- `version` (Optional, Computed, Forces new resource) — Exact published image version. Omit to import the newest discovered build.

## Attribute Reference

- `id` — UUID of the imported rootfs image. Use this as `rootfs_id`.
- `name` — Name of the imported rootfs image.
- `download_url` — GitHub release asset URL the image was downloaded from.
- `size_bytes` — Size of the imported rootfs image in bytes.
