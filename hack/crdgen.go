package main

import (
	_ "github.com/rancher/system-upgrade-controller/pkg/generated/controllers/upgrade.cattle.io/v1"
	"github.com/rancher/system-upgrade-controller/pkg/upgrade/plan"
	"github.com/rancher/wrangler/pkg/crd"
	"os"
)

func main() {
	planCrd, err := plan.CRD()
	if err != nil {
		print(err)
		return
	}
	crd.Print(os.Stdout, []crd.CRD{*planCrd})
}
