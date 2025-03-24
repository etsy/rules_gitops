#!/usr/bin/env bash
set -x
set -e

CURR_GIT_COMMIT=$(git rev-parse HEAD)
GIT_COMMIT=${BUILDKITE_COMMIT:-$CURR_GIT_COMMIT}

%{prer} --git_commit $GIT_COMMIT %{params}
