# lattice_kernel_catalog (Data Source)

Looks up a kernel from LatticeVE's Kernel Catalog (`GET /kernel-catalog`) — built-in entries plus anything discovered from Firecracker's CI bucket. Catalog entries are not yet usable as `kernel_id`; import one with `lattice_kernel_catalog_import` first. At least one filter must be provided.

## Example Usage

```hcl
data "lattice_kernel_catalog" "latest_fc_amd64" {
  distro = "firecracker"
  arch   = "amd64"
}

resource "lattice_kernel_catalog_import" "fc_kernel" {
  entry_id = data.lattice_kernel_catalog.latest_fc_amd64.id
}

resource "lattice_vm" "fc" {
  name      = "fc-01"
  vm_type   = "firecracker"
  kernel_id = lattice_kernel_catalog_import.fc_kernel.id
  cpus      = 2
  memory_mb = 1024
}
```

## Argument Reference

At least one of `distro`, `name`, `version`, `version_glob`, or `arch` must be set.

- `distro` (Optional) — Distro name, e.g. `firecracker`.
- `name` (Optional) — Exact catalog entry name.
- `version` (Optional) — Exact kernel version. Mutually exclusive with `version_glob`.
- `version_glob` (Optional) — Glob pattern matched against the kernel version, e.g. `6.1.*`. Mutually exclusive with `version`.
- `arch` (Optional) — Architecture: `amd64` or `arm64`.

## Attribute Reference

- `id` — Catalog entry ID — pass to `lattice_kernel_catalog_import.entry_id`.
- `vmlinuz_url` — Source URL the kernel is downloaded from on import.
- `vmlinuz_size_mb` — Approximate kernel image size in MB.
- `description` — Catalog entry description.
- `imported` — Whether this entry has already been imported on this controller.
