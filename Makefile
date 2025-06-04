build:
	docker build . -t harbor.cognifog.rid-intrasoft.eu/orchestrator-agent/orch-agent-ocm-description-svc:dev

push:
	docker push harbor.cognifog.rid-intrasoft.eu/orchestrator-agent/orch-agent-ocm-description-svc:dev

sidecar-push:
	docker pull registry.atosresearch.eu:18516/edgeharbor/ocm-descriptor-sidecar
	docker tag registry.atosresearch.eu:18516/edgeharbor/ocm-descriptor-sidecar harbor.cognifog.rid-intrasoft.eu/orchestrator-agent/orch-agent-ocm-descriptor-sidecar:latest
	docker push harbor.cognifog.rid-intrasoft.eu/orchestrator-agent/orch-agent-ocm-descriptor-sidecar:latest

install:
	helm upgrade -i ocm-descriptor ocm-descriptor/ -n jobmanager

uninstall:
	helm uninstall ocm-descriptor -n jobmanager

template:
	helm template --name-template ocm-descriptor ocm-descriptor/ -n cognifog-dev \
	>> jenkins/manifests.yaml