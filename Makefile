VERSION=`git describe --tags`
BUILD=`date +%FT%T%z`
BRANCH=`git branch | sed -n '/\* /s///p'`
IMAGE_CONFIG=`cat zke_image.yml`

LDFLAGS=-ldflags "-w -s -X main.version=${VERSION} -X main.build=${BUILD} -X github.com/zdnscloud/singlecloud/pkg/zke.singleCloudVersion=${VERSION} -X 'github.com/zdnscloud/singlecloud/vendor/github.com/zdnscloud/zke/types.imageConfig=${IMAGE_CONFIG}'"
GOSRC = $(shell find . -type f -name '*.go')

build: singlecloud

singlecloud: $(GOSRC) 
	go build ${LDFLAGS} cmd/singlecloud/singlecloud.go

docker: build-image
	docker push zdnscloud/singlecloud:${BRANCH}

build-image:
	docker build -t zdnscloud/singlecloud:${BRANCH} --build-arg version=${VERSION} --build-arg buildtime=${BUILD} --no-cache .
	docker image prune -f

clean:
	rm -rf singlecloud

clean-image:
	docker rmi zdnscloud/singlecloud:${VERSION}

.PHONY: clean install
