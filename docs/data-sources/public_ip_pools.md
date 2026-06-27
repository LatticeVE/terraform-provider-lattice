# lattice_public_ip_pools (Data Source)

Lists all public IP pools registered in LatticeVE.

## Example Usage

```hcl
data "lattice_public_ip_pools" "all" {}

output "pool_ids" {
  value = [for p in data.lattice_public_ip_pools.all.pools : p.id]
}
```

## Attribute Reference

- `pools` — List of pools. Each entry:
  - `id` — Pool UUID.
  - `name` — Pool name.
  - `interface` — Host NIC.
  - `cidr` — CIDR block.
