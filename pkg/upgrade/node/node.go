package node

import corev1 "k8s.io/api/core/v1"

func Hostname(node *corev1.Node) string {
	if node.Labels != nil {
		if hostname, ok := node.Labels[corev1.LabelHostname]; ok {
			return hostname
		}
	}
	return node.Name
}
