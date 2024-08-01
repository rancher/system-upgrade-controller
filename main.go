//go:generate go run pkg/codegen/cleanup/cleanup.go
//go:generate rm -rf pkg/generated
//go:generate go run pkg/codegen/codegen.go

package main

// Copyright 2019 Rancher Labs, Inc.
// SPDX-License-Identifier: Apache-2.0

import (
	"fmt"
	"os"
	"time"

	"github.com/rancher/system-upgrade-controller/pkg/upgrade"
	"github.com/rancher/system-upgrade-controller/pkg/version"
	"github.com/rancher/wrangler/v3/pkg/signals"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	debug                               bool
	kubeConfig, masterURL               string
	namespace, name, serviceAccountName string
	threads                             int
)

func main() {
	app := cli.NewApp()
	app.Name = "system-upgrade-controller"
	app.Usage = "in ur system controllin ur upgradez"
	app.Version = fmt.Sprintf("%s (%s)", version.Version, version.GitCommit)
	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:        "debug",
			EnvVar:      "SYSTEM_UPGRADE_CONTROLLER_DEBUG",
			Destination: &debug,
		},
		cli.StringFlag{
			Name:   "kubeconfig",
			EnvVar: "SYSTEM_UPGRADE_CONTROLLER_KUBE_CONFIG",
			//Value:  "${HOME}/.kube/config",
			Destination: &kubeConfig,
		},
		cli.StringFlag{
			Name:        "master",
			EnvVar:      "SYSTEM_UPGRADE_CONTROLLER_MASTER_URL",
			Destination: &masterURL,
		},
		cli.StringFlag{
			Name:        "name",
			EnvVar:      "SYSTEM_UPGRADE_CONTROLLER_NAME",
			Required:    true,
			Destination: &name,
		},
		cli.StringFlag{
			Name:        "namespace",
			EnvVar:      "SYSTEM_UPGRADE_CONTROLLER_NAMESPACE",
			Required:    true,
			Destination: &namespace,
		},
		cli.StringFlag{
			Name:        "service-account",
			Hidden:      true,
			Destination: &serviceAccountName,
		},
		cli.IntFlag{
			Name:        "threads",
			EnvVar:      "SYSTEM_UPGRADE_CONTROLLER_THREADS",
			Value:       2,
			Destination: &threads,
		},
	}
	app.Action = Run

	if serviceAccountName != "" {
		logrus.Warn("deprecated flag `service-account` is ignored")
	}

	if err := app.Run(os.Args); err != nil {
		logrus.Fatal(err)
	}
}

func Run(_ *cli.Context) {
	if debug {
		logrus.SetLevel(logrus.DebugLevel)
		logrus.SetReportCaller(true)
	}
	cfg, err := clientcmd.BuildConfigFromFlags(masterURL, kubeConfig)
	if err != nil {
		logrus.Fatal(err)
	}
	ctl, err := upgrade.NewController(cfg, namespace, name, 2*time.Hour)
	if err != nil {
		logrus.Fatal(err)
	}
	ctx := signals.SetupSignalContext()
	if err := ctl.Start(ctx, threads); err != nil {
		logrus.Fatalf("Error starting: %v", err)
	}
	<-ctx.Done()
}
