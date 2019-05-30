# This Makefile functions similarly to a compose file; the reason for using it
# instead of a compose file + docker-compose is that docker-compose has several
# bugs associated with modify paths in a way no longer compatible with toolbox.
# Furthermore, since EdgeX relies on several deprecated, non-swarm-compatible
# compose directives, it can't work with Swarm, whereas passing secrets requires
# Swarm. To avoid the issues, this Makefile just runs the docker commands itself.

REPO?=280211473891.dkr.ecr.us-west-2.amazonaws.com/rsp/data-provider-service
TAG?=latest

.PHONY: up down server edgex data-provider cloud-connector net

up: net server edgex cloud-connector data-provider

net:
	-docker network create edgex-network

server:
	-docker stop $@
	docker run -d --rm --name $@ \
		--net edgex-network \
		--network-alias asn_data \
		--network-alias sku_data \
		-v $$(PWD)/app/testdata:/files \
		-v $$(PWD)/app/testdata/nginx.conf:/etc/nginx/nginx.conf:ro \
		nginx

edgex:
	docker-compose -f edgex-compose.yml up -d

data-provider:
	-docker stop $@
	docker run -it --rm --name $@ \
		-v $$(PWD)/app/testdata:/run/secrets \
		-v $$(PWD)/app/config:/app/config \
		--net edgex-network \
		--env no_proxy="cloud-connector,edgex-core-consul,edgex-core-data" \
		--env NO_PROXY="cloud-connector,edgex-core-consul,edgex-core-data" \
		--env runtimeConfigPath="/app/config/configuration.json" \
		$(REPO):$(TAG)

cloud-connector:
	-docker stop $@
	docker run --rm -d --name $@ \
		-v $$(PWD)/app/testdata:/files \
		-u 2000:2000 \
		--net edgex-network \
		--env no_proxy="asn_data,sku_data" \
		--env NO_PROXY="asn_data,sku_data" \
		--env runtimeConfigPath="/files/cc-config.json" \
		280211473891.dkr.ecr.us-west-2.amazonaws.com/cloud-connector-service@sha256:8f7356f7ed9c3b9edde01b618fdf4266983ff42e89d9a5d30b90ff575f70610b

down:
	-docker stop cloud-connector data-provider
	-docker-compose -f edgex-compose.yml down
	-docker network rm edgex-network
