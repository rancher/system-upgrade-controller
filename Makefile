SCRIPT_TARGETS := build ci e2e-sonobuoy e2e-verify package package-controller package-e2e-tests release test validate
UNAME_ARCH := $(shell uname -m)
ifeq ($(UNAME_ARCH),x86_64)
	ARCH ?= amd64
else ifeq ($(UNAME_ARCH),aarch64)
	ARCH ?= arm64
else
	ARCH ?= $(UNAME_ARCH)
endif
export ARCH

BUILDX_BUILDER := system-upgrade-controller
IMAGE_BUILDER ?= docker buildx
DEFAULT_PLATFORMS := linux/amd64,linux/arm64,linux/arm
BUILDX_ARGS ?= --provenance=false --sbom=false
PLATFORM ?= linux/$(ARCH)

$(SCRIPT_TARGETS):
	./scripts/$@

.PHONY: buildx
buildx:
	@bash -c 'set -eo pipefail; \
	source ./scripts/version; \
	BUILDX_PLATFORM="$${BUILDX_PLATFORM:-$${TARGET_PLATFORMS:-$(PLATFORM)}}"; \
	$(IMAGE_BUILDER) build \
		$${IID_FILE_FLAG:-} \
		--file package/Dockerfile \
		$${BUILDX_BUILDER_FLAG:+--builder $${BUILDX_BUILDER_FLAG}} \
		--build-arg REPO="$${REPO}" \
		--build-arg TAG="$${TAG}" \
		--build-arg VERSION="$${VERSION}" \
		--build-arg COMMIT="$${COMMIT}" \
		$${BUILDX_TARGET:+--target $${BUILDX_TARGET}} \
		$${BUILDX_PLATFORM:+--platform $${BUILDX_PLATFORM}} \
		$${BUILDX_OUTPUT:+--output $${BUILDX_OUTPUT}} \
		$${BUILDX_TAG_ARGS:-} \
		$${BUILDX_EXTRA_ARGS:-} \
		$${BUILDX_PUSH:+--push} \
		.'

.PHONY: build-controller
build-controller:
	@echo "Building github.com/rancher/system-upgrade-controller ..."
	@mkdir -p bin
	@$(MAKE) --no-print-directory buildx \
		BUILDX_TARGET=controller-binary \
		BUILDX_OUTPUT="type=local,dest=./bin"
	@chmod +x bin/system-upgrade-controller

.PHONY: build-e2e-tests
build-e2e-tests:
	@echo "Building github.com/rancher/system-upgrade-controller/e2e ..."
	@mkdir -p bin
	@$(MAKE) --no-print-directory buildx \
		BUILDX_TARGET=e2e-tests-binary \
		BUILDX_OUTPUT="type=local,dest=./bin"
	@chmod +x bin/system-upgrade-controller.test

e2e: e2e-sonobuoy
	$(MAKE) e2e-verify

clean:
	rm -rvf ./bin ./dist

.PHONY: buildx-builder
buildx-builder:
	@docker buildx inspect $(BUILDX_BUILDER) >/dev/null 2>&1 || docker buildx create --name=$(BUILDX_BUILDER) --platform=$(DEFAULT_PLATFORMS)

push-image: buildx-builder
	@echo "--- Building and Pushing Image ---"
	@bash -c 'set -eo pipefail; \
	source ./scripts/version; \
	IMAGE="$${REPO}/system-upgrade-controller:$${TAG}"; \
	$(MAKE) --no-print-directory buildx \
		BUILDX_BUILDER_FLAG="$(BUILDX_BUILDER)" \
		BUILDX_TARGET=controller \
		BUILDX_TAG_ARGS="--tag $${IMAGE}" \
		BUILDX_EXTRA_ARGS="$(BUILDX_ARGS)" \
		BUILDX_PUSH=1 \
		IID_FILE_FLAG="$${IID_FILE_FLAG:-}"; \
	echo "Pushed $${IMAGE}"'

.PHONY: push-prime-image
push-prime-image:
	@$(MAKE) --no-print-directory push-image \
		BUILDX_ARGS="--sbom=true --attest type=provenance,mode=max"

.DEFAULT_GOAL := ci

.PHONY: $(SCRIPT_TARGETS) e2e clean push-image
