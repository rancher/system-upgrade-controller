# Install the all-in-one binary so we can copy our run-time images into the image
# which helps avoid pulling them when running e2e tests.
ARG BCI="registry.suse.com/bci/bci-base:15.6"
FROM ${BCI} AS k3s
ARG ARCH
# renovate: datasource=github-release-attachments depName=k3s-io/k3s
ARG K3S_VERSION="v1.36.2+k3s1"
# renovate: datasource=github-release-attachments depName=k3s-io/k3s digestVersion=v1.36.2+k3s1
ARG K3S_SUM_amd64="65a55ec56c24eab44383086166ec620a491952b7e23941a49ddca6e8a4c4b4de"
# renovate: datasource=github-release-attachments depName=k3s-io/k3s digestVersion=v1.36.2+k3s1
ARG K3S_SUM_arm64="1dc5fc17f15c28fa0a3f011cee28ad613f918c3d967a426e5a05d43ddb239817"
# renovate: datasource=github-release-attachments depName=k3s-io/k3s digestVersion=v1.36.2+k3s1
ARG K3S_SUM_arm="3e4cc1de8dc38eeed28f9c0f3de810964ae505a683e8f2342cbb6faafc520fe2"
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
