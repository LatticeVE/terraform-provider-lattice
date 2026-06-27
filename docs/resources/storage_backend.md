# lattice_storage_backend

Registers a storage backend in LatticeVE. Backends define where and how volumes are provisioned (LVM, LINSTOR, NFS, Ceph, local).

## Example Usage

```hcl
resource "lattice_storage_backend" "linstor" {
  name = "linstor-default"
  type = "linstor"
  config = {
    controller = "linstor://10.0.0.1:3370"
    pool_name  = "thinpool"
  }
}

resource "lattice_storage_backend" "lvm" {
  name = "lvm-local"
  type = "lvm"
  config = {
    vg_name = "data"
  }
}
```

## Argument Reference

- `name` (Required) — Backend name.
- `type` (Required, Forces new resource) — Backend driver: `lvm`, `linstor`, `nfs`, `ceph`, or `local`.
- `config` (Optional) — Map of string key/value configuration for the backend driver. Keys depend on the backend type.

## Attribute Reference

- `id` — Backend UUID.
- `is_default` — Whether this is the default storage backend.
- `created_at` — ISO 8601 creation timestamp.
