PLUGIN_NAME  = rchicoli/docker-log-elasticsearch
PLUGIN_TAG  ?= development

BASE_DIR    ?= .
ROOTFS_DIR   = ./plugin/rootfs

.PHONY: all clean docker rootfs create install enable

all: clean docker rootfs create enable clean

clean:
	@echo ""
	@echo "# clean: removes rootfs directory"
	test -d ${ROOTFS_DIR} && rm -rf ${ROOTFS_DIR} || true
	# 	docker rmi $(DOCKER_IMAGE):$(APP_VERSION)

docker:
	@echo ""
	@echo "### docker build: rootfs image with splunk-log-plugin"
	docker build -t ${PLUGIN_NAME}:rootfs ${BASE_DIR}

rootfs:
	@echo ""
	@echo "### create rootfs directory in ./plugin/rootfs"
	mkdir -p ${BASE_DIR}/plugin/rootfs
	docker create --name tmprootfs ${PLUGIN_NAME}:rootfs

	@echo
	docker export tmprootfs | tar -x -C ${BASE_DIR}/plugin/rootfs
	docker rm -vf tmprootfs

create:
	@echo ""
	@echo "### remove existing plugin ${PLUGIN_NAME}:${PLUGIN_TAG} if exists"
	docker plugin rm -f ${PLUGIN_NAME}:${PLUGIN_TAG} || true

	@echo
	@echo "### create new plugin ${PLUGIN_NAME}:${PLUGIN_TAG} from ./plugin"
	docker plugin create ${PLUGIN_NAME}:${PLUGIN_TAG} ${BASE_DIR}/plugin

install:
	docker plugin install ${PLUGIN_NAME}:${PLUGIN_TAG} --alias elasticsearch

enable:
	@echo ""
	@echo "### enable plugin ${PLUGIN_NAME}:${PLUGIN_TAG}"
	docker plugin enable ${PLUGIN_NAME}:${PLUGIN_TAG}

push: clean docker rootfs create enable
	@echo ""
	@echo "### push plugin ${PLUGIN_NAME}:${PLUGIN_TAG}"
	docker plugin push ${PLUGIN_NAME}:${PLUGIN_TAG}
