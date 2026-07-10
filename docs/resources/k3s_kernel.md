# lattice_k3s_kernel

Imports or reuses a Kubernetes-compatible Firecracker kernel published by [latticeve-k3s-images](https://github.com/LatticeVE/latticeve-k3s-images). These kernels include the networking features required by k3s.

```hcl
resource "lattice_k3s_kernel" "kube" {
  arch    = "amd64"
  version = "6.1.175" # optional; omit for newest discovered build
}
```

Use `id` as `kernel_id` on `lattice_kube_cluster`. Both `arch` and `version` require replacement. The resource is idempotent: it reuses an already-imported `latticeve-k3s` kernel with the same architecture and discovered name/version, and downloads only when no match exists.

On destroy, Terraform deletes kernels downloaded by this resource. Kernels that were already present and reused by this resource are left in LatticeVE. Existing clusters retain their copied boot kernel, but normal Terraform references still ensure cluster changes and destruction occur in dependency order. The LatticeVE API also blocks deletion while a non-deleting Kubernetes cluster still references the kernel.

## Arguments

- `arch` (Required) — `amd64` or `arm64`.
- `version` (Optional, Computed) — Exact Linux kernel version. Omit to select the newest discovered version.

## Attributes

- `id` — Imported kernel UUID.
- `name` — Imported kernel name.
- `download_url` — Verified release asset URL.
- `size_bytes` — Kernel size.
- `managed` — `true` when Terraform downloaded the kernel; `false` when Terraform reused an existing imported kernel.

## Notes

Use `data "lattice_kernel"` when you want an explicitly read-only reference to an already-imported kernel. Use `resource "lattice_k3s_kernel"` when Terraform should ensure the requested k3s kernel exists, downloading it only if needed.
