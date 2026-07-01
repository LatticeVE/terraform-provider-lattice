# lattice_vm_security_group

Attaches a `lattice_security_group` to a `lattice_vm`. Keeping the relationship separate gives Terraform the dependency order needed to detach it before either object is destroyed.

```hcl
resource "lattice_vm_security_group" "web" {
  vm_id             = lattice_vm.web.id
  security_group_id = lattice_security_group.web.id
}
```
