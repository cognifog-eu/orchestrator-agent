# COGNIFOG Orchestrator Agent

COGNIFOG Orchestrator Agent Service - Responsible for offloading Jobs to underlying OCM Orchestrator

# Getting Started

## Set enviroment variables

| Variable         | Description     |
| ---------------- | --------------- |
| JOBMANAGER_URL        | Upstream jobmanager URL to retrieve applications to be deployed           |
| LIGHTHOUSE_BASE_URL          | For future use      |
| MATCHMAKING_URL          | For future use             |

## Run application

`go run main.go`

## Make execute API call to retrieve new applications to be deployed

`curl -H "Authorization: Bearer $TOKEN" http://localhost:8083/deploy-manager/execute`