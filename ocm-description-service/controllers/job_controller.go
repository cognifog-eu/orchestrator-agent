/*
Copyright 2023 Bull SAS

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
	"encoding/json"
	"etsn/server/ocm-description-service/models"
	"etsn/server/ocm-description-service/responses"
	"etsn/server/ocm-description-service/utils/logs"
	"fmt"
	"io"
	"net/http"
	"os"
)

var (
	jobmanagerBaseURL = os.Getenv("JOBMANAGER_URL") // "http://jobmanager-service:8082"
	// lighthouseBaseURL  = os.Getenv("LIGHTHOUSE_BASE_URL")
	// apiV3              = "/api/v3"
	// matchmackerBaseURL = os.Getenv("MATCHMAKING_URL")
)

// PullJobs example
//
//	@Description	pull and execute jobs
//	@ID				pull-and-execute-jobs
//	@Accept			json
//	@Produce		json
//	@Param			Authorization	header		string	true	"Authentication header"
//	@Success		200				{string}	string	"Ok"
//	@Router			/deploy-manager/execute [get]
func (server *Server) PullJobs(w http.ResponseWriter, r *http.Request) {

	jobs := []models.Job{}
	// get jobs with specific state; CREATED for now
	logs.Logger.Println("Requesting Jobs...")
	reqJobs, err := http.NewRequest("GET", jobmanagerBaseURL+"/jobmanager/jobs/executable/orchestrator/ocm", http.NoBody)
	if err != nil {
		responses.ERROR(w, http.StatusUnprocessableEntity, err)
		return
	}
	logs.Logger.Println("GET Request to Job Manager being created: ")
	logs.Logger.Println(reqJobs.URL)
	// add bearer
	reqJobs.Header.Add("Authorization", r.Header.Get("Authorization"))

	// do request
	client := &http.Client{}
	respJobs, err := client.Do(reqJobs)
	if err != nil {
		logs.Logger.Println("ERROR " + err.Error())
		responses.ERROR(w, http.StatusServiceUnavailable, err)
		return
	}
	defer respJobs.Body.Close()

	bodyJobs, err := io.ReadAll(respJobs.Body)
	if err != nil {
		logs.Logger.Println("ERROR " + err.Error())
		responses.ERROR(w, http.StatusBadRequest, err)
		return
	}
	logs.Logger.Println("Job's body: " + string(bodyJobs))
	err = json.Unmarshal(bodyJobs, &jobs)
	if err != nil {
		logs.Logger.Println("ERROR " + err.Error())
		responses.ERROR(w, respJobs.StatusCode, err)
		return
	}

	// create the in-cluster config
	err = models.InClusterConfig()
	if err != nil {
		logs.Logger.Println("Kubeconfig error occured", err)
	}
	// for each job, call exec
	for _, job := range jobs {
		// execute job -> creates manifestWork and deploy it, update UID, State, locker=false -> unlocked
		logs.Logger.Println("Executing Job: " + job.ID.String())
		if len(job.Targets) < 1 {
			logs.Logger.Println("No target were provided...")
		} else {
			job, err := models.Execute(&job)
			if err != nil {
				logs.Logger.Println("Error occurred during Job execution...")
			}
			// HTTP PUT to update UUIDs, State into JOB MANAGER -> updateJob call
			logs.Logger.Println("Job executed, sending details to Job Manager...")
			jobBody, err := json.Marshal(job)
			if err != nil {
				logs.Logger.Println("Could not unmarshall job...", err)
			}
			fmt.Printf("Job details: %#v", job)
			reqState, err := http.NewRequest("PUT", jobmanagerBaseURL+"jobmanager/jobs/"+job.ID.String(), bytes.NewReader(jobBody))
			query := reqState.URL.Query()
			query.Add("uuid", job.UUID.String())
			query.Add("orchestrator", "ocm")
			query.Encode()
			logs.Logger.Println("PUT Request to Job Manager being created: ")
			logs.Logger.Println(reqState.URL)

			reqState.Header.Add("Authorization", r.Header.Get("Authorization"))
			if err != nil {
				responses.ERROR(w, http.StatusUnprocessableEntity, err)
				return
			}
			// do request
			client2 := &http.Client{}
			resp, err := client2.Do(reqState)
			fmt.Println("Update Job Request " + logs.FormatRequest(reqState))
			logs.Logger.Println("Update Job Response " + resp.Status)
			if err != nil {
				logs.Logger.Println("Error occurred during Job details notification...")
				responses.ERROR(w, resp.StatusCode, err)
				// TODO retry? rollback?
				// keep executing
			}
			defer reqState.Body.Close()
		}
	}
	// TODO: update the jobs list with updated jobs
	responses.JSON(w, http.StatusOK, jobs)
}
