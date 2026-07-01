# lattice_ipam_lease

Creates a static DHCP lease in a `lattice_ipam_pool`.

```hcl
resource "lattice_ipam_lease" "web" {
  pool_id  = lattice_ipam_pool.main.id
  mac      = "02:00:00:00:00:10"
  ip       = "192.168.100.10"
  hostname = "web-01"
  vm_id    = lattice_vm.web.id
}
```

The address must be inside the pool subnet and cannot be the gateway.
