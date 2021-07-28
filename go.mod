module github.com/rancher/system-upgrade-controller

go 1.13

require (
	github.com/docker/distribution v2.7.1+incompatible
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/onsi/ginkgo v1.11.0
	github.com/onsi/gomega v1.8.1
	github.com/rancher/lasso v0.0.0-20210408231703-9ddd9378d08d
	github.com/rancher/system-upgrade-controller/pkg/apis v0.0.0
	github.com/rancher/wrangler v0.8.1-0.20210424045532-c389a3e0ae51
	github.com/sirupsen/logrus v1.4.2
	github.com/urfave/cli v1.22.2
	k8s.io/api v0.18.8
	k8s.io/apimachinery v0.18.8
	k8s.io/client-go v11.0.1-0.20190409021438-1a26190bd76a+incompatible
	k8s.io/kubectl v0.0.0
	k8s.io/kubernetes v1.17.2
)

replace (
	github.com/rancher/system-upgrade-controller/pkg/apis => ./pkg/apis
	k8s.io/api => github.com/rancher/kubernetes/staging/src/k8s.io/api v1.18.0-k3s.1
	k8s.io/apiextensions-apiserver => github.com/rancher/kubernetes/staging/src/k8s.io/apiextensions-apiserver v1.18.0-k3s.1
	k8s.io/apimachinery => github.com/rancher/kubernetes/staging/src/k8s.io/apimachinery v1.18.0-k3s.1
	k8s.io/apiserver => github.com/rancher/kubernetes/staging/src/k8s.io/apiserver v1.18.0-k3s.1
	k8s.io/cli-runtime => github.com/rancher/kubernetes/staging/src/k8s.io/cli-runtime v1.18.0-k3s.1
	k8s.io/client-go => github.com/rancher/kubernetes/staging/src/k8s.io/client-go v1.18.0-k3s.1
	k8s.io/cloud-provider => github.com/rancher/kubernetes/staging/src/k8s.io/cloud-provider v1.18.0-k3s.1
	k8s.io/cluster-bootstrap => github.com/rancher/kubernetes/staging/src/k8s.io/cluster-bootstrap v1.18.0-k3s.1
	k8s.io/code-generator => github.com/rancher/kubernetes/staging/src/k8s.io/code-generator v1.18.0-k3s.1
	k8s.io/component-base => github.com/rancher/kubernetes/staging/src/k8s.io/component-base v1.18.0-k3s.1
	k8s.io/cri-api => github.com/rancher/kubernetes/staging/src/k8s.io/cri-api v1.18.0-k3s.1
	k8s.io/csi-translation-lib => github.com/rancher/kubernetes/staging/src/k8s.io/csi-translation-lib v1.18.0-k3s.1
	k8s.io/kube-aggregator => github.com/rancher/kubernetes/staging/src/k8s.io/kube-aggregator v1.18.0-k3s.1
	k8s.io/kube-controller-manager => github.com/rancher/kubernetes/staging/src/k8s.io/kube-controller-manager v1.18.0-k3s.1
	k8s.io/kube-proxy => github.com/rancher/kubernetes/staging/src/k8s.io/kube-proxy v1.18.0-k3s.1
	k8s.io/kube-scheduler => github.com/rancher/kubernetes/staging/src/k8s.io/kube-scheduler v1.18.0-k3s.1
	k8s.io/kubectl => github.com/rancher/kubernetes/staging/src/k8s.io/kubectl v1.18.0-k3s.1
	k8s.io/kubelet => github.com/rancher/kubernetes/staging/src/k8s.io/kubelet v1.18.0-k3s.1
	k8s.io/kubernetes => github.com/rancher/kubernetes v1.18.0-k3s.1
	k8s.io/legacy-cloud-providers => github.com/rancher/kubernetes/staging/src/k8s.io/legacy-cloud-providers v1.18.0-k3s.1
	k8s.io/metrics => github.com/rancher/kubernetes/staging/src/k8s.io/metrics v1.18.0-k3s.1
	k8s.io/sample-apiserver => github.com/rancher/kubernetes/staging/src/k8s.io/sample-apiserver v1.18.0-k3s.1
	k8s.io/sample-cli-plugin => github.com/rancher/kubernetes/staging/src/k8s.io/sample-cli-plugin v1.18.0-k3s.1
	k8s.io/sample-controller => github.com/rancher/kubernetes/staging/src/k8s.io/sample-controller v1.18.0-k3s.1
)
