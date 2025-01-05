variable "proxmox_api_token_id" {
    type = string
    description = "API token ID for Proxmox user with power user privileges"
}

variable "proxmox_api_token_secret" {
    type = string
    description = "API token secret for Proxmox user with power user privileges"
}

variable "proxmox_api_token_username" {
    type = string
    description = "API token usrname for Proxmox user with power user privileges"
}

variable "proxmox_endpoint" {
    type = string
    description = "The endpoint URL of the Proxmox server"
}

variable "proxmox_node" {
    type = string
    description = "The name of the Proxmox VE primary node"
}

variable "proxmox_uses_untrusted_ssl_cert" {
    type = bool
    description = "Whether or not the Proxmox server uses a self-signed or untrusted SSL certificate for its interface"
    default = true
}