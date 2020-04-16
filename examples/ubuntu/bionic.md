# examples/ubuntu:bionic

An example **Plan** for orchestrating upgrades on Ubuntu Bionic/18.04 nodes (that happen to be running K3s).
It should work correctly on any Ubuntu Bionic node running kubernetes with the label `plan.upgrade.cattle.io/bionic`
that exists and isn't equal to `disabled`.

For your convenience, there is a [Docker Compose spec](docker-compose.yaml) that stands up a K3s cluster with nodes
built from a [Dockerfile](bionic/k3s/Dockerfile) (ubuntu:bionic + k3s). To run it in it's full glory:
`docker-compose up -d --scale master=2 --scale worker=5`. *(The "masters" are subordinate to the initial "leader" which
is why there are only two.)*

For demonstration purposes the nodes are build with curl and openssl (and their libs) pinned at obsolete versions. The
supplied **Plan** will bump both curl and openssl to their newest versions at the time of this writing. You and/or 
your organization will probably build your own upgrade containers but I wanted to show the power and simplicity of the
system-upgrade-controller (a.k.a. SUC) combined with official container images that match the OS of the nodes you are
upgrading.

The SUC will normally trigger jobs for a **Plan** when there is a new `.status.latestVersion` which can be specified
directly in `.spec.version` or resolved from the URL in `.spec.channel` (i.e. a GitHub latest release link that 
redirects to the most recent release tag). But what about more granular upgrade operations (i.e. individual packages)
for an OS that is pinned at the LTS version or codename (in this case `bionic`)? The SUC allows for this by also
triggering jobs if any underlying secrets change that the **Plan** has referenced. We will be leveraging this
functionality in this example.

Inspecting `bionic.yaml` we see a curious secret:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: bionic
  namespace: system-upgrade
type: Opaque
stringData:
  curl: 7.58.0-2ubuntu3.8
  openssl: 1.1.1-1ubuntu2.1~18.04.5
  upgrade.sh: |
    #!/bin/sh
    set -e
    secrets=$(dirname $0)
    apt-get --assume-yes update
    apt-get --assume-yes install \
      curl=$(cat $secrets/curl) \
      libcurl4=$(cat $secrets/curl) \
      libssl1.1=$(cat $secrets/openssl) \
      openssl=$(cat $secrets/openssl)
```

We have a version for both the `curl` and `openssl` packages and a script that applies the installation of these two
packages (also libcurl4 and libssl1.1). The **Plan** to execute the script looks like:

```yaml
apiVersion: upgrade.cattle.io/v1
kind: Plan
metadata:
  name: bionic
  namespace: system-upgrade
spec:
  concurrency: 2
  nodeSelector:
    matchExpressions:
      - {key: plan.upgrade.cattle.io/bionic, operator: Exists}
  serviceAccountName: system-upgrade
  secrets:
    - name: bionic
      path: /host/run/system-upgrade/secrets/bionic
  drain:
    force: true
  version: bionic
  upgrade:
    image: ubuntu
    command: ["chroot", "/host"]
    args: ["sh", "/run/system-upgrade/secrets/bionic/upgrade.sh"]

```

The SUC normally mounts secrets under `/run/system-upgrade/secrets` but as you can see we have overridden this with the
same value but prefixed by `/host` and this is because the SUC mounts the node's root filesystem at `/host` combined
with the fact that we want to leverage the node's built in package management system, APT/dpkg. We could have copied
the secrets in the script prior to invoking `chroot /host` but this has the advantage of being simpler AND avoiding the
need to cleanup our potentially sensitive secrets afterwards.

With a `.spec.concurrency` of `2` you should see no more than that number of nodes upgrading at one time. Try
`docker exec -it ubuntu-leader kubectl get all -A` to see.

## k3s

As a bonus, because we are running K3s, I have included two plans in [`../k3s-upgrade.yaml`](../k3s-upgrade.yaml) that are made available to
the cluster and activated by running `docker exec -it kubectl label node --all k3s-upgrade=enabled`. These two plans
coordinate to upgrade all of the masters with a concurrency of 1 and then all of the workers with a concurrency of 2.
See https://github.com/rancher/k3s-upgrade, written by https://github.com/galal-hussein.