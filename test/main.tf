terraform {
  required_providers {
    lattice = {
      source  = "latticeve/lattice"
      version = "0.1.0"
    }
  }
}

provider "lattice" {
  endpoint = "https://localhost:8006"
  insecure = true
}

resource "lattice_vm" "test_vm" {
  name           = "terraform-test"
  cpus           = 2
  memory_mb      = 2048
  status         = "running"
  boot_disk_gb   = 40
  disk_interface = "scsi"

  cloud_init = {
    user_data = "echo 'hello' > /var/tmp/hello.txt"
    meta_data = "instance-id: test-id"
  }

  extra_disks = [
    {
      size_gb   = 10
      interface = "scsi"
    }
  ]

  nics = [
    {
      bridge = "virbr0"
      model  = "e1000"
    }
  ]
}
