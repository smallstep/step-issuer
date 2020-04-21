PKG?=github.com/smallstep/step-issuer
BINNAME?=manager
PREFIX?=

# Set V to 1 for verbose output from the Makefile
Q=$(if $V,,@)
# Image URL to use all building/pushing image targets
IMG ?= smallstep/step-issuer:latest
# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd:trivialVersions=true"

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

all: build test lint

.PHONY: all

#########################################
# Bootstrapping
#########################################

bootstra%:
	# Using a released version of golangci-lint to take into account custom replacements in their go.mod
	$Q curl -sSfL https://raw.githubusercontent.com/smallstep/cli/master/make/golangci-install.sh | sh -s -- -b $(shell go env GOPATH)/bin v1.23.8

.PHONY: bootstra%

#################################################
# Determine the type of `push` and `version`
#################################################

# If TRAVIS_TAG is set then we know this ref has been tagged.
ifdef TRAVIS_TAG
VERSION := $(TRAVIS_TAG)
NOT_RC  := $(shell echo $(VERSION) | grep -v -e -rc)
	ifeq ($(NOT_RC),)
PUSHTYPE := release-candidate
	else
PUSHTYPE := release
	endif
else
VERSION ?= $(shell [ -d .git ] && git describe --tags --always --dirty="-dev")
VERSION := $(or $(VERSION),v0.0.0)
PUSHTYPE := master
endif

VERSION := $(shell echo $(VERSION) | sed 's/^v//')

ifdef V
$(info    TRAVIS_TAG is $(TRAVIS_TAG))
$(info    VERSION is $(VERSION))
$(info    PUSHTYPE is $(PUSHTYPE))
endif

#########################################
# Test
#########################################

test: generate fmt vet manifests
	$Q go test ./api/... ./controllers/... -coverprofile cover.out

.PHONY: test

#########################################
# Build
#########################################

DATE    := $(shell date -u '+%Y-%m-%d %H:%M UTC')
LDFLAGS := -ldflags='-w -X "main.Version=$(VERSION)" -X "main.BuildTime=$(DATE)"'
GOFLAGS := CGO_ENABLED=0

build: $(PREFIX)bin/$(BINNAME)
	@echo "Build Complete!"

download:
	$Q go mod download

$(PREFIX)bin/$(BINNAME): download generate $(call rwildcard,*.go)
	$Q mkdir -p $(@D)
	$Q $(GOOS_OVERRIDE) $(GOFLAGS) go build -v -o $(PREFIX)bin/$(BINNAME) $(LDFLAGS) $(PKG)

#########################################
# Generate
#########################################

# Generate code
generate: controller-gen
	$Q $(CONTROLLER_GEN) object:headerFile=./hack/boilerplate.go.txt paths=./api/...

# find or download controller-gen
# download controller-gen if necessary
controller-gen:
ifeq (, $(shell which controller-gen))
	$Q go get sigs.k8s.io/controller-tools/cmd/controller-gen@v0.2.5
CONTROLLER_GEN=$(GOBIN)/controller-gen
else
CONTROLLER_GEN=$(shell which controller-gen)
endif

#########################################
# Install
#########################################

# Generate manifests e.g. CRD, RBAC etc.
manifests: controller-gen
	$Q $(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases

# Install CRDs into a cluster
install: manifests
	$Q kubectl apply -f config/crd/bases

# Deploy controller in the configured Kubernetes cluster in ~/.kube/config
deploy: manifests
	$Q kubectl apply -f config/crd/bases
	$Q kustomize build config/default | kubectl apply -f -

#########################################
# Format and Linting
#########################################

# Run go fmt against code
fmt:
	$Q go fmt ./...

# Run go vet against code
vet:
	$Q go vet ./...

lint:
	$Q LOG_LEVEL=error golangci-lint run

.PHONY: fmt vet lint

#########################################
# Dev
#########################################

# Run against the configured Kubernetes cluster in ~/.kube/config
run: generate
	$Q go run ./main.go

.PHONY: run

#########################################
# Clean
#########################################

clean:
ifneq ($(BINNAME),"")
	$Q rm -f bin/$(BINNAME)
endif

.PHONY: clean

#################################################
# Docker
#################################################

DOCKER_OUTPUT=$(OUTPUT_ROOT)docker/
DOCKER_MAKE=V=$V GOOS_OVERRIDE='GOOS=linux GOARCH=amd64' PREFIX=$(1) make $(1)bin/$(2)
DOCKER_BUILD=$Q docker build -t $(IMG) -f $(2) --build-arg BINPATH=$(DOCKER_OUTPUT)bin/$(1) .

docker: docker-make Dockerfile
	$(call DOCKER_BUILD,manager,Dockerfile)

docker-make:
	$Q mkdir -p $(DOCKER_OUTPUT)
	$(call DOCKER_MAKE,$(DOCKER_OUTPUT),manager)

.PHONY: docker docker-make

# Make sure to run a local registry
# docker run -d -p 5000:5000 --restart=always --name registry registry:2
docker-dev: docker
	$Q docker tag ${IMG} localhost:5000/${IMG}
	$Q docker push localhost:5000/${IMG}

.PHONY: docker-dev

#################################################
# Releasing Docker Images
#################################################

DOCKER_TAG=docker tag smallstep/$(1):latest smallstep/$(1):$(2)
DOCKER_PUSH=docker push smallstep/$(1):$(2)

docker-tag:
	$(call DOCKER_TAG,step-issuer,$(VERSION))

docker-push-tag: docker-tag
	$(call DOCKER_PUSH,step-issuer,$(VERSION))

docker-push-tag-latest:
	$(call DOCKER_PUSH,step-issuer,latest)

# Rely on DOCKER_USERNAME and DOCKER_PASSWORD being set inside the CI or
# equivalent environment
docker-login:
	$Q docker login -u="$(DOCKER_USERNAME)" -p="$(DOCKER_PASSWORD)"

.PHONY: docker-login docker-tag docker-push-tag docker-push-tag-latest

#################################################
# Targets for pushing the docker images
#################################################

# For all builds we build the docker container
docker-master: docker

# For all builds with a release candidate tag
docker-release-candidate: docker-master docker-login docker-push-tag

# For all builds with a release tag
docker-release: docker-release-candidate docker-push-tag-latest

.PHONY: docker-master docker-release-candidate docker-release

#################################################
# Targets for creating step artifacts
#################################################

# This command is called by travis directly *after* a successful build
artifacts: docker-$(PUSHTYPE)

.PHONY: artifacts
