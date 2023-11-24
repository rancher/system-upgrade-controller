ARG BUILDER_GOLANG_VERSION

FROM --platform=$TARGETPLATFORM gcr.io/spectro-images-public/golang:${BUILDER_GOLANG_VERSION}-alpine as builder
ARG TARGETOS
ARG TARGETARCH
ARG CRYPTO_LIB
ENV GOEXPERIMENT=${CRYPTO_LIB:+boringcrypto}

WORKDIR /workspace

COPY . .

RUN apk -U add coreutils gcc musl-dev

RUN mkdir -p bin

RUN if [ ${CRYPTO_LIB} ]; \
    then \
      go-build-fips.sh -a -o bin/system-upgrade-controller ;\
    else \
      go-build-static.sh -a -o bin/system-upgrade-controller ;\
    fi

FROM --platform=$TARGETPLATFORM scratch AS controller
WORKDIR /bin
COPY --from=builder /workspace/bin/system-upgrade-controller .
ENTRYPOINT ["/bin/system-upgrade-controller"]
