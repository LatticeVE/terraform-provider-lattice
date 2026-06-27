# lattice_security_group

Manages a LatticeVE security group with stateless ingress/egress rules.

## Example Usage

```hcl
resource "lattice_security_group" "web" {
  name        = "web-sg"
  description = "Web tier"

  rules = [
    {
      direction = "ingress"
      protocol  = "tcp"
      port_from = 443
      port_to   = 443
      cidr      = "0.0.0.0/0"
      action    = "accept"
      priority  = 100
    },
    {
      direction = "egress"
      protocol  = "all"
      port_from = 0
      port_to   = 0
      cidr      = "0.0.0.0/0"
      action    = "accept"
      priority  = 1
    },
  ]
}
```

## Argument Reference

- `name` (Required) — Security group name.
- `description` (Optional) — Human-readable description.
- `rules` (Optional) — Ordered list of rules. Each block:
  - `direction` (Required) — `ingress` or `egress`.
  - `protocol` (Required) — `tcp`, `udp`, `icmp`, or `all`.
  - `port_from` (Optional) — Start of port range. `0` means all ports.
  - `port_to` (Optional) — End of port range.
  - `cidr` (Required) — CIDR to match.
  - `action` (Required) — `accept` or `drop`.
  - `priority` (Optional) — Rule priority (higher wins).
  - `id` (Computed) — Rule UUID.

## Attribute Reference

- `id` — Security group UUID.
- `created_at` — ISO 8601 creation timestamp.
