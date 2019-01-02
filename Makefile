VERSION_MAJOR ?= 0
VERSION_MINOR ?= 5
VERSION_BUILD ?= 3
VERSION ?= v$(VERSION_MAJOR).$(VERSION_MINOR).$(VERSION_BUILD)

GOOS ?= $(shell go env GOOS)

ORG := github.com
OWNER := inwinstack
REPOPATH ?= $(ORG)/$(OWNER)/pa-controller

$(shell mkdir -p ./out)

.PHONY: build
build: out/controller

.PHONY: out/controller
out/controller:
	GOOS=$(GOOS) go build \
	  -ldflags="-s -w -X $(REPOPATH)/pkg/version.version=$(VERSION)" \
	  -a -o $@ cmd/main.go

.PHONY: dep 
dep:
	dep ensure

.PHONY: test
test:
	./hack/test-go.sh

.PHONY: build_image
build_image:
	docker build -t $(OWNER)/pa-controller:$(VERSION) .

.PHONY: push_image
push_image:
	docker push $(OWNER)/pa-controller:$(VERSION)

.PHONY: clean
clean:
	rm -rf out/

