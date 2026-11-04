# Install the all-in-one binary so we can copy our run-time images into the image
# which helps avoid pulling them when running e2e tests.
ARG BCI="registry.suse.com/bci/bci-base:16.1"
FROM ${BCI} AS k3s
ARG ARCH
# renovate: datasource=github-release-attachments depName=k3s-io/k3s
ARG K3S_VERSION="v1.37.2+k3s1"
# renovate: datasource=github-release-attachments depName=k3s-io/k3s digestVersion=v1.37.1+k3s1
ARG K3S_SUM_amd64="265b3ee6a3198ef14033c63ef46cbb32370d83795199fb6d1b01f21575eddbda"
# renovate: datasource=github-release-attachments depName=k3s-io/k3s digestVersion=v1.37.1+k3s1
ARG K3S_SUM_arm64="a779ccf1e5e2e33a62f5a29c2f1c0bbf99431e100579d4c10f0aa0742a6aa84b"
# renovate: datasource=github-release-attachments depName=k3s-io/k3s digestVersion=v1.37.1+k3s1
ARG K3S_SUM_arm="183a5e5cadee7ccfc192bb0e2633bbf1f4a1b16025e6a8e6620fc537cc2d3d9c"
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
