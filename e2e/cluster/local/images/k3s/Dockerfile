# Install the all-in-one binary so we can copy our run-time images into the image
# which helps avoid pulling them when running e2e tests.
ARG BCI="registry.suse.com/bci/bci-base:15.7"
FROM ${BCI} AS k3s
ARG ARCH
# renovate: datasource=github-release-attachments depName=k3s-io/k3s
ARG K3S_VERSION="v1.36.3+k3s1"
# renovate: datasource=github-release-attachments depName=k3s-io/k3s digestVersion=v1.36.3+k3s1
ARG K3S_SUM_amd64="2f98a9f8fe5782479ee2d54e70a1b10a7f6fd4cae8d38ed3098452dc6eed76b5"
# renovate: datasource=github-release-attachments depName=k3s-io/k3s digestVersion=v1.36.3+k3s1
ARG K3S_SUM_arm64="c9a209103f480f163b7c6a56f00862b4481927b284dc29a3716bb70d886691a8"
# renovate: datasource=github-release-attachments depName=k3s-io/k3s digestVersion=v1.36.3+k3s1
ARG K3S_SUM_arm="4f02209cdb2b5fc2ba45bd57a81a86cddbbe4fdf7d4e976976006ac00f69fee4"
RUN set -x \
   && zypper -n in \
   ca-certificates \
   curl \
   && case "${ARCH?required}" in \
   amd64) \
   K3S_ASSET="k3s"; \
   K3S_SUM="${K3S_SUM_amd64}"; \
   ;; \
   arm64) \
   K3S_ASSET="k3s-arm64"; \
   K3S_SUM="${K3S_SUM_arm64}"; \
   ;; \
   arm) \
   K3S_ASSET="k3s-armhf"; \
   K3S_SUM="${K3S_SUM_arm}"; \
   ;; \
   *) \
   echo "unsupported ARCH ${ARCH}" >&2; \
   exit 1; \
   ;; \
   esac \
   && curl -fsSL "https://github.com/k3s-io/k3s/releases/download/${K3S_VERSION}/${K3S_ASSET}" > /bin/k3s \
   && echo "${K3S_SUM}  /bin/k3s" | sha256sum -c - \
   && chmod +x /bin/k3s \
   && ln -s /bin/k3s /bin/kubectl \
   && ln -s /bin/k3s /bin/ctr \
   && k3s --version

COPY scratch/*-${ARCH}.tar /var/lib/rancher/k3s/agent/images/

VOLUME /var/lib/cni
VOLUME /var/lib/kubelet
VOLUME /var/lib/rancher/k3s
VOLUME /var/log

ENTRYPOINT ["k3s"]
CMD ["server"]
