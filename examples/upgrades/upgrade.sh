#!/bin/bash

# load plans
kubectl apply -f bionic.yaml -f centos.yaml

# label all nodes for centos 7
kubectl get nodes -o=custom-columns=NAME:.metadata.name,NODE:.status.nodeInfo.osImage | awk '/CentOS Linux 7/ { print $1 }' | xargs -t -I % -n 1 kubectl label node % plan.upgrade.cattle.io/centos7=true 
# force os upgrade on master and workers
kubectl label nodes --selector='plan.upgrade.cattle.io/centos7-master' plan.upgrade.cattle.io/centos7-master-
kubectl label nodes --selector='plan.upgrade.cattle.io/centos7-worker' plan.upgrade.cattle.io/centos7-worker-

# label all nodes for ubuntu 18.04
kubectl get nodes -o=custom-columns=NAME:.metadata.name,NODE:.status.nodeInfo.osImage | awk '/Ubuntu 18.04/ { print $1 }' | xargs -t -I % -n 1 kubectl label node % plan.upgrade.cattle.io/bionic=true 
# force os upgrade on master and workers
kubectl label nodes --selector='plan.upgrade.cattle.io/bionic-master' plan.upgrade.cattle.io/bionic-master-
kubectl label nodes --selector='plan.upgrade.cattle.io/bionic-worker' plan.upgrade.cattle.io/bionic-worker-

