# lattice_vpc (Data Source)

Look up a LatticeVE VPC by ID or name.

## Example Usage

```hcl
data "lattice_vpc" "existing" {
  name = "mgmt"
}

resource "lattice_ipam_pool" "mgmt" {
  name        = "mgmt-pool"
  bridge      = data.lattice_vpc.existing.bridge
  subnet      = data.lattice_vpc.existing.cidr
  gateway     = data.lattice_vpc.existing.gateway
  range_start = "10.0.0.50"
  range_end   = "10.0.0.200"
}
```

## Argument Reference

One of `id` or `name` must be provided.

- `id` (Optional) — VPC UUID.
- `name` (Optional) — VPC name.

## Attribute Reference

- `id` — VPC UUID.
- `name` — VPC name.
- `cidr` — IPv4 CIDR.
- `bridge` — Linux bridge name.
- `gateway` — Gateway IP.
- `status` — VPC status.
