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
	"cognifog/server/ocm-description-service/models"
	"cognifog/server/ocm-description-service/responses"
	"cognifog/server/ocm-description-service/utils/logs"
	"encoding/json"
	"io"
	"net/http"
	"os"
)

var (
	jobmanagerBaseURL  = os.Getenv("JOBMANAGER_URL")
	lighthouseBaseURL  = os.Getenv("LIGHTHOUSE_BASE_URL")
	apiV3              = "/api/v3"
	matchmackerBaseURL = os.Getenv("MATCHMAKING_URL")
)

func (server *Server) PullJobs(w http.ResponseWriter, r *http.Request) {

	jobs := []models.Job{}
	// get jobs with specific state; CREATED for now
	reqJobs, err := http.NewRequest("GET", jobmanagerBaseURL+"/jobs/executable", http.NoBody)
	if err != nil {
		responses.ERROR(w, http.StatusUnprocessableEntity, err)
		return
	}
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
	}
	logs.Logger.Println("Job's body: " + string(bodyJobs))
	err = json.Unmarshal(bodyJobs, &jobs)
	if err != nil {
		logs.Logger.Println("ERROR " + err.Error())
		responses.ERROR(w, respJobs.StatusCode, err)
		return
	}
	// for each job, call exec
	for _, job := range jobs {
		// execute job -> creates manifestWork and deploy it, update UID, State, locker=false -> unlocked
		logs.Logger.Println("Executing Job: " + job.ID.String())
		err := job.Execute()
		if err != nil {
			responses.ERROR(w, http.StatusUnprocessableEntity, err)
			// keep executing
			// return
		} else {
			// HTTP PUT to update UUIDs, State into JOB MANAGER -> updateJob call
			jobBody, err := json.Marshal(job)
			reqState, err := http.NewRequest("PUT", jobmanagerBaseURL+"/jobs/"+job.ID.String(), bytes.NewReader(jobBody))
			reqState.Header.Add("Authorization", r.Header.Get("Authorization"))
			if err != nil {
				responses.ERROR(w, http.StatusUnprocessableEntity, err)
				return
			}
			// do request
			client2 := &http.Client{}
			resp, err := client2.Do(reqState)
			if err != nil {
				responses.ERROR(w, resp.StatusCode, err)
				// retry ??
				// keep executing
				// return
			}
			defer reqState.Body.Close()
		}

	}
	// TODO: update the jobs list with updated jobs
	responses.JSON(w, http.StatusOK, jobs)
}
