variable "lattice_endpoint" {
  description = "LatticeVE controller URL, for example https://10.0.7.48:8006."
  type        = string
}

variable "lattice_api_key" {
  description = "LatticeVE API key."
  type        = string
  sensitive   = true
}

variable "lattice_insecure" {
  description = "Allow self-signed controller TLS certificates."
  type        = bool
  default     = true
}

variable "resource_prefix" {
  description = "Unique prefix for all staging resources. Use a short disposable value, for example tf-kube-20260710."
  type        = string
  default     = "tf-kube-staging"
}

variable "arch" {
  description = "Cluster node architecture."
  type        = string
  default     = "amd64"
}

variable "kernel_version" {
  description = "Optional exact k3s kernel version. Null selects newest discovered for arch."
  type        = string
  default     = null
}

variable "rootfs_version" {
  description = "Optional exact k3s rootfs version. Null selects newest discovered for arch."
  type        = string
  default     = null
}

variable "public_bridge" {
  description = "External bridge for optional public IP pool."
  type        = string
  default     = "br0"
}

variable "public_pool_cidr" {
  description = "Optional public IP pool CIDR. Empty string keeps the cluster VPC-only."
  type        = string
  default     = ""
}

variable "cp_count" {
  description = "Control-plane count. Must be 1, 3, or 5."
  type        = number
  default     = 1
}

variable "worker_count" {
  description = "Worker count. Increase this to validate scale-out."
  type        = number
  default     = 1
}

variable "cp_vcpus" {
  type    = number
  default = 2
}

variable "cp_memory_mb" {
  type    = number
  default = 4096
}

variable "cp_disk_gb" {
  type    = number
  default = 20
}

variable "worker_vcpus" {
  type    = number
  default = 2
}

variable "worker_memory_mb" {
  type    = number
  default = 4096
}

variable "worker_disk_gb" {
  type    = number
  default = 20
}
