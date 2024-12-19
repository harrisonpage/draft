#!/usr/bin/bash

set -euo pipefail
shopt -s inherit_errexit nullglob
cd "$(dirname "$0")"

export DRAFT_BUILD_VERSION=1.0.3
make test
make
