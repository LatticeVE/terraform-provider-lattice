# lattice_ipam_pool

Manages a LatticeVE IPAM pool for automatic IP assignment to VMs inside a VPC bridge.

## Example Usage

```hcl
resource "lattice_ipam_pool" "app" {
  name        = "app-ipam"
  bridge      = lattice_vpc.app.bridge
  subnet      = "10.10.0.0/24"
  gateway     = "10.10.0.1"
  range_start = "10.10.0.10"
  range_end   = "10.10.0.200"
  dns         = ["1.1.1.1", "8.8.8.8"]
}
```

## Argument Reference

Pool arguments are updated in place. LatticeVE revalidates the subnet, gateway, range, DNS values, and managed-network overlap before applying a change.

- `name` (Required) — Pool name.
- `bridge` (Required) — Host bridge interface (typically from `lattice_vpc.X.bridge`).
- `subnet` (Required) — Subnet CIDR, e.g. `10.10.0.0/24`.
- `gateway` (Required) — Default gateway IP.
- `range_start` (Required) — First allocatable IP.
- `range_end` (Required) — Last allocatable IP.
- `dns` (Optional) — List of DNS server IPs.

## Attribute Reference

- `id` — Pool UUID.
- `created_at` — ISO 8601 creation timestamp.
