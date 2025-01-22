build:
	docker build . -t harbor.cognifog.rid-intrasoft.eu/orchestrator-agent/orch-agent-ocm-description-svc:dev

push:
	docker push harbor.cognifog.rid-intrasoft.eu/orchestrator-agent/orch-agent-ocm-description-svc:dev

helm:
	helm upgrade -i ocm-descriptor ocm-descriptor/ -n jobmanager
