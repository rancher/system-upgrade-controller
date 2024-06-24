TARGETS := $(shell ls scripts)

GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)
GOPATH ?= $(shell go env GOPATH)
TARGETARCH ?= amd64

FIPS_ENABLE ?= ""
BUILDER_GOLANG_VERSION ?= 1.22
BUILD_ARGS = --build-arg CRYPTO_LIB=${FIPS_ENABLE} --build-arg BUILDER_GOLANG_VERSION=${BUILDER_GOLANG_VERSION}

IMG_PATH ?= gcr.io/spectro-dev-public/release
ifeq ($(FIPS_ENABLE),yes)
	IMG_PATH = gcr.io/spectro-dev-public/release-fips
endif
IMG_TAG ?= v0.11.5_spectro
IMG_SERVICE_URL ?= ${IMG_PATH}/
SUC_IMG ?= ${IMG_SERVICE_URL}system-upgrade-controller:${IMG_TAG}

.dapper:
	@echo Downloading dapper
	@curl -sL https://releases.rancher.com/dapper/latest/dapper-`uname -s`-`uname -m` > .dapper.tmp
	@@chmod +x .dapper.tmp
	@./.dapper.tmp -v
	@mv .dapper.tmp .dapper

$(TARGETS): .dapper
	./.dapper $@

e2e: e2e-sonobuoy
	$(MAKE) e2e-verify

clean:
	rm -rvf ./bin ./dist

docker:
	docker buildx build --platform linux/amd64,linux/arm64 --push . -t ${SUC_IMG} ${BUILD_ARGS} -f Dockerfile

.DEFAULT_GOAL := ci

.PHONY: $(TARGETS) e2e clean
