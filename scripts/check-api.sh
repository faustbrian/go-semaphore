#!/bin/sh
set -eu

root="$(git rev-parse --show-toplevel)"
GOWORK=off "$root/.golib/scripts/check-api-baseline.sh" .
