#!/bin/bash

# This is the entrypoint for the image and meant to wrap the
# logic of gathering/reporting results to the Sonobuoy worker.

results_dir="${RESULTS_DIR:-/tmp/results}"

# save_results prepares the results for handoff to the Sonobuoy worker.
# See: https://github.com/vmware-tanzu/sonobuoy/blob/master/docs/plugins.md
save_results() {
    cd ${results_dir}

    # Sonobuoy worker expects a tar file.
	tar czf results.tar.gz *

	# Signal to the worker that we are done and where to find the results.
	printf ${results_dir}/results.tar.gz > ${results_dir}/done
}

# Ensure that we tell the Sonobuoy worker we are done regardless of results.
trap save_results EXIT

ginkgo -p "${SUITE_EXE:-/bin/system-upgrade-controller.test}"
