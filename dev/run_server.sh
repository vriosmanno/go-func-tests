#!/bin/bash
# Runs server from a local directory, not the installed directory.
# Used to test a developmental version before installing it into the
# official location.
#

set -e	  # Exit immediately if a command exits with a non-zero status.

repodir="$( cd "$( dirname "${BASH_SOURCE[0]}" )/.." && pwd )"

srvc_name=testing

# Run
source ${repodir}/conf/${srvc_name}.env
imagePath="${DATA_DIR}/test_files/Star5.jpg"
go run ${repodir}/server/server/TestFunctions.go -i ${imagePath}