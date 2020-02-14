# System Upgrade Controller

A ***S***ystem ***U***pgrade ***C***ontroller for ***K***ubernetes that doesn't S.U.C.K.

## Introduction

This project aims to provide a general-purpose, Kubernetes-native upgrade controller (for nodes).
It introduces a new CRD, the **Plan**, for defining any and all of your upgrade policies/requirements.
For up-to-date details on defining a plan please review [v1/types.go](pkg/apis/upgrade.cattle.io/v1/types.go).

The Controller manages Plans by selecting Nodes to run upgrade Jobs on.
A Plan defines which Nodes are eligible by specifying label selector.
When a Job has run to completion successfully the Controller will label the Node on which it ran
according to the Plan that was applied by the Job.

### Considerations

Purporting to support general-purpose node upgrades (essentially, arbitrary mutations) this controller attempts
minimal imposition of opinion. Our design constraints, such as they are, follow:

- Content delivery via container image a.k.a. container command pattern
- Operator-overridable command(s)
- A very privileged job/pod/container:
  - Host IPC, PID
  - CAP_SYS_BOOT
  - Host root mounted at `/host` (read/write)
- Optional opt-in/opt-out via node labels
- Optional cordon/drain a la `kubectl`

_Additionally, one should take care when defining upgrades by ensuring that such are idempotent--**there be dragons**._

### Example Upgrade Plan

Below is example Plan in development for [k3OS](https://github.com/rancher/k3os) that implements something like an
`rsync` of content from the container image to the host, preceded by a remount if necessary, immediately followed by a reboot.

```
---
apiVersion: upgrade.cattle.io/v1
kind: Plan

metadata:
  # This `name` should be short but descriptive.
  name: k3os-latest

  # The same `namespace` as is used for the system-upgrade-controller Deployment.
  namespace: k3os-system

spec:
  # The maximum number of concurrent nodes to apply this update on.
  concurrency: 1

  # The value for `channel` is assumed to be a URL that returns HTTP 302 with the last path element of the value
  # returned in the Location header assumed to be an image tag.
  channel: https://github.com/rancher/k3os/releases/latest

  # Providing a value for `version` will prevent polling/resolution of the `channel` if specified.
  # version: v0.9.0-dev

  # Select which nodes this plan can be applied to.
  nodeSelector:
    matchExpressions:
      # This limits application of this upgrade only to nodes that have opted in by applying this label.
      # Additionally, a value of `disabled` for this label on a node will cause the controller to skip over the node.
      # NOTICE THAT THE NAME PORTION OF THIS LABEL MATCHES THE PLAN NAME. This is related to the fact that the
      # system-upgrade-controller will tag the node with this very label having the value of the applied plan.status.latestHash.
      - {key: plan.upgrade.cattle.io/k3os-latest, operator: Exists}
      # This label is set by k3OS, therefore a node without it should not apply this upgrade.
      - {key: k3os.io/mode, operator: Exists}
      # Additionally, do not attempt to upgrade nodes booted from "live" CDROM.
      - {key: k3os.io/mode, operator: NotIn, values: ["live"]}

  # The service account for the pod to use. As with normal pods, if not specified the `default` service account from the namespace will be assigned.
  # See https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/
  serviceAccountName: k3os-upgrade

  # The prepare init container is run before cordon/drain which is run before the upgrade container.
  # Shares the same format as the `upgrade` container
  # prepare:
  #   image: alpine:3.11
  #   command: [sh, -c]
  #   args: [" echo '### ENV ###'; env | sort; echo '### RUN ###'; find /run/system-upgrade | sort"]

  # See https://kubernetes.io/docs/tasks/administer-cluster/safely-drain-node/#use-kubectl-drain-to-remove-a-node-from-service
  drain:
    # deleteLocalData: true
    # ignoreDaemonSets: true
    force: true

  upgrade:
    # The tag portion of the image will be overridden with the value from `.status.latestVersion` a.k.a. the resolved version.
    # SEE https://github.com/rancher/system-upgrade-controller/blob/v0.1.0/pkg/apis/upgrade.cattle.io/v1/types.go#L47
    image: rancher/k3os
    command: [k3os, --debug]
    # It is safe to specify `--kernel` on overlay installations as the destination path will not exist and so the
    # upgrade of the kernel component will be skipped (with a warning in the log).
    args:
      - upgrade
      - --kernel
      - --rootfs
      - --remount
      - --sync
      - --reboot
      - --lock-file=/host/run/k3os/upgrade.lock
      - --source=/k3os/system
      - --destination=/host/k3os/system
```

## Building

```shell script
make
```

### Local Execution

Use `./bin/system-upgrade-controller`.

Also see [`manifests/system-upgrade-controller.yaml`](manifests/system-upgrade-controller.yaml) that spells out what a
"typical" deployment might look like with default environment variables that parameterize various operational aspects
of the controller and the resources spawned by it.

### End-to-End Testing

Integration tests are bundled as a [Sonobuoy plugin](https://sonobuoy.io/docs/v0.17.2/plugins/) that expects to be run within a pod.
To verify locally:

```shell script
make e2e
```

This will, via Dapper, stand up a local cluster (using docker-compose) and then run the Sonobuoy plugin against/within it.
The Sonobuoy results are parsed and a `Status: passed` results in a clean exit, whereas `Status: failed` exits non-zero.

Alternatively, if you have a working cluster and Sonobuoy installation, provided you've pushed the images (consider building with
something like `make REPO=dweomer TAG=dev`), then you can run the e2e tests thusly:

```shell script
sonobuoy run --plugin dist/artifacts/system-upgrade-controller-e2e-tests.yaml --wait
sonobuoy results $(sonobuoy retrieve)
```

## License
Copyright (c) 2019-2020 [Rancher Labs, Inc.](http://rancher.com)

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

[http://www.apache.org/licenses/LICENSE-2.0](http://www.apache.org/licenses/LICENSE-2.0)

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
