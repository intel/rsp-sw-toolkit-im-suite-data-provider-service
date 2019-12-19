# This Makefile functions similarly to a compose file; the reason for using it
# instead of a compose file + docker-compose is that docker-compose has several
# bugs associated with modify paths in a way no longer compatible with toolbox.
# Furthermore, since EdgeX relies on several deprecated, non-swarm-compatible
# compose directives, it can't work with Swarm, whereas passing secrets requires
# Swarm. To avoid the issues, this Makefile just runs the docker commands itself.

REPO?=rsp/data-provider-service
TAG?=latest

CONTAINERS=server cloud-connector mqtt data-provider

.PHONY: up down build update net edgex
.PHONY: $(CONTAINERS)

up: net edgex $(CONTAINERS)
down:
	-docker rm -f $(CONTAINERS)
	-docker-compose -f edgex-compose.yml down
	-docker network rm edgex-network

null  :=
space := $(null) #
comma := ,
COMMA_CONTAINERS := $(subst $(space),$(comma),$(strip $(CONTAINERS)))

up: net server edgex cloud-connector data-provider

net:
	-docker network create edgex-network

edgex:
	docker-compose -f edgex-compose.yml up -d

data-provider:
	-docker rm -f $@
	docker run -it --rm --name $@ \
		-v $$(PWD)/app/testdata:/run/secrets \
		-v $$(PWD)/app/config:/app/config \
		--net edgex-network \
		--env no_proxy="$(COMMA_CONTAINERS),edgex-core-consul,edgex-core-data" \
		--env NO_PROXY="$(COMMA_CONTAINERS),edgex-core-consul,edgex-core-data" \
		--env runtimeConfigPath="/app/config/configuration.json" \
		$(REPO):$(TAG)


mqtt:
	-docker rm -f $@
	docker run -d --name $@ \
		--net edgex-network \
		--env NO_PROXY="*" \
		--env no_proxy="*" \
		-v $$(PWD)/app/testdata/mosquitto.conf:/mosquitto/config/mosquitto.conf:ro \
		eclipse-mosquitto

server:
	-docker rm -f $@
	docker run -d --rm --name $@ \
		--net edgex-network \
		--network-alias asn_data \
		--network-alias sku_data \
		--network-alias clusterConfig \
		-v $$(PWD)/app/testdata:/files \
		-v $$(PWD)/app/testdata/nginx.conf:/etc/nginx/nginx.conf:ro \
		nginx

cloud-connector:
	-docker rm -f $@
	docker run --rm -d --name $@ \
		-v $$(PWD)/app/testdata:/files \
		-u 2000:2000 \
		--net edgex-network \
		--env no_proxy="asn_data,sku_data,clusterConfig" \
		--env NO_PROXY="asn_data,sku_data,clusterConfig" \
		--env runtimeConfigPath="/files/cc-config.json" \
		rsp/cloud-connector-service

