/*
Copyright 2023-2024 Bull SAS

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package controllers

import (
	"bytes"
	"context"
	"encoding/json"
	"etsn/server/ocm-description-service/models"
	"etsn/server/ocm-description-service/responses"
	"etsn/server/ocm-description-service/utils/logs"
	"io"
	"net/http"
	"os"
)

var (
	jobmanagerBaseURL = os.Getenv("JOBMANAGER_URL") // "http://10.160.3.20:32300/"
	// lighthouseBaseURL  = os.Getenv("LIGHTHOUSE_BASE_URL")
	// apiV3              = "/api/v3"
	// matchmackerBaseURL = os.Getenv("MATCHMAKING_URL")
)

// PullJobs example
//
// @Summary		Pull and execute jobs from job manager
// @Description	Pull and execute jobs
// @Tags			jobs
// @Accept			json
// @Produce			json
// @Success		200				{array}		models.Job "List of executed jobs"
// @Failure		400				{object}	string	"Bad Request"
// @Failure		500				{object}	string	"Internal Server Error"
// @Router			/deploy-manager/execute [get]
func (server *Server) PullJobs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	jobs := []models.Job{}

	if err := models.InClusterConfig(); err != nil {
		logs.Logger.Println("Kubeconfig error occurred:", err)
	}

	ownerId, err := models.FetchClusterManagerUID("cluster-manager")
	if err != nil {
		logs.Logger.Println("Error fetching cluster manager UID:", err)
		responses.ERROR(w, http.StatusInternalServerError, err)
		return
	}

	respJobs, err := getExecutableJobs(ctx, w, r, ownerId)
	if err != nil {
		logs.Logger.Println("Error getting executable jobs:", err)
		return
	}
	defer respJobs.Body.Close()

	bodyJobs, err := io.ReadAll(respJobs.Body)
	if err != nil {
		logs.Logger.Println("Error reading response body:", err)
		responses.ERROR(w, http.StatusBadRequest, err)
		return
	}
	logs.Logger.Println("Job's body:", string(bodyJobs))

	if err = json.Unmarshal(bodyJobs, &jobs); err != nil {
		logs.Logger.Println("Error unmarshaling response body:", err)
		responses.ERROR(w, respJobs.StatusCode, err)
		return
	}

	executeJobs(ctx, jobs, w, r, ownerId)
	responses.JSON(w, http.StatusOK, jobs)
}

func getExecutableJobs(ctx context.Context, w http.ResponseWriter, r *http.Request, ownerId string) (*http.Response, error) {
	logs.Logger.Println("Requesting Jobs...")
	reqJobs, err := http.NewRequestWithContext(ctx, "GET", jobmanagerBaseURL+"jobmanager/jobs/executable/ocm/"+ownerId, http.NoBody)
	if err != nil {
		logs.Logger.Println("Error creating new request:", err)
		responses.ERROR(w, http.StatusUnprocessableEntity, err)
		return nil, err
	}
	reqJobs.Header.Add("Authorization", r.Header.Get("Authorization"))

	client := &http.Client{}
	respJobs, err := client.Do(reqJobs)
	if err != nil {
		logs.Logger.Println("Error performing request to job manager:", err)
		responses.ERROR(w, http.StatusServiceUnavailable, err)
		return nil, err
	}
	return respJobs, nil
}

func executeJobs(ctx context.Context, jobs []models.Job, w http.ResponseWriter, r *http.Request, ownerId string) {
	for i := range jobs {
		job := &jobs[i]
		logs.Logger.Println("Executing Job:", job.ID)

		if job.Target.NodeName == "" {
			logs.Logger.Println("No targets were provided")
			continue
		}

		job.OwnerID = ownerId
		if err := job.PromoteJob(r.Header.Get("Authorization"), job.OwnerID); err != nil {
			logs.Logger.Println("Error promoting job:", err)
			responses.ERROR(w, http.StatusUnprocessableEntity, err)
			continue
		}

		executedJob, err := models.Execute(job)
		if err != nil {
			logs.Logger.Println("Error executing job:", err)
			// responses.ERROR(w, http.StatusUnprocessableEntity, err)
			continue
		}

		*job = *executedJob

		jobBody, err := json.Marshal(job)
		if err != nil {
			logs.Logger.Println("Error marshaling job:", err)
			continue
		}

		updateJob(ctx, job, w, r, jobBody)
	}
}

func updateJob(ctx context.Context, job *models.Job, w http.ResponseWriter, r *http.Request, jobBody []byte) {
	reqState, err := http.NewRequestWithContext(ctx, "PUT", jobmanagerBaseURL+"jobmanager/jobs", bytes.NewReader(jobBody))
	if err != nil {
		logs.Logger.Println("Error creating update job request:", err)
		responses.ERROR(w, http.StatusUnprocessableEntity, err)
		return
	}

	query := reqState.URL.Query()
	query.Add("id", job.ID)
	query.Add("orchestrator", "ocm")
	reqState.URL.RawQuery = query.Encode()

	reqState.Header.Add("Authorization", r.Header.Get("Authorization"))

	client := &http.Client{}
	resp, err := client.Do(reqState)
	if err != nil {
		logs.Logger.Println("Error performing update job request:", err)
		responses.ERROR(w, http.StatusServiceUnavailable, err)
		return
	}
	defer resp.Body.Close()

	logs.Logger.Println("Update Job Response:", resp.Status)
}
