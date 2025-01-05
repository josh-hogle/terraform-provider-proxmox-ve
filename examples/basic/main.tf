terraform {
  required_providers {
    proxmoxve = {
      source = "registry.terraform.io/josh-hogle/proxmox-ve"
    }
  }
}

provider "proxmoxve" {
  endpoint = var.proxmox_endpoint
  api_token_username = var.proxmox_api_token_username
  api_token_id = var.proxmox_api_token_id
  api_token_secret = var.proxmox_api_token_secret
  ignore_untrusted_ssl_certificate = var.proxmox_uses_untrusted_ssl_cert
}

data "proxmoxve_vm_config" "test" {
  filter = {
    node_name = var.proxmox_node
    vm_id = 100
  }
}

output "vm_test_config" {
  value = data.proxmoxve_vm_config.test
}
