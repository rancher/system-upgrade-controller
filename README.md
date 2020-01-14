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
  name: k3os-latest
  namespace: k3os-system
spec:
  concurrency: 1
  channel: https://github.com/rancher/k3os/releases/latest
  version: v0.9.0-dev
  nodeSelector:
    matchExpressions:
      - {key: plan.upgrade.cattle.io/k3os-latest, operator: Exists}
      - {key: k3os.io/mode, operator: NotIn, values: ["live"]}
  drain:
    force: true
  upgrade:
    image: dweomer/k3os
    command: [k3os, --debug]
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

This plan specifies via `concurrency` that only one node at a time in the cluster can be applying this plan.
It specifies a `channel` URL that should adhere to the simple contract exhibited by Github latest release browser URLs
which is to simply return an HTTP 302 with Location header pointing to the latest release tag. The controller will attempt to
resolve `channel` URL redirects every 15 minutes by default. If, as in this example, the `version` is specified then 
`channel` resolution is skipped and only the specified `version` is honored.
To specify which nodes in the cluster are eligible for application of this Plan a `nodeSelector` entry must be provided.
The format of `nodeSelector` is the same as `nodeSelectorTerms` in the `nodeAffinity` section of the `affinity` spec for
Pods.
Not shown in this example is a `cordon` boolean, default `false`, that would indicate that `kubectl cordon` should be
run against the node prior to invoking the upgrade.
Instead we have a non-nil `drain` (which will set the same unscheduleable taint as `cordon`) with parameters
corresponding to those used for `kubectl drain` minus the selectors. Additionally, the `deleteLocalData` and 
`ignoreDaemonSets` parameters both default to `true` if, as in this example, `drain` is specified.
Both the `drain` and `cordon` `kubectl` invocations are run in init containers for the Pod.
Finally, to specify the `upgrade`, we have a very truncated container template: `image`, `command`, and `args`.

## Building
`make`

### Local Execution

Use `./bin/system-upgrade-controller`.

Also see [`manifests/system-upgrade-controller.yaml`](manifests/system-upgrade-controller.yaml) that spells out what a
"typical" deployment might look like with default environment variables that parameterize various operational aspects
of the controller and the resources spawned by it.

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
