PLUGIN_NAME         := rchicoli/docker-log-elasticsearch
PLUGIN_TAG          ?= development

BASE_DIR            ?= .
ROOTFS_DIR          ?= $(BASE_DIR)/plugin/rootfs
DOCKER_COMPOSE_FILE ?= $(BASE_DIR)/docker/docker-compose.yml
DOCKER_DIR          ?= $(BASE_DIR)/docker
SCRIPTS_DIR         ?= $(BASE_DIR)/scripts
TESTS_DIR           ?= $(BASE_DIR)/tests

CLIENT_VERSION      ?= 5
DEBUG_LEVEL			?= debug
TZ					?= Europe/Berlin

SHELL               := /bin/bash
SYSCTL              := $(shell which sysctl)
DOCKER_COMPOSE      := $(shell which docker-compose)

.PHONY: all

all: clean docker_build plugin_create plugin_set plugin_enable clean

local: clean build unit_tests plugin_create plugin_set plugin_enable clean

clean:
	@echo ""
	test -d $(ROOTFS_DIR) && rm -rf $(ROOTFS_DIR) || true

build:
	@echo ""
	CGO_ENABLED=0 go build -v -a -installsuffix cgo -o docker-log-elasticsearch
	docker build -t $(PLUGIN_NAME):rootfs -f $(BASE_DIR)/Dockerfile.local $(BASE_DIR)

docker_build:
	@echo ""
	docker build -t $(PLUGIN_NAME):rootfs $(BASE_DIR)

rootfs:
	@echo ""
	mkdir -p $(BASE_DIR)/plugin/rootfs
	docker create --name tmprootfs ${PLUGIN_NAME}:rootfs

	@echo
	docker export tmprootfs | tar -x -C ${BASE_DIR}/plugin/rootfs
	docker rm -vf tmprootfs

plugin_create: rootfs
	@echo ""
	docker plugin rm -f $(PLUGIN_NAME):$(PLUGIN_TAG) || true

	@echo
	docker plugin create $(PLUGIN_NAME):$(PLUGIN_TAG) ${BASE_DIR}/plugin

plugin_install:
	docker plugin install $(PLUGIN_NAME):$(PLUGIN_TAG) --alias elasticsearch

plugin_enable:
	@echo ""
	docker plugin enable $(PLUGIN_NAME):$(PLUGIN_TAG)

plugin_push:
	@echo ""
	docker plugin push $(PLUGIN_NAME):$(PLUGIN_TAG)

plugin_set:
	@echo ""
	docker plugin set $(PLUGIN_NAME):$(PLUGIN_TAG) LOG_LEVEL=$(DEBUG_LEVEL) TZ=$(TZ)

push: plugin_push

client_version:
ifeq ($(CLIENT_VERSION),6)
ifeq ($(TLS),true)
    ELASTIC_VERSION=$(DOCKER_DIR)/elastic-v6-tls.yml
else
    ELASTIC_VERSION=$(DOCKER_DIR)/elastic-v6.yml
endif
else ifeq ($(CLIENT_VERSION),5)
ifeq ($(TLS),true)
    ELASTIC_VERSION=$(DOCKER_DIR)/elastic-v5-tls.yml
else
    ELASTIC_VERSION=$(DOCKER_DIR)/elastic-v5.yml
endif
else ifeq ($(CLIENT_VERSION),2)
ifeq ($(TLS),true)
    ELASTIC_VERSION=$(DOCKER_DIR)/elastic-v2-tls.yml
else
    ELASTIC_VERSION=$(DOCKER_DIR)/elastic-v2.yml
endif
else ifeq ($(CLIENT_VERSION),1)
    ELASTIC_VERSION=$(DOCKER_DIR)/elastic-v1.yml
endif

#####################
#    ENVIRONMENT    #
#####################

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

deploy_nginx: docker_compose client_version deploy_elasticsearch
	$(SCRIPTS_DIR)/wait-for-it.sh elasticsearch 9200 docker-compose -f "$(DOCKER_COMPOSE_FILE)" -f "$(ELASTIC_VERSION)" $(DOCKER_LOG_OPTIONS) up -d nginx

undeploy_nginx: docker_compose skip
	# create a container for logging to elasticsearch
	$(SKIP) docker-compose -f "$(DOCKER_COMPOSE_FILE)" rm -s -f nginx

#####################
#      TESTS        #
#####################

unit_tests:
	go test -cover -race -v ./...

acceptance_tests:
	$(SCRIPTS_DIR)/basht.sh --test-dir $(TESTS_DIR)/acceptance-tests

acceptance_test_file:
	$(SCRIPTS_DIR)/basht.sh --test-file $(TESTS_DIR)/acceptance-tests/$(BASHT_TESTFILE)

integration_tests:
	$(SCRIPTS_DIR)/basht.sh --test-dir $(TESTS_DIR)/integration-tests

integration_test_file:
	$(SCRIPTS_DIR)/basht.sh --test-file $(TESTS_DIR)/integration-tests/$(BASHT_TESTFILE)

create_environment: deploy_elasticsearch deploy_webapper

delete_environment: undeploy_elasticsearch undeploy_webapper
