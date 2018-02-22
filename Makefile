PLUGIN_NAME         := rchicoli/docker-log-elasticsearch
PLUGIN_TAG          ?= development

BASE_DIR            ?= .
ROOTFS_DIR          ?= $(BASE_DIR)/plugin/rootfs
DOCKER_COMPOSE_FILE ?= $(BASE_DIR)/docker/docker-compose.yml
DOCKER_DIR          ?= $(BASE_DIR)/docker
SCRIPTS_DIR         ?= $(BASE_DIR)/scripts
TESTS_DIR           ?= $(BASE_DIR)/tests

CLIENT_VERSION      ?= 5

SHELL               := /bin/bash
SYSCTL              := $(shell which sysctl)
DOCKER_COMPOSE      := $(shell which docker-compose)

.PHONY: all clean rootfs plugin install enable

all: clean build rootfs plugin enable clean

clean:
	@echo ""
	test -d $(ROOTFS_DIR) && rm -rf $(ROOTFS_DIR) || true
	# docker rmi $(DOCKER_IMAGE):$(APP_VERSION)

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

push: clean build rootfs
	@echo ""
	docker plugin push $(PLUGIN_NAME):$(PLUGIN_TAG)

client_version:
ifeq ($(CLIENT_VERSION),6)
    ELASTIC_VERSION=$(DOCKER_DIR)/elastic-v6.yml
else ifeq ($(CLIENT_VERSION),5)
ifeq ($(TLS),true)
    ELASTIC_VERSION=$(DOCKER_DIR)/elastic-v5-tls.yml
else
    ELASTIC_VERSION=$(DOCKER_DIR)/elastic-v5.yml
endif
else ifeq ($(CLIENT_VERSION),2)
    ELASTIC_VERSION=$(DOCKER_DIR)/elastic-v2.yml
else ifeq ($(CLIENT_VERSION),1)
    ELASTIC_VERSION=$(DOCKER_DIR)/elastic-v1.yml
endif

docker_compose:
ifeq (, $(DOCKER_COMPOSE))
	$(error "docker-compose: command not found")
endif

docker_log_options:
ifdef DOCKER_LOG_OPTIONS
DOCKER_LOG_OPTIONS := -f $(DOCKER_LOG_OPTIONS)
endif

deploy_elasticsearch: docker_compose client_version docker_log_options
ifeq (, $(SYSCTL))
	$(error "sysctl: command not found")
endif

	# max virtual memory areas vm.max_map_count [65530] is too low, increase to at least [262144]
	$(SYSCTL) -q -w vm.max_map_count=262144

	# create and run elasticsearch as a container
	docker-compose -f "$(DOCKER_COMPOSE_FILE)" -f "$(ELASTIC_VERSION)" up -d elasticsearch

stop_elasticsearch: docker_compose client_version
	docker-compose -f "$(DOCKER_COMPOSE_FILE)" stop elasticsearch

skip:
ifeq ($(SKIP),true)
SKIP := :
else
SKIP :=
endif

undeploy_elasticsearch: docker_compose client_version skip
	$(SKIP) docker-compose -f "$(DOCKER_COMPOSE_FILE)" rm --stop --force elasticsearch

deploy_webapper: docker_compose client_version deploy_elasticsearch
	# create a container for logging to elasticsearch
	$(SCRIPTS_DIR)/wait-for-it.sh elasticsearch 9200 docker-compose -f "$(DOCKER_COMPOSE_FILE)" -f "$(ELASTIC_VERSION)" $(DOCKER_LOG_OPTIONS) up -d webapper

stop_webapper: docker_compose client_version
	# create a container for logging to elasticsearch
	docker-compose -f "$(DOCKER_COMPOSE_FILE)" stop webapper

undeploy_webapper: skip
	# create a container for logging to elasticsearch
	$(SKIP) docker-compose -f "$(DOCKER_COMPOSE_FILE)" rm -s -f webapper

create_environment: deploy_elasticsearch deploy_webapper

delete_environment: undeploy_webapper undeploy_elasticsearch

acceptance_tests: create_environment
	bats $(TESTS_DIR)/acceptance-tests/$(BATS_TESTFILE)

integration_tests: create_environment
	bats $(TESTS_DIR)/integration-tests/$(BATS_TESTFILE)