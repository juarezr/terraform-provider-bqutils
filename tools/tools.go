//go:build tools

package tools

import (
	_ "github.com/boumenot/gocover-cobertura"
	_ "github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs"
	_ "github.com/wadey/gocovmerge"
)
