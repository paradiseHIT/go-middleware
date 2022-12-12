BUILD=go build
DOCKER?=docker
SOURCE_DIR=app
BIN_NAME=api-service
REGISTRY?=registry.cn-beijing.aliyuncs.com/adebug-middlewares
DOCKER_TAG?=0.0.1
TEMP_DIR_SERVER:=$(shell mktemp -d)

.PHONY: build clean
#setup:
#  go mod vendor
build:
	CGO_ENABLED=0 GOARCH=amd64 GOOS=linux ${BUILD} -o ${BIN_NAME} *.go
#release:
#  cd ${SOURCE_DIR}; CGO_ENABLED=0 GOARCH=amd64 GOOS=linux ${BUILD} -o ${BIN_NAME} .
#  cd ${SOURCE_DIR}; mv ${BIN_NAME} ${TEMP_DIR_SERVER}/appd
#  cp docker/Dockerfile ${TEMP_DIR_SERVER}/
#  cp conf/config.json.production ${TEMP_DIR_SERVER}/config.json
#  cd ${TEMP_DIR_SERVER}  &&  ${DOCKER} build  -t ${REGISTRY}/${BIN_NAME}:${DOCKER_TAG} .
#  ${DOCKER} push ${REGISTRY}/${BIN_NAME}:${DOCKER_TAG}

clean:
	rm -rf ${BIN_NAME}
