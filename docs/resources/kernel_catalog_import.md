# lattice_kernel_catalog_import

Imports a `lattice_kernel_catalog` entry into LatticeVE's kernels table. Imports are asynchronous on the controller; this resource polls until the import finishes (10-minute timeout). The resulting `id` matches the catalog entry's `id` and can be used directly as `kernel_id`.

## Example Usage

```hcl
data "lattice_kernel_catalog" "fc_amd64" {
  distro = "firecracker"
  arch   = "amd64"
}

resource "lattice_kernel_catalog_import" "fc_kernel" {
  entry_id = data.lattice_kernel_catalog.fc_amd64.id
}

resource "lattice_kube_cluster" "prod" {
  name      = "prod"
  kernel_id = lattice_kernel_catalog_import.fc_kernel.id
  # ...
}
```

## Argument Reference

- `entry_id` (Required, Forces new resource) — ID of the catalog entry to import (from `lattice_kernel_catalog.id`).

## Attribute Reference

- `id` — Resulting kernel ID, equal to `entry_id`. Use this as `kernel_id`.
- `name`, `distro`, `version`, `arch` — Kernel metadata copied from the kernels table after import.
- `vmlinuz_path` — Host path to the imported kernel image.
- `size_bytes` — Size of the imported kernel image in bytes.

## Notes

Deleting this resource calls `DELETE /kernels/{id}`. If the kernel is still referenced by a VM or cluster, the underlying API call may fail — remove those references first.
