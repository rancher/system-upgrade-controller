FROM --platform=$TARGETPLATFORM golang:1.19.10-alpine as builder
ARG TARGETOS
ARG TARGETARCH
ARG CRYPTO_LIB
ENV GOEXPERIMENT=${CRYPTO_LIB:+boringcrypto}

WORKDIR /workspace

COPY . .

RUN apk -U add coreutils bash expect git gcc musl-dev
RUN apk --update add \
    util-linux-dev
RUN mkdir -p bin

RUN if [ ${CRYPTO_LIB} ]; \
    then \
      CGO_ENABLED=1 GOOS=${TARGETOS} GOARCH=${TARGETARCH} GO111MODULE=on go build -ldflags "-linkmode=external -extldflags=-static" -a -o bin/system-upgrade-controller ;\
    else \
      CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} GO111MODULE=on go build -a -o bin/system-upgrade-controller ;\
    fi

FROM --platform=$TARGETPLATFORM scratch AS controller
WORKDIR /bin
COPY --from=builder /workspace/bin/system-upgrade-controller .
ENTRYPOINT ["/bin/system-upgrade-controller"]
