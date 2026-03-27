# Install the all-in-one binary so we can copy our run-time images into the image
# which helps avoid pulling them when running e2e tests.
ARG BCI="registry.suse.com/bci/bci-base:15.6"
FROM ${BCI} AS k3s
ARG ARCH
# renovate: datasource=github-release-attachments depName=k3s-io/k3s
ARG K3S_VERSION="v1.32.5+k3s1"
# renovate: datasource=github-release-attachments depName=k3s-io/k3s digestVersion=v1.32.5+k3s1
ARG K3S_SUM_amd64="9624ef0534b1be432b4ce8dd94dcd7241ed02451d970a4d38943baccf5c60dce"
# renovate: datasource=github-release-attachments depName=k3s-io/k3s digestVersion=v1.32.5+k3s1
ARG K3S_SUM_arm64="eb62d8ba346f7c6a0bb5c874ea92335ecad7cb3fa167508f94888aaeb43e7f22"
# renovate: datasource=github-release-attachments depName=k3s-io/k3s digestVersion=v1.32.5+k3s1
ARG K3S_SUM_arm="d9ea2edae8710246f7ccd0b0eb688e665937072a4ab8aa08a1b99d8b3ca89788"
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
