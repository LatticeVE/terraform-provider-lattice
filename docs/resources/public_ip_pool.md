# lattice_public_ip_pool

Manages a LatticeVE public IP pool. A pool defines a routable CIDR block tied to a host NIC from which individual IPs can be allocated.

## Example Usage

```hcl
resource "lattice_public_ip_pool" "main" {
  name      = "main-pool"
  interface = "eth0"
  cidr      = "192.168.100.128/27"
}
```

## Argument Reference

- `name` (Required) — Pool name.
- `interface` (Required, Forces new resource) — Host network interface, e.g. `eth0`.
- `cidr` (Required, Forces new resource) — Routable CIDR block to assign from, e.g. `192.168.100.128/27`.

## Attribute Reference

- `id` — Pool UUID.
- `created_at` — ISO 8601 creation timestamp.
