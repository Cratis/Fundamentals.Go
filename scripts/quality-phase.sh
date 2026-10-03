#!/usr/bin/env bash
# Copyright (c) Cratis. All rights reserved.
# Licensed under the MIT license. See LICENSE file in the project root for full license information.
set -euo pipefail

# One native phase per gate; pi users get bounded, cooperative execution.
# Other contributors need only Go. The caller must bound the overall gate run.
case "${1:-}" in
    build|vet) args=("$1" ./...) ;;
    test) args=(test -count=1 -timeout=2m ./...) ;;
    *) printf 'usage: quality-phase.sh build|vet|test\n' >&2; exit 2 ;;
esac
export GOWORK=off GOTOOLCHAIN=local
if command -v pi-phase >/dev/null 2>&1; then
    kind=build
    [ "$1" != test ] || kind="test"
    exec pi-phase run --kind "$kind" --label "Go $1" \
        --timeout 120 --queue-timeout 30 -- go "${args[@]}"
fi
exec go "${args[@]}"
