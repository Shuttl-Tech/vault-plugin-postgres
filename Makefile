# Metadata about this makefile and position
MKFILE_PATH := $(lastword $(MAKEFILE_LIST))
CURRENT_DIR := $(patsubst %/,%,$(dir $(realpath $(MKFILE_PATH))))

# Ensure GOPATH
GOPATH ?= $(HOME)/go

# List all our actual files, excluding vendor
GOFILES ?= $(shell go list $(TEST) | grep -v /vendor/)

# Tags specific for building
GOTAGS ?=

# Number of procs to use
GOMAXPROCS ?= 4

# Get the project metadata
GOVERSION := 1.9.2
PROJECT := $(CURRENT_DIR:$(GOPATH)/src/%=%)
OWNER := $(notdir $(patsubst %/,%,$(dir $(PROJECT))))
NAME := $(notdir $(PROJECT))
GIT_COMMIT ?= $(shell git rev-parse --short HEAD)
VERSION := $(shell awk -F\" '/Version/ { print $$2; exit }' "${CURRENT_DIR}/version/version.go")
EXTERNAL_TOOLS = \
	github.com/golang/dep/cmd/dep

# Current system information
GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)

# Default os-arch combination to build
XC_OS ?= darwin linux windows
XC_ARCH ?= 386 amd64 arm
XC_EXCLUDE ?= darwin/arm windows/arm

# GPG Signing key (blank by default, means no GPG signing)
GPG_KEY ?=

# List of ldflags
LD_FLAGS ?= \
	-s \
	-w \
	-X ${PROJECT}/version.Name=${NAME} \
	-X ${PROJECT}/version.GitCommit=${GIT_COMMIT}

# List of tests to run
TEST ?= ./...

# Create a cross-compile target for every os-arch pairing. This will generate
# a make target for each os/arch like "make linux/amd64" as well as generate a
# meta target (build) for compiling everything.
define make-xc-target
  $1/$2:
  ifneq (,$(findstring ${1}/${2},$(XC_EXCLUDE)))
		@printf "%s%20s %s\n" "-->" "${1}/${2}:" "${PROJECT} (excluded)"
  else
		@printf "%s%20s %s\n" "-->" "${1}/${2}:" "${PROJECT}"
		@docker run \
			--interactive \
			--rm \
			--dns="8.8.8.8" \
			--volume="${CURRENT_DIR}:/go/src/${PROJECT}" \
			--workdir="/go/src/${PROJECT}" \
			"golang:${GOVERSION}" \
			env \
				CGO_ENABLED="0" \
				GOOS="${1}" \
				GOARCH="${2}" \
				go build \
				  -a \
					-o="pkg/${NAME}${3}_${1}_${2}" \
					-ldflags "${LD_FLAGS}" \
					-tags "${GOTAGS}"
  endif
  .PHONY: $1/$2

  $1:: $1/$2
  .PHONY: $1

  build:: $1/$2
  .PHONY: build
endef
$(foreach goarch,$(XC_ARCH),$(foreach goos,$(XC_OS),$(eval $(call make-xc-target,$(goos),$(goarch),$(if $(findstring windows,$(goos)),.exe,)))))

# bootstrap installs the necessary go tools for development or build.
bootstrap:
	@echo "==> Bootstrapping ${PROJECT}"
	@for t in ${EXTERNAL_TOOLS}; do \
		echo "--> Installing $$t" ; \
		go get -u "$$t"; \
	done
.PHONY: bootstrap

# deps updates all dependencies for this project.
deps:
	@echo "==> Updating deps for ${PROJECT}"
	@dep ensure -update
	@dep prune
.PHONY: deps

# dev builds and installs the project locally.
dev:
	@echo "==> Installing ${NAME} for ${GOOS}/${GOARCH}"
	@rm -f "${GOPATH}/pkg/${GOOS}_${GOARCH}/${PROJECT}/version.a" # ldflags change and go doesn't detect
	@env \
		CGO_ENABLED="0" \
		go install \
			-ldflags "${LD_FLAGS}" \
			-tags "${GOTAGS}"
.PHONY: dev

# dist builds the binaries and then signs and packages them for distribution
dist:
ifndef GPG_KEY
	@echo "==> ERROR: No GPG key specified! Without a GPG key, this release cannot"
	@echo "           be signed. Set the environment variable GPG_KEY to the ID of"
	@echo "           the GPG key to continue."
	@exit 127
else
	@$(MAKE) -f "${MKFILE_PATH}" _cleanup
	@$(MAKE) -f "${MKFILE_PATH}" -j4 build
	@$(MAKE) -f "${MKFILE_PATH}" _checksum _sign
endif
.PHONY: dist

# test runs the test suite.
test: fmtcheck errcheck
	@echo "==> Testing ${NAME}"
	@go test -v -timeout=300s -parallel=20 -tags="${GOTAGS}" ${GOFILES} ${TESTARGS}
.PHONY: test

# test-race runs the test suite.
test-race: fmtcheck errcheck
	@echo "==> Testing ${NAME} (race)"
	@go test -v -timeout=300s -race -tags="${GOTAGS}" ${GOFILES} ${TESTARGS}
.PHONY: test-race

# _cleanup removes any previous binaries
_cleanup:
	@rm -rf "${CURRENT_DIR}/pkg/"
	@rm -rf "${CURRENT_DIR}/bin/"

# _checksum produces the checksums for the binaries in pkg
_checksum:
	@cd "${CURRENT_DIR}/pkg" && \
		shasum --algorithm 256 * > ${CURRENT_DIR}/pkg/${NAME}_${VERSION}_SHA256SUMS && \
		cd - &>/dev/null
.PHONY: _checksum

# _sign signs the binaries using the given GPG_KEY. This should not be called
# as a separate function.
_sign:
	@echo "==> Signing ${PROJECT} at v${VERSION}"
	@gpg \
		--default-key "${GPG_KEY}" \
		--detach-sig "${CURRENT_DIR}/pkg/${NAME}_${VERSION}_SHA256SUMS"
	@git commit \
		--allow-empty \
		--gpg-sign="${GPG_KEY}" \
		--message "Release v${VERSION}" \
		--quiet \
		--signoff
	@git tag \
		--annotate \
		--create-reflog \
		--local-user "${GPG_KEY}" \
		--message "Version ${VERSION}" \
		--sign \
		"v${VERSION}" master
	@echo "--> Do not forget to run:"
	@echo ""
	@echo "    git push && git push --tags"
	@echo ""
	@echo "And then upload the binaries in dist/!"
.PHONY: _sign

fmt:
	@echo "==> Fixing source code with gofmt..."
	gofmt -s -w ./backend ./version ./main.go

buildall: darwin/386 darwin/amd64 freebsd/386 freebsd/amd64 freebsd/arm linux/386 linux/amd64 linux/arm windows/386 windows/amd64
.PHONY: all

fmtcheck:
	@sh -c "'$(CURDIR)/scripts/fmtcheck.sh'"
.PHONY: fmtcheck

errcheck:
	@sh -c "'$(CURDIR)/scripts/errcheck.sh'"
.PHONY: errcheck