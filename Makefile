PLUGIN_NAME         := rchicoli/docker-log-elasticsearch
PLUGIN_TAG          ?= development

BASE_DIR            ?= .
ROOTFS_DIR          ?= $(BASE_DIR)/plugin/rootfs
DOCKER_COMPOSE_FILE ?= $(BASE_DIR)/docker/docker-compose.yml
SCRIPTS_DIR         ?= $(BASE_DIR)/scripts
TESTS_DIR           ?= $(BASE_DIR)/tests

SHELL               := /bin/bash
SYSCTL              := $(shell which sysctl)
DOCKER_COMPOSE      := $(shell which docker-compose)

.PHONY: all clean docker rootfs plugin install enable

all: clean build rootfs plugin enable clean

clean:
	@echo ""
	test -d $(ROOTFS_DIR) && rm -rf $(ROOTFS_DIR) || true
	# 	docker rmi $(DOCKER_IMAGE):$(APP_VERSION)

build:
	@echo ""
	docker build -t $(PLUGIN_NAME):rootfs $(BASE_DIR)

rootfs:
	@echo ""
	mkdir -p $(BASE_DIR)/plugin/rootfs
	docker create --name tmprootfs ${PLUGIN_NAME}:rootfs

	@echo
	docker export tmprootfs | tar -x -C ${BASE_DIR}/plugin/rootfs
	docker rm -vf tmprootfs

plugin:
	@echo ""
	docker plugin rm -f $(PLUGIN_NAME):$(PLUGIN_TAG) || true

	@echo
	docker plugin create $(PLUGIN_NAME):$(PLUGIN_TAG) ${BASE_DIR}/plugin

install:
	docker plugin install $(PLUGIN_NAME):$(PLUGIN_TAG) --alias elasticsearch

enable:
	@echo ""
	docker plugin enable $(PLUGIN_NAME):$(PLUGIN_TAG)

push: clean build rootfs create enable
	@echo ""
	docker plugin push $(PLUGIN_NAME):$(PLUGIN_TAG)

docker_compose:
ifeq (, $(DOCKER_COMPOSE))
	$(error "docker-compose: command not found")
endif

deploy_elasticsearch: docker_compose
ifeq (, $(SYSCTL))
	$(error "sysctl: command not found")
endif

	# max virtual memory areas vm.max_map_count [65530] is too low, increase to at least [262144]
	$(SYSCTL) -q -w vm.max_map_count=262144

	# create and run elasticsearch as a container
	docker-compose -f "$(DOCKER_COMPOSE_FILE)" up -d elasticsearch

stop_elasticsearch: docker_compose
	docker-compose -f "$(DOCKER_COMPOSE_FILE)" stop elasticsearch

undeploy_elasticsearch: docker_compose
	docker-compose -f "$(DOCKER_COMPOSE_FILE)" rm --stop --force elasticsearch

deploy_webapper: deploy_elasticsearch docker_compose
	# create a container for logging to elasticsearch
	$(SCRIPTS_DIR)/wait-for-it.sh elasticsearch 9200 docker-compose -f "$(DOCKER_COMPOSE_FILE)" up -d webapper

stop_webapper:
	# create a container for logging to elasticsearch
	docker-compose -f "$(DOCKER_COMPOSE_FILE)" stop webapper

undeploy_webapper:
	# create a container for logging to elasticsearch
	docker-compose -f "$(DOCKER_COMPOSE_FILE)" rm -s -f webapper

create_environment: deploy_elasticsearch deploy_webapper

delete_environment: stop_webapper stop_elasticsearch

acceptance_tests:
	bats $(TESTS_DIR)/main.bats