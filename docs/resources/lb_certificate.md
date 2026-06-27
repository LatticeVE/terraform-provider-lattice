# lattice_lb_certificate

Manages a TLS certificate for LatticeVE VPC load balancers.

LatticeVE stores the private key encrypted at rest and never returns it from the API. Terraform therefore keeps `key_pem` as sensitive state.

## Example Usage

```hcl
resource "lattice_lb_certificate" "web" {
  name        = "web-cert"
  description = "Public certificate for web traffic"

  cert_pem  = file("${path.module}/fullchain.pem")
  key_pem   = sensitive(file("${path.module}/privkey.pem"))
  chain_pem = file("${path.module}/chain.pem")
}
```

## Argument Reference

- `name` (Required) — Certificate name.
- `description` (Optional) — Human-readable description.
- `cert_pem` (Required, Sensitive) — Leaf certificate PEM.
- `key_pem` (Required, Sensitive) — Private key PEM.
- `chain_pem` (Optional, Sensitive) — Intermediate certificate chain PEM.

## Attribute Reference

- `id` — Certificate UUID.
- `subject` — Parsed certificate subject.
- `dns_names` — Parsed DNS subject alternative names.
- `not_before` — Certificate validity start timestamp.
- `not_after` — Certificate validity end timestamp.
- `fingerprint` — Certificate fingerprint.
- `created_at` — Creation timestamp.
- `updated_at` — Last update timestamp.
