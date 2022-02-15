package main

import (
	"os"

	v1 "github.com/rancher/system-upgrade-controller/pkg/apis/upgrade.cattle.io/v1"
	_ "github.com/rancher/system-upgrade-controller/pkg/generated/controllers/upgrade.cattle.io/v1"
	"github.com/rancher/wrangler/pkg/crd"
)

func main() {
	plan := crd.NamespacedType("Plan.upgrade.cattle.io/v1").
		WithSchemaFromStruct(v1.Plan{}).
		WithColumn("Image", ".spec.upgrade.image").
		WithColumn("Channel", ".spec.channel").
		WithColumn("Version", ".spec.version")
	crd.Print(os.Stdout, []crd.CRD{plan})
}
