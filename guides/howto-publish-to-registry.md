# Publishing to the Terraform Registry

This repository is used for development. For Registry publishing, the provider binary is expected under a repository named **`terraform-provider-bqutils`** (provider type `bqutils`).

## Typical publishing flow

1. Move/push the code to `github.com/<namespace>/terraform-provider-bqutils`
2. Create a GPG signing key and upload the public key to the Terraform Registry
3. Tag a release (`v0.1.0`) and use GoReleaser (see `.goreleaser.yml`) via the Release GitHub Action
4. Add the provider on [registry.terraform.io](https://registry.terraform.io/publish/provider)
