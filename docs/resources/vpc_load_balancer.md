# lattice_vpc_load_balancer

Manages a LatticeVE VPC load balancer.

Load balancers are separate resources instead of nested `lattice_vpc` blocks so Terraform can manage their lifecycle without replacing or churning the parent VPC.

## Example Usage

```hcl
resource "lattice_vpc_load_balancer" "web" {
  vpc_id           = lattice_vpc.app.id
  name             = "web"
  port             = 443
  protocol         = "https"
  certificate_id   = lattice_lb_certificate.web.id
  backend_protocol = "http"

  backends = [
    {
      vm_id   = lattice_vm.web_1.id
      port    = 8080
      weight  = 100
    },
    {
      address = "10.10.0.11:8080"
      weight  = 100
    },
  ]
}
```

## Argument Reference

- `vpc_id` (Required, Forces replacement) — VPC ID.
- `name` (Required, Forces replacement) — Load balancer name.
- `port` (Required, Forces replacement) — Frontend listen port.
- `protocol` (Required, Forces replacement) — Frontend protocol: `tcp`, `http`, or `https`.
- `certificate_id` (Optional, Forces replacement) — Required for `https`; references `lattice_lb_certificate.id`.
- `backend_protocol` (Optional, Forces replacement) — Backend protocol: `tcp` or `http`. Defaults server-side from `protocol`.
- `backends` (Required, Forces replacement) — Backend targets:
  - `vm_id` + `port` — Preferred managed target. LatticeVE reconciles the backend if the VM's VPC address changes.
  - `address` — Explicit backend address in `ip:port` form for unmanaged targets.
  - `weight` (Optional, Computed) — Backend weight. Defaults to `1`.

## Attribute Reference

- `id` — Load balancer UUID.
- `backends[*].id` — Backend UUID assigned by LatticeVE.
