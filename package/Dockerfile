ARG SLES=registry.suse.com/suse/sle15:15.3
ARG GOLANG=registry.suse.com/bci/golang:1.17-11.33

FROM ${GOLANG} AS e2e-ginkgo
ENV GOBIN=/bin
RUN go install github.com/onsi/ginkgo/ginkgo@v1.16.4

FROM ${SLES} AS e2e-tests
ARG ARCH
ARG REPO=rancher
ARG TAG
ENV SYSTEM_UPGRADE_CONTROLLER_IMAGE=${REPO}/system-upgrade-controller:${TAG}
COPY --from=e2e-ginkgo /bin/ginkgo /bin/ginkgo
COPY dist/artifacts/system-upgrade-controller.test-${ARCH} /bin/system-upgrade-controller.test
COPY e2e/plugin/run.sh /run.sh
RUN set -x \
 && chmod +x /run.sh
RUN set -x \
 && zypper -n in tar gzip
ENTRYPOINT ["/run.sh"]

FROM scratch AS controller
ARG ARCH
COPY dist/artifacts/system-upgrade-controller-${ARCH} /bin/system-upgrade-controller
ENTRYPOINT ["/bin/system-upgrade-controller"]
