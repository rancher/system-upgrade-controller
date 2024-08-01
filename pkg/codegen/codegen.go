package main

import (
	"github.com/rancher/wrangler/v3/pkg/controller-gen/args"

	v1 "github.com/rancher/system-upgrade-controller/pkg/apis/upgrade.cattle.io/v1"
	controllergen "github.com/rancher/wrangler/v3/pkg/controller-gen"
)

func main() {
	controllergen.Run(args.Options{
		Boilerplate:   "hack/boilerplate.go.txt",
		OutputPackage: "github.com/rancher/system-upgrade-controller/pkg/generated",
		Groups: map[string]args.Group{
			"upgrade.cattle.io": {
				Types: []interface{}{
					v1.Plan{},
				},
				GenerateTypes:   true,
				GenerateClients: true,
			},
		},
	})
}
