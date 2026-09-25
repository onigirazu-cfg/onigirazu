# Infrastructure names come from CI variables; nothing about the estate is
# hard-coded in this public repository.

variable "vsphere_server" { type = string }
variable "vsphere_user" { type = string }
variable "vsphere_password" {
  type      = string
  sensitive = true
}

variable "datacenter" { type = string }
variable "cluster" { type = string }
variable "host" { type = string }
variable "datastore" { type = string }
variable "network" { type = string }
variable "folder" { type = string }
variable "library" {
  description = "Content library whose [latest] items name the templates"
  type        = string
}

variable "images" {
  description = "Short OS key => content library item name"
  type        = map(string)
}

variable "run_id" {
  description = "Unique per run; part of every VM name"
  type        = string
  validation {
    condition     = can(regex("^[a-z0-9-]{1,24}$", var.run_id))
    error_message = "run_id: 1-24 chars of a-z, 0-9, -."
  }
}

variable "run_url" {
  type    = string
  default = "local run"
}

variable "public_key" {
  description = "One-time SSH public key for the e2e user"
  type        = string
}

variable "ttl_hours" {
  type    = number
  default = 3
}

variable "cpus" {
  type    = number
  default = 2
}

variable "memory_mb" {
  type    = number
  default = 2048
}
