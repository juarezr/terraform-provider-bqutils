# How to test the provider locally (without publishing)

## Question

How can I test this provider on this computer without publishing to the Terraform Registry?

## Answer

You can run it fully offline with either a **filesystem mirror install** (available via the Makefile) or **dev overrides** (best for day-to-day coding). Automated tests are a third option that needs no Registry at all.

### Option A — `make install` (filesystem mirror)

```bash
cd ~/src/terraform-provider-bqutils
make install
```

That places the binary at:

`~/.terraform.d/plugins/registry.terraform.io/juarezr/bqutils/0.1.0/linux_amd64/terraform-provider-bqutils`

Then in a test directory (e.g. `examples/routine`):

```hcl
terraform {
  required_providers {
    bqutils = {
      source  = "juarezr/bqutils"
      version = "0.1.0"
    }
  }
}

provider "bqutils" {}
```

```bash
terraform init
terraform plan
```

Terraform loads the local binary instead of downloading from the Registry.

### Option B — `dev_overrides` (recommended while developing)

Create or edit `~/.terraformrc`:

```hcl
provider_installation {
  dev_overrides {
    "juarezr/bqutils" = "~/.terraform.d/plugins/registry.terraform.io/juarezr/bqutils/0.1.0/linux_amd64/terraform-provider-bqutils"
  }
  direct {}
}
```

Rebuild after changes:

```bash
cd ~/src/github/mine/terraform-provider-bigquery-utils
go build -o terraform-provider-bqutils .
```

With overrides, **skip `terraform init` for this provider** (Terraform will warn that it is in override mode). Just run `terraform plan` / `terraform apply` in your test module.

Use a `required_providers` block with `source = "juarezr/bqutils"` (the version can be omitted under overrides).

### Option C — automated tests (no Registry)

```bash
go test ./internal/sqlparse/ -v
TF_ACC=1 go test ./internal/provider/ -v -count=1
```

These use the in-process provider server and do not need a published release.

---

**Practical tip:** use **Option B** while iterating on code; use **Option A** when you want a real `terraform init` experience. For a quick smoke test, `examples/routine` only needs `bqutils` (no Google credentials required).
