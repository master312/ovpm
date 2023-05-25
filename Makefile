.PHONY: deps build test bundle-webui clean-bundle bundle-swagger proto bundle build
docker-build:
	docker run --rm -i -t -e TRAVIS_GO_VERSION=$(TRAVIS_GO_VERSION) -e TRAVIS_BUILD_NUMBER=$(TRAVIS_BUILD_NUMBER) -e TRAVIS_TAG=$(TRAVIS_TAG) -v `pwd`:/fs/src/github.com/master312/ovpm -w /fs/src/github.com/master312/ovpm cadthecoder/ovpm-builder:latest
docker-build-shell:
	docker run --rm -i -t -e TRAVIS_GO_VERSION=$(TRAVIS_GO_VERSION) -e TRAVIS_BUILD_NUMBER=$(TRAVIS_BUILD_NUMBER) -e TRAVIS_TAG=$(TRAVIS_TAG) -v `pwd`:/fs/src/github.com/master312/ovpm -w /fs/src/github.com/master312/ovpm cadthecoder/ovpm-builder:latest /bin/bash

development-deps:
	# grpc related dependencies
	go get -u github.com/grpc-ecosystem/grpc-gateway/protoc-gen-grpc-gateway
	go get -u github.com/grpc-ecosystem/grpc-gateway/protoc-gen-swagger
	go get -u github.com/golang/protobuf/protoc-gen-go/...

	# static asset bundling
	go get github.com/kevinburke/go-bindata/...

	# for creating rpm, deb packages
	go get github.com/goreleaser/nfpm/cmd/nfpm@latest

	# webui related dependencies
	#pacman -Sy yarn

# Runs unit tests.
test:
	go test -count=1 -race -coverprofile=coverage.txt -covermode=atomic .

proto:
	protoc -I/usr/local/include -I api/pb/ -I/usr/local/include -I$(shell go list -m -f "{{.Dir}}" github.com/grpc-ecosystem/grpc-gateway)/third_party/googleapis api/pb/user.proto api/pb/vpn.proto api/pb/network.proto api/pb/auth.proto --grpc-gateway_out=logtostderr=true:api/pb
	protoc -I/usr/local/include -I api/pb/ -I/usr/local/include -I$(shell go list -m -f "{{.Dir}}" github.com/grpc-ecosystem/grpc-gateway)/third_party/googleapis api/pb/user.proto api/pb/vpn.proto api/pb/network.proto api/pb/auth.proto --go_out=plugins=grpc:api/pb

clean-bundle:
	@echo Cleaning up bundle/
	rm -rf bundle/
	mkdir -p bundle/

bundle-webui:
	@echo Bundling webui
	yarn --cwd webui/ovpm/ install
	yarn --cwd webui/ovpm/ build 
	cp -r webui/ovpm/build/* bundle

bundle-swagger: proto
	protoc -I/usr/local/include -I api/pb/  -I/usr/local/include -I$(shell go list -m -f "{{.Dir}}" github.com/grpc-ecosystem/grpc-gateway)/third_party/googleapis api/pb/user.proto api/pb/vpn.proto api/pb/network.proto api/pb/auth.proto --swagger_out=logtostderr=true:bundle

bundle: clean-bundle bundle-webui bundle-swagger
	go-bindata -pkg bundle -o bundle/bindata.go bundle/...

# Builds server and client binaries under ./bin folder. Accetps $VERSION env var.
build: bundle
	@echo Building
	rm -rf bin/
	mkdir -p bin/
	#CGO_ENABLED=0  GOOS=linux go build -ldflags="-w -X 'github.com/master312/ovpm.Version=$(VERSION)' -extldflags '-static'" -o ./bin/ovpm  ./cmd/ovpm
	#CGO_ENABLED=0  GOOS=linux go build -ldflags="-w -X 'github.com/master312/ovpm.Version=$(VERSION)' -extldflags '-static'" -o ./bin/ovpmd ./cmd/ovpmd

	# Link dynamically for now
	CGO_CFLAGS="-g -O2 -Wno-return-local-addr" go build -ldflags="-X 'github.com/master312/ovpm.Version=$(VERSION)'" -o ./bin/ovpm  ./cmd/ovpm
	CGO_CFLAGS="-g -O2 -Wno-return-local-addr" go build -ldflags="-X 'github.com/master312/ovpm.Version=$(VERSION)'" -o ./bin/ovpmd ./cmd/ovpmd

clean-dist:
	rm -rf dist/
	mkdir -p dist/

# Builds rpm and dep packages under ./dist folder. Accepts $VERSION env var.
dist: clean-dist build
	@echo Generating VERSION=$(VERSION) rpm and deb packages under dist/
	nfpm pkg -t ./dist/ovpm.rpm
	nfpm pkg -t ./dist/ovpm.deb
