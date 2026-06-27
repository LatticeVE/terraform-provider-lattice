# lattice_vpc

Manages a LatticeVE Virtual Private Cloud (VPC). A VPC is an isolated L2/L3 network with its own bridge, CIDR, gateway, firewall, and port-forward rules.

## Example Usage

```hcl
resource "lattice_vpc" "app" {
  name           = "app-vpc"
  cidr           = "10.10.0.0/24"
  default_action = "drop"

  firewall_rules = [
    {
      direction = "ingress"
      proto     = "tcp"
      port      = "443"
      cidr      = "0.0.0.0/0"
      action    = "accept"
      desc      = "HTTPS"
    },
  ]

  port_forwards = [
    {
      proto     = "tcp"
      ext_port  = 8443
      dest_ip   = "10.10.0.10"
      dest_port = 443
    },
  ]
}
```

## Argument Reference

- `name` (Required) — VPC display name.
- `cidr` (Optional, Forces new resource) — IPv4 CIDR block, e.g. `10.100.1.0/24`.
- `cidr_v6` (Optional, Forces new resource) — IPv6 CIDR block.
- `default_action` (Optional) — Default firewall action: `accept` (default) or `drop`.
- `port_forwards` (Optional) — List of port-forward rules. Each block:
  - `proto` (Required) — `tcp` or `udp`.
  - `ext_port` (Required) — External port on the VPC gateway.
  - `dest_ip` (Required) — Destination VM IP inside the VPC.
  - `dest_port` (Required) — Destination port.
  - `desc` (Optional) — Description.
  - `id` (Computed) — Rule UUID.
- `firewall_rules` (Optional) — List of stateless firewall rules. Each block:
  - `direction` (Required) — `ingress`, `egress`, or `both`.
  - `proto` (Required) — `tcp`, `udp`, `icmp`, or `all`.
  - `port` (Optional) — Port or range, e.g. `80` or `8080-8090`. Empty means all ports.
  - `cidr` (Required) — Source/destination CIDR.
  - `action` (Required) — `accept` or `drop`.
  - `desc` (Optional) — Description.
  - `id` (Computed) — Rule UUID.

## Attribute Reference

- `id` — VPC UUID.
- `bridge` — Linux bridge interface name on the host.
- `gateway` — IPv4 gateway address.
- `gateway_v6` — IPv6 gateway address.
- `status` — VPC lifecycle status.
