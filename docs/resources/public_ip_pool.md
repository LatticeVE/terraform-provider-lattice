# lattice_public_ip_pool

Manages a LatticeVE public IP pool. A pool defines a reserved CIDR inside an eligible external Linux bridge's connected IPv4 subnet.

## Example Usage

```hcl
resource "lattice_public_ip_pool" "main" {
  name      = "main-pool"
  interface = "br0"
  cidr      = "10.0.7.128/28"
}
```

## Argument Reference

- `name` (Required) — Pool name.
- `interface` (Required, Forces new resource) — Eligible external bridge, e.g. `br0` or `vmbr0`.
- `cidr` (Required, Forces new resource) — Canonical IPv4 CIDR inside the bridge subnet. Reserve it from upstream DHCP; it cannot contain the bridge IP or overlap another managed network.

## Attribute Reference

- `id` — Pool UUID.
- `created_at` — ISO 8601 creation timestamp.
