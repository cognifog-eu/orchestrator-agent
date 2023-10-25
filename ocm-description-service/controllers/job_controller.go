package controllers

import (
	"bytes"
	"encoding/json"
	"icos/server/ocm-description-service/models"
	"icos/server/ocm-description-service/responses"
	"icos/server/ocm-description-service/utils/logs"
	"io"
	"net/http"
)

const (
	lighthouseBaseURL  = "http://lighthouse.icos-project.eu:8080"
	apiV3              = "/api/v3"
	matchmackerBaseURL = ""
	jobmanagerBaseURL  = "http://192.168.137.201/jobmanager"
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
	logs.Logger.Println("Jobs body: " + string(bodyJobs))
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
			// HTTP PUT to update UUIDs into JOB MANAGER -> updateJob call
			jobBody, err := json.Marshal(job)
			reqState, err := http.NewRequest("PUT", jobmanagerBaseURL+"/jobs/"+job.UUID.String(), bytes.NewReader(jobBody))
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
	responses.JSON(w, http.StatusOK, jobs)
}
