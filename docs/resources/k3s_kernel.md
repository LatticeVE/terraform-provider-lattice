# lattice_k3s_kernel

Imports a Kubernetes-compatible Firecracker kernel published by [latticeve-k3s-images](https://github.com/LatticeVE/latticeve-k3s-images). These kernels include the networking features required by k3s.

```hcl
resource "lattice_k3s_kernel" "kube" {
  arch    = "amd64"
  version = "6.1.175" # optional; omit for newest discovered build
}
```

Use `id` as `kernel_id` on `lattice_kube_cluster`. Both `arch` and `version` require replacement. Existing clusters retain their copied boot kernel, but normal Terraform references still ensure cluster changes and destruction occur in dependency order.

## Arguments

- `arch` (Required) — `amd64` or `arm64`.
- `version` (Optional, Computed) — Exact Linux kernel version. Omit to select the newest discovered version.

## Attributes

- `id` — Imported kernel UUID.
- `name` — Imported kernel name.
- `download_url` — Verified release asset URL.
- `size_bytes` — Kernel size.
