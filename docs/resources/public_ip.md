# lattice_public_ip

Allocates a public IP from a `lattice_public_ip_pool`. Optionally enables static NAT (1:1 DNAT) to a private IP inside a VPC.

## Example Usage

```hcl
resource "lattice_public_ip" "web" {
  pool_id     = lattice_public_ip_pool.main.id
  description = "web-01 public IP"
  private_ip  = "10.10.0.10"
}
```

## Argument Reference

- `pool_id` (Required, Forces new resource) — ID of the `lattice_public_ip_pool` to allocate from.
- `description` (Optional) — Human-readable label.
- `private_ip` (Optional) — Private IP inside a VPC. When set, a static NAT rule (1:1 DNAT) is created so inbound traffic to the public IP is forwarded here. Clearing this value removes the NAT rule.

## Attribute Reference

- `id` — Public IP UUID.
- `ip` — Allocated IPv4 address.
- `created_at` — ISO 8601 creation timestamp.
