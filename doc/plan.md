# API Reference

## Packages
- [upgrade.cattle.io/v1](#upgradecattleiov1)


## upgrade.cattle.io/v1






#### ContainerSpec



ContainerSpec is a simplified container template spec, used to configure the prepare and upgrade
containers of the Job Pod.



_Appears in:_
- [PlanSpec](#planspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `image` _string_ | Image name. If the tag is omitted, the value from .status.latestVersion will be used. |  | Required: \{\} <br /> |
| `command` _string array_ |  |  |  |
| `args` _string array_ |  |  |  |
| `envs` _[EnvVar](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#envvar-v1-core) array_ |  |  |  |
| `envFrom` _[EnvFromSource](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#envfromsource-v1-core) array_ |  |  |  |
| `volumes` _[VolumeSpec](#volumespec) array_ |  |  |  |
| `securityContext` _[SecurityContext](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#securitycontext-v1-core)_ |  |  |  |


#### Day

_Underlying type:_ _string_



_Validation:_
- Enum: [0 su sun sunday 1 mo mon monday 2 tu tue tuesday 3 we wed wednesday 4 th thu thursday 5 fr fri friday 6 sa sat saturday]

_Appears in:_
- [TimeWindowSpec](#timewindowspec)



#### DrainSpec



DrainSpec encapsulates kubectl drain parameters minus node/pod selectors. See:
- https://kubernetes.io/docs/tasks/administer-cluster/safely-drain-node/
- https://kubernetes.io/docs/reference/generated/kubectl/kubectl-commands#drain



_Appears in:_
- [PlanSpec](#planspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `timeout` _[Duration](#duration)_ |  |  |  |
| `gracePeriod` _integer_ |  |  |  |
| `deleteLocalData` _boolean_ |  |  |  |
| `deleteEmptydirData` _boolean_ |  |  |  |
| `ignoreDaemonSets` _boolean_ |  |  |  |
| `force` _boolean_ |  |  |  |
| `disableEviction` _boolean_ |  |  |  |
| `skipWaitForDeleteTimeout` _integer_ |  |  |  |
| `podSelector` _[LabelSelector](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#labelselector-v1-meta)_ |  |  |  |


#### Plan



Plan represents a set of Jobs to apply an upgrade (or other operation) to set of Nodes.



_Appears in:_
- [PlanList](#planlist)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[PlanSpec](#planspec)_ |  |  |  |
| `status` _[PlanStatus](#planstatus)_ |  |  |  |




#### PlanSpec



PlanSpec represents the user-configurable details of a Plan.



_Appears in:_
- [Plan](#plan)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `concurrency` _integer_ | The maximum number of concurrent nodes to apply this update on. |  |  |
| `jobActiveDeadlineSecs` _integer_ | Sets ActiveDeadlineSeconds on Jobs generated to apply this Plan.<br />If the Job does not complete within this time, the Plan will stop processing until it is updated to trigger a redeploy.<br />If set to 0, Jobs have no deadline. If not set, the controller default value is used. |  |  |
| `nodeSelector` _[LabelSelector](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#labelselector-v1-meta)_ | Select which nodes this plan can be applied to. |  |  |
| `serviceAccountName` _string_ | The service account for the pod to use. As with normal pods, if not specified the default service account from the namespace will be assigned. |  |  |
| `channel` _string_ | A URL that returns HTTP 302 with the last path element of the value returned in the Location header assumed to be an image tag (after munging "+" to "-"). |  |  |
| `version` _string_ | Providing a value for version will prevent polling/resolution of the channel if specified. |  |  |
| `secrets` _[SecretSpec](#secretspec) array_ | Secrets to be mounted into the Job Pod. |  |  |
| `tolerations` _[Toleration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#toleration-v1-core) array_ | Specify which node taints should be tolerated by pods applying the upgrade.<br />Anything specified here is appended to the default of:<br />- `\{key: node.kubernetes.io/unschedulable, effect: NoSchedule, operator: Exists\}` |  |  |
| `exclusive` _boolean_ | Jobs for exclusive plans cannot be run alongside any other exclusive plan. |  |  |
| `window` _[TimeWindowSpec](#timewindowspec)_ | A time window in which to execute Jobs for this Plan.<br />Jobs will not be generated outside this time window, but may continue executing into the window once started. |  |  |
| `prepare` _[ContainerSpec](#containerspec)_ | The prepare init container, if specified, is run before cordon/drain which is run before the upgrade container. |  |  |
| `upgrade` _[ContainerSpec](#containerspec)_ | The upgrade container; must be specified. |  |  |
| `cordon` _boolean_ | If Cordon is true, the node is cordoned before the upgrade container is run.<br />If drain is specified, the value for cordon is ignored, and the node is cordoned.<br />If neither drain nor cordon are specified and the node is marked as schedulable=false it will not be marked as schedulable=true when the Job completes. |  |  |
| `drain` _[DrainSpec](#drainspec)_ | Configuration for draining nodes prior to upgrade. If left unspecified, no drain will be performed. |  |  |
| `imagePullSecrets` _[LocalObjectReference](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#localobjectreference-v1-core) array_ | Image Pull Secrets, used to pull images for the Job. |  |  |
| `postCompleteDelay` _[Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#duration-v1-meta)_ | Time after a Job for one Node is complete before a new Job will be created for the next Node. |  |  |
| `priorityClassName` _string_ | Priority Class Name of Job, if specified. |  |  |


#### PlanStatus



PlanStatus represents the resulting state from processing Plan events.



_Appears in:_
- [Plan](#plan)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `conditions` _GenericCondition array_ | `LatestResolved` indicates that the latest version as per the spec has been determined.<br />`Validated` indicates that the plan spec has been validated.<br />`Complete` indicates that the latest version of the plan has completed on all selected nodes. If any Jobs for the Plan fail to complete, this condition will remain false, and the reason and message will reflect the source of the error. |  |  |
| `latestVersion` _string_ | The latest version, as resolved from .spec.version, or the channel server. |  |  |
| `latestHash` _string_ | The hash of the most recently applied plan .spec. |  |  |
| `applying` _string array_ | List of Node names that the Plan is currently being applied on. |  |  |


#### SecretSpec



SecretSpec describes a Secret to be mounted for prepare/upgrade containers.



_Appears in:_
- [PlanSpec](#planspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Secret name |  | Required: \{\} <br /> |
| `path` _string_ | Path to mount the Secret volume within the Pod. |  | Required: \{\} <br /> |
| `ignoreUpdates` _boolean_ | If set to true, the Secret contents will not be hashed, and changes to the Secret will not trigger new application of the Plan. |  |  |
| `defaultMode` _integer_ | Mode to mount the Secret volume with. |  | Optional: \{\} <br /> |


#### TimeWindowSpec



TimeWindowSpec describes a time window in which a Plan should be processed.



_Appears in:_
- [PlanSpec](#planspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `days` _[Day](#day) array_ | Days that this time window is valid for |  | Enum: [0 su sun sunday 1 mo mon monday 2 tu tue tuesday 3 we wed wednesday 4 th thu thursday 5 fr fri friday 6 sa sat saturday] <br />MinItems: 1 <br /> |
| `startTime` _string_ | Start of the time window. |  |  |
| `endTime` _string_ | End of the time window. |  |  |
| `timeZone` _string_ | Time zone for the time window; if not specified UTC will be used. |  |  |


#### VolumeSpec



HostPath volume to mount into the pod



_Appears in:_
- [ContainerSpec](#containerspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name of the Volume as it will appear within the Pod spec. |  | Required: \{\} <br /> |
| `source` _string_ | Path on the host to mount. |  | Required: \{\} <br /> |
| `destination` _string_ | Path to mount the Volume at within the Pod. |  | Required: \{\} <br /> |


