#!/bin/bash

set -x

# This is the entrypoint for the image and meant to wrap the
# logic of gathering/reporting results to the Sonobuoy worker.

results_dir="${RESULTS_DIR:-/tmp/results}"

mkdir -p ${results_dir}

# saveResults prepares the results for handoff to the Sonobuoy worker.
# See: https://github.com/vmware-tanzu/sonobuoy/blob/main/site/content/docs/main/plugins.md
saveResults() {
    cd ${results_dir}

    # Sonobuoy worker expects a tar file.
    tar czf results.tar.gz *

    # Signal to the worker that we are done and where to find the results.
    printf ${results_dir}/results.tar.gz > ${results_dir}/done
}

# Ensure that we tell the Sonobuoy worker we are done regardless of results.
trap saveResults EXIT

ginkgo run --output-dir=${results_dir} --junit-report=junit.xml -p "${SUITE_EXE:-/bin/system-upgrade-controller.test}" -- --provider=local &>${results_dir}/out
