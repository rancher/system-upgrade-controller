# Install the all-in-one binary so we can copy our run-time images into the image
# which helps avoid pulling them when running e2e tests.
ARG BCI="registry.suse.com/bci/bci-base:16.1"
FROM ${BCI} AS k3s
ARG ARCH
# renovate: datasource=github-release-attachments depName=k3s-io/k3s
ARG K3S_VERSION="v1.37.0+k3s1"
# renovate: datasource=github-release-attachments depName=k3s-io/k3s digestVersion=v1.37.0+k3s1
ARG K3S_SUM_amd64="39eed8f53f277497dfc2542f66eab0ed68a94dfc598946dbebfb50366916c7a2"
# renovate: datasource=github-release-attachments depName=k3s-io/k3s digestVersion=v1.37.0+k3s1
ARG K3S_SUM_arm64="9bc2c128a597bf7c10ee45df844f2838251ba53806f7673d7fea24a6cb1b6a99"
# renovate: datasource=github-release-attachments depName=k3s-io/k3s digestVersion=v1.37.0+k3s1
ARG K3S_SUM_arm="8ae038bbec210618d6af4977ab8b7088f093ce5155d9c26b3708397216cdf74b"
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
