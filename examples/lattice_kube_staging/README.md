# LatticeKube staging workflow

This example validates the real Terraform lifecycle for a LatticeKube cluster:

1. ensure a k3s-compatible kernel exists without duplicate imports,
2. import a k3s rootfs image,
3. create a VPC-only cluster by default,
4. scale workers by changing `worker_count`,
5. destroy the cluster and Terraform-owned artifacts cleanly.

The optional public IP path is enabled by setting `public_pool_cidr`.

## Prerequisites

- Staging controller reachable from this machine.
- API key with permission to manage kernels, rootfs images, public IP pools, and Kubernetes clusters.
- At least one Firecracker-capable agent.
- k3s kernel/rootfs releases available from `latticeve-k3s-images`.

For local provider development, use a Terraform CLI config with a `dev_overrides`
entry pointing at your locally built provider binary. Otherwise Terraform uses
the released provider declared in `main.tf`.

## Configure

```bash
cp terraform.tfvars.example terraform.tfvars
```

Edit:

- `lattice_endpoint`
- `lattice_api_key`
- `resource_prefix`
- optionally `kernel_version` and `rootfs_version`

Leave `public_pool_cidr = ""` for the first run. That validates the VPC-only
cluster path where LatticeVE auto-creates a managed VPC and no public IP pool is
used.

## Run VPC-only create

```bash
terraform init
terraform apply
terraform output
```

Expected:

- cluster reaches `ready`,
- `cluster_vpc.managed` is `true`,
- `cluster_public_ip` is empty,
- `kernel.managed` is `false` if the matching kernel already existed, otherwise `true`.

Run `terraform apply` again. It should be a no-op and must not import another
matching k3s kernel.

## Scale workers

Change:

```hcl
worker_count = 2
```

Then run:

```bash
terraform apply
```

Expected:

- one new worker is added,
- cluster returns to `ready`.

## Optional public IP coverage

Set `public_pool_cidr` to a reserved CIDR inside `public_bridge`'s connected
IPv4 subnet, excluded from upstream DHCP:

```hcl
public_bridge    = "br0"
public_pool_cidr = "10.0.7.128/28"
```

Then run `terraform apply`.

Expected:

- cluster endpoint is exposed through a public IP from the pool,
- LatticeVE CCM can allocate public IPs for Kubernetes `LoadBalancer` services.

## Destroy

```bash
terraform destroy
```

Expected:

- cluster and managed VPC are deleted,
- Terraform-created public IP pool is deleted,
- Terraform-imported k3s rootfs is deleted,
- `lattice_k3s_kernel` deletes only kernels it downloaded itself; reused kernels remain.

## Cleanup failed runs

The cleanup tool is dry-run by default:

```bash
../../tools/cleanup_lattice_kube_staging.py --prefix tf-kube-staging
```

Execute cleanup:

```bash
../../tools/cleanup_lattice_kube_staging.py --prefix tf-kube-staging --execute
```

The tool reads:

- `LATTICE_ENDPOINT`
- `LATTICE_API_KEY`
- optional `LATTICE_INSECURE=true`

It deletes resources whose name or description starts with the prefix, in this
order: Kubernetes clusters, VMs, public IPs, public IP pools, then VPCs.
