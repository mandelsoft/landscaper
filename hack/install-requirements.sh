#!/bin/bash
#
# Copyright (c) 2018 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
#
# SPDX-License-Identifier: Apache-2.0

set -e

CURRENT_DIR=$(dirname $0)
PROJECT_ROOT="${CURRENT_DIR}"/..

curl -sfL "https://install.goreleaser.com/github.com/golangci/golangci-lint.sh" | sh -s -- -b $(go env GOPATH)/bin v1.32.2
mkdir -p ${PROJECT_ROOT}/tmp/test/registry
curl -sfL "https://storage.googleapis.com/gardener-public/test/oci-registry/registry-$(go env GOOS)-$(go env GOARCH)" --output ${PROJECT_ROOT}/tmp/test/registry/registry
chmod +x ${PROJECT_ROOT}/tmp/test/registry/registry

GO111MODULE=off go get golang.org/x/tools/cmd/goimports
