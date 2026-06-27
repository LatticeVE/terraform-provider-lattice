# lattice_kernel (Data Source)

Look up a kernel from the LatticeVE kernel catalog. At least one filter must be provided. If multiple kernels match, the most recently built one is returned.

Use the result's `id` as the `kernel_id` argument on `lattice_vm` when `vm_type = "firecracker"`.

## Example Usage

```hcl
# Alpine 3.24.1 — use the distro's own release version
data "lattice_kernel" "alpine" {
  distro         = "alpine"
  distro_version = "3.24.1"
}

# Ubuntu 26.04 LTS
data "lattice_kernel" "ubuntu" {
  distro         = "ubuntu"
  distro_version = "26.04"
}

# Pin to an exact upstream kernel version (distro-agnostic)
data "lattice_kernel" "pinned" {
  distro  = "alpine"
  version = "6.12.9"
}

# Float within an upstream minor via glob (useful for custom/uploaded kernels
# that may not have distro_version set)
data "lattice_kernel" "glob" {
  distro       = "alpine"
  version_glob = "6.12.*"
}

# Exact name match (useful for custom-named catalog entries)
data "lattice_kernel" "talos" {
  name = "talos-v1.9.0"
}

resource "lattice_vm" "fc" {
  name      = "fc-01"
  vm_type   = "firecracker"
  kernel_id = data.lattice_kernel.alpine.id
  cpus      = 2
  memory_mb = 1024
  # ...
}
```

## Argument Reference

At least one of `distro`, `name`, or `version` must be set. All provided filters are ANDed together.

- `distro` (Optional) — Distro name: `alpine`, `ubuntu`, `debian`, `fedora-coreos`, `talos`.
- `distro_version` (Optional) — Distro release version in the distro's own scheme: `3.24.1` for Alpine, `26.04` for Ubuntu. Preferred over `version` when you want "whatever kernel ships with this release".
- `name` (Optional) — Exact kernel name as registered in the catalog.
- `version` (Optional) — Exact upstream Linux kernel version, e.g. `6.12.9`. Mutually exclusive with `version_glob`.
- `version_glob` (Optional) — Glob pattern matched against the upstream kernel version. Supports `*` and `?`. E.g. `6.12.*` floats across patches, `6.*` matches any 6.x kernel. Useful for uploaded/custom kernels without a `distro_version`. Mutually exclusive with `version`.

## Attribute Reference

- `id` — Kernel UUID. Pass this to `lattice_vm.kernel_id`.
- `resolved_distro_version` — Distro release version of the selected kernel (e.g. `3.24.1`, `26.04`). Useful when filtering by `distro` only and you want to confirm which release was picked.
- `built_at` — ISO 8601 timestamp of when the kernel was built or imported.
- `size_bytes` — Combined size of vmlinuz + initramfs in bytes.
- `vmlinuz_path` — Host path to the kernel image file.
- `initramfs_path` — Host path to the initramfs image file.

## Notes

Kernels are managed via `GET/POST/DELETE /kernels` on the LatticeVE API. New kernels can be imported from a URL (`POST /kernels/import`) or uploaded directly (`POST /kernels/upload`). LatticeVE ships curated kernels for Alpine (~6 MB), Ubuntu, Debian, and Fedora CoreOS; Talos kernels are pulled from the Talos image factory.
