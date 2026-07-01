# lattice_affinity_group

Creates an `affinity` or `anti-affinity` VM placement group.

```hcl
resource "lattice_affinity_group" "web" {
  name   = "web-spread"
  policy = "anti-affinity"
}
```
