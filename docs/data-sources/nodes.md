# lattice_nodes (Data Source)

Lists LatticeVE host nodes. Filter by `arch` to discover nodes suitable for placing VMs that require a specific CPU architecture (amd64 or arm64). Use the returned node names in `lattice_vm.node` to pin a VM to a specific host, or pass `arch` directly on `lattice_vm` to let the scheduler pick.

## Example Usage

```hcl
# All online nodes
data "lattice_nodes" "all" {}

# Only arm64 nodes
data "lattice_nodes" "arm64" {
  arch = "arm64"
}

# Use arch filter on the VM — scheduler picks any matching node
resource "lattice_vm" "arm_worker" {
  name         = "arm-worker-01"
  cpus         = 8
  memory_mb    = 16384
  image_id     = data.lattice_image.ubuntu_arm.id
  boot_disk_gb = 50
  arch         = "arm64"
  nics         = [{ bridge = lattice_vpc.main.bridge }]
}

# Pin to a specific node by name
resource "lattice_vm" "pinned" {
  name         = "gpu-workload"
  cpus         = 16
  memory_mb    = 32768
  image_id     = data.lattice_image.ubuntu.id
  boot_disk_gb = 100
  node         = data.lattice_nodes.arm64.nodes[0].name
  nics         = [{ bridge = lattice_vpc.main.bridge }]
}

output "arm64_node_count" {
  value = length(data.lattice_nodes.arm64.nodes)
}
```

## Argument Reference

- `arch` (Optional) — Filter by CPU architecture: `amd64` or `arm64`. Omit to return all nodes regardless of arch.

## Attribute Reference

Each entry in `nodes` contains:

- `id` — Node UUID.
- `name` — Node hostname (e.g. `kvm-arm-01`). Pass this to `lattice_vm.node` to pin placement.
- `arch` — CPU architecture: `amd64` or `arm64`.
- `status` — Node status: `online`, `offline`, or `maintenance`.
- `cpus` — Total logical CPU count.
- `memory_mb` — Total RAM in MiB.
- `memory_used_mb` — RAM currently allocated to VMs in MiB.
- `storage_gb` — Total local storage in GiB.
- `storage_used_gb` — Local storage currently in use in GiB.

## Notes

- Nodes with `status = "offline"` or `"maintenance"` are returned by this data source but the scheduler will not place new VMs on them. Use the `status` attribute in expressions to filter them out if needed.
- `arch` on `lattice_vm` is a scheduler *hint* — the controller will reject the create if no online node of the requested arch is available.
- `node` on `lattice_vm` is a hard pin — the controller will reject the create if that specific node is offline or at capacity.
