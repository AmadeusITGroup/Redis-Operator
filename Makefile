ARTIFACT_OPERATOR=redis-operator
ARTIFACT_INITCONTAINER=init-container

# 0.0 shouldn't clobber any released builds
PREFIX=redisoperator/
#PREFIX = gcr.io/google_containers/

SOURCES := $(shell find $(SOURCEDIR) ! -name "*_test.go" -name '*.go')

CMDBINS := operator redisnode

TAG?=$(shell git tag|tail -1)
COMMIT=$(shell git rev-parse HEAD)
VERSION?=$(shell cat version.txt)
DATE=$(shell date +%Y-%m-%d/%H:%M:%S )
BUILDINFOPKG=github.com/amadeusitgroup/redis-operator/pkg/utils
LDFLAGS= -ldflags "-w -X ${BUILDINFOPKG}.TAG=${TAG} -X ${BUILDINFOPKG}.COMMIT=${COMMIT} -X ${BUILDINFOPKG}.VERSION=${VERSION} -X ${BUILDINFOPKG}.BUILDTIME=${DATE} -s"

all: build

plugin: build-kubectl-plugin install-plugin

install-plugin:
	./tools/install-plugin.sh

build-%:
	CGO_ENABLED=0 go build -i -installsuffix cgo ${LDFLAGS} -o bin/$* ./cmd/$*

buildlinux-%: ${SOURCES}
	CGO_ENABLED=0 GOOS=linux go build -i -installsuffix cgo ${LDFLAGS} -o docker/$*/$* ./cmd/$*/main.go

container-%: buildlinux-%
	@cd docker/$* && docker build -t $(PREFIX)$*:$(TAG) .

container-redisnode: buildlinux-redisnode
	@cd docker/redisnode && docker build -t $(PREFIX)redisnode:$(TAG) .

build: $(addprefix build-,$(CMDBINS))

buildlinux: $(addprefix buildlinux-,$(CMDBINS))

container: $(addprefix container-,$(CMDBINS))

test:
	./go.test.sh


push-%: container-%
	docker push $(PREFIX)$*:$(TAG)

push: $(addprefix push-,$(CMDBINS))

clean:
	rm -f ${ARTIFACT_OPERATOR}

# Install all the build and lint dependencies
setup:
	go get -u github.com/alecthomas/gometalinter
	gometalinter --install
	echo "make check" > .git/hooks/pre-commit
	chmod +x .git/hooks/pre-commit
.PHONY: setup

# gofmt and goimports all go files
fmt:
	find . -name '*.go' -not -wholename './vendor/*' | while read -r file; do gofmt -w -s "$$file"; goimports -w "$$file"; done
.PHONY: fmt

# Run all the linters
lint:
	gometalinter --vendor ./... -e pkg/client -e _generated -e test --deadline 15m -D gocyclo -D errcheck -D aligncheck -D maligned -D gas
.PHONY: lint

# Run only fast linters
lint-fast:
	gometalinter --fast --vendor ./... -e pkg/client -e _generated -e test --deadline 9m -D gocyclo -D errcheck -D aligncheck -D maligned
.PHONY: lint-fast


.PHONY: build push clean test
