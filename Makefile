
# docker login --username=haierspi@aliyun.com registry.cn-hangzhou.aliyuncs.com
include .env
#export $(shell sed 's/=.*//' .env)

# These are the values we want to pass for Version and BuildTime
GitTag	= $(shell git describe --tags)
BuildTime=$(shell date +%FT%T%z)


# Go parameters
goCmd	=	go
version	=	$(shell cat VERSION)

# Setup the -ldflags option for go build here, interpolate the variable values
LDFLAGS=-ldflags "-X main.GitTag=$(GitTag) -X main.BuildTime=$(BuildTime) -X main.Version=$(version)"


goBuild	=	$(goCmd) build ${LDFLAGS}
goRun	=	$(goCmd) run ${LDFLAGS}

goClean	=	$(goCmd) clean
goTest	=	$(goCmd) test
goGet	=	$(goCmd) get -u

projectName		=	$(shell basename "$(PWD)")
projectRootDir	=	$(shell pwd)



sourceDir	=	$(projectRootDir)
bin			=	multi-node-webhook
cfgFile		=	$(projectRootDir)/config.json
keystoreDir	=	$(projectRootDir)/keystore
buildDir	=	$(projectRootDir)/build


.PHONY: all build run test clean build-linux build-windows build-macos 
all: test build
build:
	$(call init)
	$(goBuild) -o $(bin) -v $(sourceDir)
	@echo "Build OK"
	mv $(bin) $(buildDir)
run:
	$(call init)
	$(goRun)-v $(sourceDir)

test:
	@echo "Test Completed"
# $(goTest) -v -race -coverprofile=coverage.txt -covermode=atomic $(sourceAdmDir)
# $(goTest) -v -race -coverprofile=coverage.txt -covermode=atomic $(sourceNodeDir)
clean:
	rm -rf $(buildDir)

build-macos:
	$(call init)
	GOOS=darwin GOARCH=amd64 $(goBuild) -o $(buildDir)/$(bin)-macos-x86_64 -v $(sourceDir)

build-linux:
	$(call init)
	GOOS=linux GOARCH=amd64 $(goBuild) -o  $(buildDir)/$(bin)-linux-x86_64 -v $(sourceDir)

build-windows:
	$(call init)
	CGO_ENABLED=1 GOOS=windows GOARCH=amd64 CC="x86_64-w64-mingw32-gcc -fno-stack-protector -D_FORTIFY_SOURCE=0 -lssp" $(goBuild) -o $(buildDir)/$(bin)-win-x86_64.exe -v $(sourceDir)

define init
	@echo "Build Init"
	mkdir -p $(buildDir)
	@cp -rf $(cfgFile) $(buildDir)
endef
# @cp -rf $(keystoreDir) $(buildDir)