#!/usr/bin/env bash
# Copyright 2020 Adobe. All rights reserved.
# This file is licensed to you under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License. You may obtain a copy
# of the License at http://www.apache.org/licenses/LICENSE-2.0

# Unless required by applicable law or agreed to in writing, software distributed under
# the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR REPRESENTATIONS
# OF ANY KIND, either express or implied. See the License for the specific language
# governing permissions and limitations under the License.

# TODO: disable trace
set -euo pipefail
[ -o xtrace ] && env

function guess_runfiles() {
    if [ -d ${BASH_SOURCE[0]}.runfiles ]; then
        # Runfiles are adjacent to the current script.
        echo "$( cd ${BASH_SOURCE[0]}.runfiles && pwd )"
    else
        # The current script is within some other script's runfiles.
        mydir="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
        echo $mydir | sed -e 's|\(.*\.runfiles\)/.*|\1|'
    fi
}

RUNFILES=${TEST_SRCDIR:-$(guess_runfiles)}
TEST_UNDECLARED_OUTPUTS_DIR=${TEST_UNDECLARED_OUTPUTS_DIR:-.}

KUBECTL=%{kubectl}
KUBECONFIG=%{kubeconfig}
CLUSTER_FILE=%{cluster}

SET_NAMESPACE=%{set_namespace}
IT_MANIFEST_FILTER=%{it_manifest_filter}


NAMESPACE_NAME_FILE=${TEST_UNDECLARED_OUTPUTS_DIR}/namespace
KUBECONFIG_FILE=${TEST_UNDECLARED_OUTPUTS_DIR}/kubeconfig

# get cluster and username from provided configuration
if [ -n "${K8S_TEST_CLUSTER:-}" ]
then
    CLUSTER=${K8S_TEST_CLUSTER}
else
    CLUSTER=$(cat ${CLUSTER_FILE})
fi

USER=$(${KUBECTL} --kubeconfig=${KUBECONFIG} config view -o jsonpath='{.users[?(@.name == '"\"${CLUSTER}\")].name}")

echo "Cluster: ${CLUSTER}" >&2
echo "User: ${USER}" >&2

set +e

ns_cleanup() {
    echo "Performing namespace ${NAMESPACE} cleanup..."
    ${KUBECTL} --kubeconfig=${KUBECONFIG} --cluster=${CLUSTER} --user=${USER} delete namespace --wait=false ${NAMESPACE}
}

if [ -n "${K8S_TEST_NAMESPACE:-}" ]
then
    # use provided namespace
    NAMESPACE=${K8S_TEST_NAMESPACE}
elif [ -n "${K8S_MYNAMESPACE:-}" ]
then
    # do not create random namesspace
    NAMESPACE=$(whoami)
else
    # create random namespace
    COUNT="0"
    while true; do
        NAMESPACE=`whoami`-$(( (RANDOM) + 32767 ))
        ${KUBECTL} --kubeconfig=${KUBECONFIG} --cluster=${CLUSTER} --user=${USER} create namespace ${NAMESPACE} && break
        COUNT=$[$COUNT + 1]
        if [ $COUNT -ge 10 ]; then
            echo "Unable to create namespace in $COUNT attempts!" >&2
            exit 1
        fi
    done
    # delete namespace after the test is complete or failed
    trap ns_cleanup EXIT
fi
echo "Namespace: ${NAMESPACE}" >&2
set -e

# expose generated namespace name as rule output
mkdir -p $(dirname $NAMESPACE_NAME_FILE)
echo $NAMESPACE > $NAMESPACE_NAME_FILE

# create kubectl configuration copy with default context set to use newly created namespace
mkdir -p $(dirname $KUBECONFIG_FILE)
cat ${KUBECONFIG} > $KUBECONFIG_FILE
export KUBECONFIG=$KUBECONFIG_FILE
CONTEXT=$CLUSTER-$NAMESPACE
${KUBECTL} --cluster=$CLUSTER --user=$USER --namespace=$NAMESPACE config set-context $CONTEXT >&2
${KUBECTL} config use-context $CONTEXT >&2

# set runfiles for STMTS
export PYTHON_RUNFILES=${RUNFILES}

PIDS=()
function async() {
    # Launch the command asynchronously and track its process id.
    PYTHON_RUNFILES=${RUNFILES} "$@" &
    PIDS+=($!)
}

function waitpids() {
    # Wait for all of the subprocesses, failing the script if any of them failed.
    if [ "${#PIDS[@]}" != 0 ]; then
        for pid in ${PIDS[@]}; do
            wait ${pid}
        done
    fi
}

%{push_statements}
# create k8s objects
%{statements}

%{it_sidecar} -namespace=${NAMESPACE} %{sidecar_args} "$@"
