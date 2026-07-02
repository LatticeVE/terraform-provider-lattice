# lattice_storage_backends (Data Source)

Lists all storage backends registered in LatticeVE.

## Example Usage

```hcl
data "lattice_storage_backends" "all" {}

# Find the default backend
locals {
  default_backend = one([for b in data.lattice_storage_backends.all.backends : b if b.is_default])
}

resource "lattice_storage_volume" "data" {
  name       = "myvolume"
  size_gb    = 50
  backend_id = local.default_backend.id
}
```

## Attribute Reference

- `backends` — List of backends. Each entry:
  - `id` — Backend UUID.
  - `name` — Backend name.
  - `type` — Backend type (`lvm`, `linstor`, etc.).
  - `is_default` — Whether this is the default backend.
  - `allocation_policy` — `thin` or `preallocated`.
  - `disk_overcommit_ratio` — Configured logical-to-physical capacity ratio.
