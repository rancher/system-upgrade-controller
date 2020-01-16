#!/bin/sh
set -e

: ${REPO:=rancher/system-upgrade-controller}
: ${TAG:=$(git tag -l --contains HEAD | head -n 1)}

docker_image_build_and_push()
{
  docker buildx build \
    --build-arg ARCH=${1?required} \
    --build-arg GOLANG=${2?required} \
    --build-arg TAG=${3?required} \
    --platform linux/${1} \
    --tag ${4?required}:${3}-${1} \
    --file ${5?required}/Dockerfile.build \
  ${5?required}
  docker image push ${4?required}:${3}-${1}
}

docker_manifest_create_and_push()
{
  images=$(docker image ls $1-* --format '{{.Repository}}:{{.Tag}}')
  docker manifest create --amend ${1?required} $images
  for img in $images; do
    docker manifest annotate $1 $1-${img##*-} --os linux --arch ${img##*-}
  done
  docker manifest push $1
}

set -x

docker_image_build_and_push amd64  amd64/golang:1.13-alpine   ${TAG} ${REPO} $(dirname $0)/..
docker_image_build_and_push arm64  arm64v8/golang:1.13-alpine ${TAG} ${REPO} $(dirname $0)/..
docker_image_build_and_push arm    arm32v6/golang:1.13-alpine ${TAG} ${REPO} $(dirname $0)/..

docker_manifest_create_and_push ${REPO}:${TAG}
