# Copyright (c) Huawei Technologies Co., Ltd. 2018-2019. All rights reserved.
# isulad-tools is licensed under the Mulan PSL v1.
# You can use this software according to the terms and conditions of the Mulan PSL v1.
# You may obtain a copy of Mulan PSL v1 at:
#    http://license.coscl.org.cn/MulanPSL
# THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR
# IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR
# PURPOSE.
# See the Mulan PSL v1 for more details.
# Description: makefile for isulad-tools
# Author: zhangwei
# Create: 2018-01-18

COMMIT=$(shell git rev-parse HEAD 2> /dev/null || true)
SOURCES := $(shell find . 2>&1 | grep -E '.*\.(c|h|go)$$')
DEPS_LINK := $(CURDIR)/vendor/
TAGS="cgo static_build"
VERSION := $(shell cat ./VERSION)

BEP_DIR=/tmp/isulad-tools-build-bep
BEP_FLAGS=-tmpdir=/tmp/isulad-tools-build-bep

GO_LDFLAGS="-s -w -buildid=IdByiSula -extldflags=-zrelro -extldflags=-znow $(BEP_FLAGS) -X main.gitCommit=${COMMIT} -X main.version=${VERSION}"
ENV = GOPATH=${GOPATH} CGO_ENABLED=1

## PLEASE be noticed that the vendor dir can only work with golang > 1.6 !!

all: dep isulad-tools isulad-hooks
dep:
	mkdir -p $(BEP_DIR)

init:
	sh -x apply-patch

isulad-tools: $(SOURCES) | $(DEPS_LINK)
	@echo "Making isulad-tools..."
	${ENV} go build -tags ${TAGS} -ldflags ${GO_LDFLAGS} -o build/isulad-tools .
	@echo "Done!"

isulad-hooks: $(SOURCES) | $(DEPS_LINK)
	@echo "Making isulad-hooks..."
	${ENV} go build -tags ${TAGS} -ldflags ${GO_LDFLAGS} -o build/isulad-hooks ./hooks/isulad-hooks
	@echo "Done!"

localtest:
	go test -tags ${TAGS} -ldflags ${GO_LDFLAGS} -v ./...

clean:
	rm -rf build

install:
	cd hack && ./install.sh

.PHONY: test
