[![Terraform & OpenTofu Acceptance Tests](https://github.com/jfrog/terraform-provider-myjfrog/actions/workflows/acceptance-tests.yml/badge.svg)](https://github.com/jfrog/terraform-provider-myjfrog/actions/workflows/acceptance-tests.yml)

# Terraform Provider for MyJFrog

## Quick Start

Create a new Terraform file with `myjfrog` resource:

### HCL Example

```terraform
# Required for Terraform 1.0 and later
terraform {
  required_providers {
    myjfrog = {
      source  = "jfrog/myjfrog"
      version = "1.0.0"
    }
  }
}

provider "myjfrog" {
  // supply JFROG_MYJFROG_API_TOKEN as env var
}

resource "myjfrog_ip_allowlist" "myjfrog-ip-allowlist" {
  server_name = "my-jpd-server-name"
  ips = [
    "1.1.1.7/1",
    "2.2.2.7/1",
  ]
}
```

Initialize Terrform:
```sh
$ terraform init
```

Plan (or Apply):
```sh
$ terraform plan
```

Detailed documentation of the resource and attributes are on [Terraform Registry](https://registry.terraform.io/providers/jfrog/myjfrog/latest/docs).

## Versioning

In general, this project follows [semver](https://semver.org/) as closely as we can for tagging releases of the package. We've adopted the following versioning policy:

* We increment the **major version** with any incompatible change to functionality, including changes to the exported Go API surface or behavior of the API.
* We increment the **minor version** with any backwards-compatible changes to functionality.
* We increment the **patch version** with any backwards-compatible bug fixes.

## Contributors

See the [contribution guide](CONTRIBUTIONS.md).

## License

Copyright (c) 2025 JFrog.

Apache 2.0 licensed, see [LICENSE][LICENSE] file.

[LICENSE]: ./LICENSE
