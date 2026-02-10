#!/usr/bin/env bash
set -x
set -e

CURR_GIT_COMMIT=$(git rev-parse HEAD)
GIT_COMMIT=${BUILDKITE_COMMIT:-$CURR_GIT_COMMIT}
BRANCH_NAME="%{branch_name_prefix}-${GIT_COMMIT:0:7}"

%{prer} --git_commit $GIT_COMMIT --branch_name $BRANCH_NAME %{params} "${@}"
