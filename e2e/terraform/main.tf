terraform {
  required_version = ">= 1.6"
  required_providers {
    vsphere = {
      source  = "vmware/vsphere"
      version = "~> 2.16"
    }
  }
}

provider "vsphere" {
  vsphere_server       = var.vsphere_server
  user                 = var.vsphere_user
  password             = var.vsphere_password
  allow_unverified_ssl = true
}

data "vsphere_datacenter" "dc" {
  name = var.datacenter
}

data "vsphere_compute_cluster" "cluster" {
  name          = var.cluster
  datacenter_id = data.vsphere_datacenter.dc.id
}

data "vsphere_host" "host" {
  name          = var.host
  datacenter_id = data.vsphere_datacenter.dc.id
}

data "vsphere_datastore" "ds" {
  name          = var.datastore
  datacenter_id = data.vsphere_datacenter.dc.id
}

data "vsphere_network" "net" {
  name          = var.network
  datacenter_id = data.vsphere_datacenter.dc.id
}

data "vsphere_content_library" "lib" {
  name = var.library
}

data "vsphere_content_library_item" "image" {
  for_each   = var.images
  name       = each.value
  library_id = data.vsphere_content_library.lib.id
  type       = "vm-template"
}

locals {
  expires = timeadd(plantimestamp(), "${var.ttl_hours}h")
}

resource "vsphere_virtual_machine" "vm" {
  for_each = var.images

  # tmp- prefix and the folder mark these VMs as disposable; the janitor
  # deletes anything in the folder with this prefix once it is past its TTL
  name             = "tmp-e2e-onigirazu-${var.run_id}-${each.key}"
  folder           = var.folder
  resource_pool_id = data.vsphere_compute_cluster.cluster.resource_pool_id
  host_system_id   = data.vsphere_host.host.id
  datastore_id     = data.vsphere_datastore.ds.id

  num_cpus = var.cpus
  memory   = var.memory_mb
  guest_id = "ubuntu64Guest"

  annotation = join("\n", [
    "TEMPORARY - onigirazu e2e test VM, deleted automatically.",
    "Run: ${var.run_url}",
    "Created: ${plantimestamp()}",
    "Safe to delete after: ${local.expires}",
  ])

  extra_config = {
    # Read once by the image's e2e-access unit on first boot
    "guestinfo.e2e_authorized_key" = var.public_key
  }

  network_interface {
    network_id   = data.vsphere_network.net.id
    adapter_type = "vmxnet3"
  }

  disk {
    label            = "disk0"
    size             = var.disk_gb
    thin_provisioned = true
    eagerly_scrub    = false
  }

  clone {
    template_uuid = data.vsphere_content_library_item.image[each.key].id

    customize {
      timeout = 20
      linux_options {
        host_name = "e2e-${each.key}"
        domain    = "e2e.invalid"
      }
      # DHCP
      network_interface {}
    }
  }

  wait_for_guest_net_timeout = 10

  lifecycle {
    ignore_changes = [annotation]
  }
}

output "hosts" {
  value = { for k, vm in vsphere_virtual_machine.vm : k => vm.default_ip_address }
}
