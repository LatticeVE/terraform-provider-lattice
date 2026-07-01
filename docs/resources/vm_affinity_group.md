# lattice_vm_affinity_group

Assigns a VM to an affinity or anti-affinity group.

```hcl
resource "lattice_vm_affinity_group" "web_1" {
  vm_id             = lattice_vm.web_1.id
  affinity_group_id = lattice_affinity_group.web.id
}
```
