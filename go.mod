module github.com/rancher/system-upgrade-controller

go 1.26.0

toolchain go1.26.3

replace (
	github.com/distribution/reference => github.com/distribution/reference v0.5.0
	github.com/rancher/system-upgrade-controller/pkg/apis => ./pkg/apis
	k8s.io/apiserver => k8s.io/apiserver v0.36.0
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.36.0
	k8s.io/client-go => k8s.io/client-go v0.36.0
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.36.0
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.36.0
	k8s.io/code-generator => k8s.io/code-generator v0.36.0
	k8s.io/component-base => k8s.io/component-base v0.36.0
	k8s.io/component-helpers => k8s.io/component-helpers v0.36.0
	k8s.io/controller-manager => k8s.io/controller-manager v0.36.0
	k8s.io/cri-api => k8s.io/cri-api v0.36.0
	k8s.io/cri-client => k8s.io/cri-client v0.36.0
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.36.0
	k8s.io/dynamic-resource-allocation => k8s.io/dynamic-resource-allocation v0.36.0
	k8s.io/endpointslice => k8s.io/endpointslice v0.36.0
	k8s.io/externaljwt => k8s.io/externaljwt v0.36.0
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.36.0
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.36.0
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.36.0
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.36.0
	k8s.io/kubelet => k8s.io/kubelet v0.36.0
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.36.0
	k8s.io/metrics => k8s.io/metrics v0.36.0
	k8s.io/mount-utils => k8s.io/mount-utils v0.36.0
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.36.0
)

require (
	github.com/docker/distribution v2.8.3+incompatible
	github.com/kubereboot/kured v1.13.1
	github.com/onsi/ginkgo/v2 v2.28.1
	github.com/onsi/gomega v1.39.1
	github.com/rancher/lasso v0.2.9
	github.com/rancher/system-upgrade-controller/pkg/apis v0.0.0
	github.com/rancher/wrangler/v3 v3.7.0-rc.1
	github.com/sirupsen/logrus v1.9.4
	github.com/urfave/cli v1.22.15
	k8s.io/api v0.36.0
	k8s.io/apiextensions-apiserver v0.36.0
	k8s.io/apimachinery v0.36.0
	k8s.io/client-go v0.36.0
	k8s.io/kubectl v0.36.0
	k8s.io/kubernetes v1.36.0
	k8s.io/pod-security-admission v0.36.0
	k8s.io/utils v0.0.0-20260210185600-b8788abfbbc2
)

require (
	cel.dev/expr v0.25.1 // indirect
	github.com/Masterminds/semver/v3 v3.4.0 // indirect
	github.com/antlr4-go/antlr/v4 v4.13.0 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/blang/semver/v4 v4.0.0 // indirect
	github.com/cenkalti/backoff/v5 v5.0.3 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.6 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/distribution/reference v0.6.0 // indirect
	github.com/emicklei/go-restful/v3 v3.13.0 // indirect
	github.com/evanphx/json-patch v5.9.11+incompatible // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/fsnotify/fsnotify v1.9.0 // indirect
	github.com/fxamacker/cbor/v2 v2.9.0 // indirect
	github.com/ghodss/yaml v1.0.0 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-openapi/jsonpointer v0.21.0 // indirect
	github.com/go-openapi/jsonreference v0.21.0 // indirect
	github.com/go-openapi/swag v0.23.0 // indirect
	github.com/go-task/slim-sprig/v3 v3.0.0 // indirect
	github.com/google/cel-go v0.26.0 // indirect
	github.com/google/gnostic-models v0.7.0 // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/google/pprof v0.0.0-20260115054156-294ebfa9ad83 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/gorilla/websocket v1.5.4-0.20250319132907-e064f32e3674 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.27.7 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/moby/spdystream v0.5.1 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.3-0.20250322232337-35a7c28c31ee // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/prometheus/client_golang v1.23.2 // indirect
	github.com/prometheus/client_model v0.6.2 // indirect
	github.com/prometheus/common v0.67.5 // indirect
	github.com/prometheus/procfs v0.19.2 // indirect
	github.com/robfig/cron/v3 v3.0.1 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/spf13/cobra v1.10.2 // indirect
	github.com/spf13/pflag v1.0.9 // indirect
	github.com/stoewer/go-strcase v1.3.0 // indirect
	github.com/x448/float16 v0.8.4 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.65.0 // indirect
	go.opentelemetry.io/otel v1.43.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.40.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.40.0 // indirect
	go.opentelemetry.io/otel/metric v1.43.0 // indirect
	go.opentelemetry.io/otel/sdk v1.43.0 // indirect
	go.opentelemetry.io/otel/trace v1.43.0 // indirect
	go.opentelemetry.io/proto/otlp v1.9.0 // indirect
	go.yaml.in/yaml/v2 v2.4.3 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	golang.org/x/exp v0.0.0-20251219203646-944ab1f22d93 // indirect
	golang.org/x/mod v0.36.0 // indirect
	golang.org/x/net v0.53.0 // indirect
	golang.org/x/oauth2 v0.34.0 // indirect
	golang.org/x/sync v0.20.0 // indirect
	golang.org/x/sys v0.43.0 // indirect
	golang.org/x/term v0.42.0 // indirect
	golang.org/x/text v0.36.0 // indirect
	golang.org/x/time v0.14.0 // indirect
	golang.org/x/tools v0.44.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20260128011058-8636f8732409 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260128011058-8636f8732409 // indirect
	google.golang.org/grpc v1.79.3 // indirect
	google.golang.org/protobuf v1.36.12-0.20260120151049-f2248ac996af // indirect
	gopkg.in/evanphx/json-patch.v4 v4.13.0 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	k8s.io/apiserver v0.36.0 // indirect
	k8s.io/code-generator v0.36.0 // indirect
	k8s.io/component-base v0.36.0 // indirect
	k8s.io/component-helpers v0.36.0 // indirect
	k8s.io/controller-manager v0.36.0 // indirect
	k8s.io/gengo v0.0.0-20250130153323-76c5745d3511 // indirect
	k8s.io/gengo/v2 v2.0.0-20250922181213-ec3ebc5fd46b // indirect
	k8s.io/klog/v2 v2.140.0 // indirect
	k8s.io/kube-openapi v0.0.0-20260317180543-43fb72c5454a // indirect
	k8s.io/kubelet v0.36.0 // indirect
	k8s.io/streaming v0.36.0 // indirect
	sigs.k8s.io/apiserver-network-proxy/konnectivity-client v0.34.0 // indirect
	sigs.k8s.io/json v0.0.0-20250730193827-2d320260d730 // indirect
	sigs.k8s.io/randfill v1.0.0 // indirect
	sigs.k8s.io/structured-merge-diff/v6 v6.3.2 // indirect
	sigs.k8s.io/yaml v1.6.0 // indirect
)
