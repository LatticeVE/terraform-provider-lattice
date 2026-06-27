# lattice_kube_releases (Data Source)

Lists all available Talos Linux releases synced from GitHub by LatticeVE.

## Example Usage

```hcl
data "lattice_kube_releases" "all" {}

# Pin to the latest available release
resource "lattice_kube_cluster" "prod" {
  talos_version = data.lattice_kube_releases.all.releases[0].version
  k8s_version   = data.lattice_kube_releases.all.releases[0].k8s_version
  # ... other arguments ...
}
```

## Attribute Reference

- `releases` — Ordered list of releases (newest first). Each entry:
  - `version` — Talos version tag, e.g. `v1.9.0`.
  - `k8s_version` — Default Kubernetes version bundled with this Talos release.
  - `published_at` — ISO 8601 release timestamp.
