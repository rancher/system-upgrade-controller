ARG BCI=registry.suse.com/bci/bci-base:15.6
ARG GOLANG=registry.suse.com/bci/golang:1.25

FROM --platform=$BUILDPLATFORM ${GOLANG} AS build-base
ARG TARGETOS
ARG TARGETARCH
ARG VERSION=dev
ARG COMMIT=HEAD
ENV CGO_ENABLED=0 \
    GOMODCACHE=/root/.cache/go/modcache \
    GOCACHE=/root/.cache/go/cache
WORKDIR /src
COPY go.mod go.sum ./
COPY pkg/apis/go.mod pkg/apis/go.sum ./pkg/apis/
RUN --mount=type=cache,target=/root/.cache/go/modcache \
    go mod download
COPY . .

FROM ${GOLANG} AS e2e-ginkgo
ENV GOBIN=/bin
RUN go install github.com/onsi/ginkgo/v2/ginkgo@v2.23.4

FROM build-base AS controller-build
RUN --mount=type=cache,target=/root/.cache/go/modcache \
    --mount=type=cache,target=/root/.cache/go/cache \
    GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build \
      -ldflags "-X github.com/rancher/system-upgrade-controller/pkg/version.Version=${VERSION} -X github.com/rancher/system-upgrade-controller/pkg/version.GitCommit=${COMMIT} -extldflags -static -s" \
      -o /dist/system-upgrade-controller .

FROM scratch AS controller-binary
COPY --from=controller-build /dist/system-upgrade-controller /system-upgrade-controller

FROM build-base AS e2e-tests-build
RUN --mount=type=cache,target=/root/.cache/go/modcache \
    --mount=type=cache,target=/root/.cache/go/cache \
    GOOS=${TARGETOS} GOARCH=${TARGETARCH} go test -c \
      -ldflags "-X github.com/rancher/system-upgrade-controller/pkg/version.Version=${VERSION} -X github.com/rancher/system-upgrade-controller/pkg/version.GitCommit=${COMMIT} -extldflags -static -s" \
      -o /dist/system-upgrade-controller.test ./e2e/suite/...

FROM scratch AS e2e-tests-binary
COPY --from=e2e-tests-build /dist/system-upgrade-controller.test /system-upgrade-controller.test

FROM ${BCI} AS e2e-tests
ARG REPO=rancher
ARG TAG
ENV SYSTEM_UPGRADE_CONTROLLER_IMAGE=${REPO}/system-upgrade-controller:${TAG}
COPY --from=e2e-ginkgo /bin/ginkgo /bin/ginkgo
COPY --from=e2e-tests-build /dist/system-upgrade-controller.test /bin/system-upgrade-controller.test
COPY e2e/plugin/run.sh /run.sh
RUN set -x \
    && chmod +x /run.sh
RUN set -x \
    && zypper -n in tar gzip
ENTRYPOINT ["/run.sh"]

FROM --platform=$BUILDPLATFORM ${GOLANG} AS zoneinfo

FROM scratch AS controller
LABEL org.opencontainers.image.source="https://github.com/rancher/system-upgrade-controller"
COPY --from=controller-build /dist/system-upgrade-controller /bin/system-upgrade-controller
COPY --from=zoneinfo /usr/share/zoneinfo /usr/share/zoneinfo
USER 65534:65534
ENTRYPOINT ["/bin/system-upgrade-controller"]
