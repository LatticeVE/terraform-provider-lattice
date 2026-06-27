# lattice_storage_volume

Provisions a storage volume on a LatticeVE storage backend. Volumes can be grown after creation (shrinking is not supported).

## Example Usage

```hcl
resource "lattice_storage_volume" "data" {
  name       = "vm-data-0"
  size_gb    = 100
  backend_id = lattice_storage_backend.linstor.id
}
```

## Argument Reference

- `name` (Required, Forces new resource) — Volume name.
- `size_gb` (Required) — Volume size in GiB. Can be increased to grow the volume; decreases are rejected.
- `backend_id` (Required, Forces new resource) — ID of the `lattice_storage_backend` to provision on.

## Attribute Reference

- `id` — Volume UUID.
- `size_bytes` — Actual provisioned size in bytes.
- `diskful_nodes` — List of storage nodes that hold a full copy of the volume (LINSTOR backends).
- `created_at` — ISO 8601 creation timestamp.
