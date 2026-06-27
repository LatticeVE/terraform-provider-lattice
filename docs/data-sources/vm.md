# lattice_vm (Data Source)

Look up a LatticeVE VM by ID or name.

## Example Usage

```hcl
data "lattice_vm" "web" {
  name = "web-01"
}

output "web_status" {
  value = data.lattice_vm.web.status
}
```

## Argument Reference

One of `id` or `name` must be provided.

- `id` (Optional) — VM UUID.
- `name` (Optional) — VM name.

## Attribute Reference

- `id` — VM UUID.
- `name` — VM name.
- `cpus` — Number of vCPUs.
- `memory_mb` — Memory in MiB.
- `status` — Current VM status.
- `disk_path` — Backing disk image path.
- `boot_disk_gb` — Boot disk size in GiB.
