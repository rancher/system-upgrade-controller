ARG ALPINE=alpine:3.11
ARG GOLANG=golang:1.13-alpine

FROM ${GOLANG} AS e2e-ginkgo
RUN set -x \
 && apk add --no-cache \
    ca-certificates \
    git \
 && go get -d -v github.com/onsi/ginkgo/ginkgo \
 && go build -o /bin/ginkgo github.com/onsi/ginkgo/ginkgo

FROM ${ALPINE} AS e2e-tests
ARG ARCH
ARG REPO=rancher
ARG TAG
ENV SYSTEM_UPGRADE_CONTROLLER_IMAGE=${REPO}/system-upgrade-controller:${TAG}
RUN set -x \
 && apk add --no-cache \
    bash
COPY --from=e2e-ginkgo /bin/ginkgo /bin/ginkgo
COPY dist/artifacts/system-upgrade-controller.test-${ARCH} /bin/system-upgrade-controller.test
COPY e2e/plugin/run.sh /run.sh
RUN set -x \
 && chmod +x /run.sh
ENTRYPOINT ["/run.sh"]

FROM scratch AS controller
ARG ARCH
COPY dist/artifacts/system-upgrade-controller-${ARCH} /bin/system-upgrade-controller
ENTRYPOINT ["/bin/system-upgrade-controller"]
