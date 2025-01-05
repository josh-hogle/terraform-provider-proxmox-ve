# Terraform Provider for Proxmox

This repository is a custom Terraform provider for working with Proxmox.

Please note that this project is in a very early stage and subject to frequent changes.

This provider is currently in active development and is **NOT** ready for production use.

## Requirements

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.0
- [Go](https://golang.org/doc/install) >= 1.22

## Building The Provider

1. Clone the repository
1. Enter the repository directory
1. Build the provider using the Go `install` command:

```shell
go install
```

## Using the provider

1. You'll need to update your `$HOME/.terraformrc` file with the following:

   ```hcl
   provider_installation {
     dev_overrides {
       "registry.terraform.io/josh-hogle/proxmox-ve" = "/your/$GOPATH/bin/folder"
     }
     direct {}
   }
   ```

1. In your `main.tf` file, include the following:

   ```hcl
   terraform {
     required_providers {
       proxmox_ve = {
         source = "registry.terraform.io/josh-hogle/proxmox-ve"
       }
     }
   }

   provider "proxmox_ve" {
     endpoint = "https://hostname_or_ip:port"
     api_token_username = "user@realm"
     api_token_id = "token_id"
     api_token_secret = "00000000-0000-0000-0000-000000000000"
     ignore_untrusted_ssl_certificate = true
   }
   ```

## Developing the Provider

If you wish to work on the provider, you'll first need [Go](http://www.golang.org) installed on your machine (see [Requirements](#requirements) above).

To compile the provider, run `go install`. This will build the provider and put the provider binary in the `$GOPATH/bin` directory.
