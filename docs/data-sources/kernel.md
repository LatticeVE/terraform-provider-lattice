# lattice_kernel (Data Source)

Look up an already-imported kernel from LatticeVE's `GET /kernels`. At least one filter must be provided. If multiple kernels match, the most recently imported one is returned.

To browse kernels available to import but not yet imported, use `lattice_kernel_catalog` instead, then `lattice_kernel_catalog_import` to import one.

Use the result's `id` as the `kernel_id` argument on `lattice_vm` (when `vm_type = "firecracker"`) or `lattice_kube_cluster`.

## Example Usage

```hcl
# Pin to an exact kernel version for a given arch
data "lattice_kernel" "pinned" {
  version = "6.1.141"
  arch    = "amd64"
}

# Float within a minor via glob
data "lattice_kernel" "glob" {
  distro       = "firecracker"
  version_glob = "6.1.*"
}

# Exact name match
data "lattice_kernel" "named" {
  name = "Firecracker Kernel 6.1.141 (amd64)"
}

resource "lattice_vm" "fc" {
  name      = "fc-01"
  vm_type   = "firecracker"
  kernel_id = data.lattice_kernel.pinned.id
  cpus      = 2
  memory_mb = 1024
  # ...
}
```

## Argument Reference

At least one of `distro`, `name`, `version`, `version_glob`, or `arch` must be set. All provided filters are ANDed together.

- `distro` (Optional) — Distro name, e.g. `firecracker`, `alpine`.
- `name` (Optional) — Exact kernel name as registered in the kernels table.
- `version` (Optional) — Exact kernel version. Mutually exclusive with `version_glob`.
- `version_glob` (Optional) — Glob pattern matched against the kernel version. Supports `*` and `?`, e.g. `6.1.*`. Mutually exclusive with `version`.
- `arch` (Optional) — Architecture: `amd64` or `arm64`.

## Attribute Reference

- `id` — Kernel UUID. Pass this to `kernel_id`.
- `created_at` — ISO 8601 timestamp of when the kernel was imported.
- `size_bytes` — Size of the kernel image in bytes.
- `vmlinuz_path` — Host path to the kernel image file.

## Notes

Imported kernels are managed via `GET/POST/DELETE /kernels` on the LatticeVE API. New kernels reach that table either by direct upload (`POST /kernels`) or by importing a Kernel Catalog entry (`POST /kernel-catalog/{id}/import`, modeled here as `lattice_kernel_catalog_import`).
