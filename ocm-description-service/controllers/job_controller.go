package controllers

import (
	"bytes"
	"encoding/json"
	"icos/server/ocm-description-service/models"
	"icos/server/ocm-description-service/responses"
	"io"
	"net/http"
)

const (
	lighthouseBaseURL  = "http://lighthouse.icos-project.eu:8080"
	apiV3              = "/api/v3"
	matchmackerBaseURL = ""
	jobmanagerBaseURL  = "https://k3s.bull1.ari-imet.eu/jobmanager"
)

func (server *Server) PullJobs(w http.ResponseWriter, r *http.Request) {

	jobs := []models.Job{}
	// get jobs with specific state; CREATED for now
	reqJobs, err := http.NewRequest("GET", jobmanagerBaseURL+"/jobs", http.NoBody)
	if err != nil {
		responses.ERROR(w, http.StatusUnprocessableEntity, err)
		return
	}

	// do request
	client := &http.Client{}
	respJobs, err := client.Do(reqJobs)
	if err != nil {
		responses.ERROR(w, http.StatusUnprocessableEntity, err)
		return
	}

	bodyJobs, err := io.ReadAll(r.Body)
	if err != nil {
		responses.ERROR(w, http.StatusUnprocessableEntity, err)
	}

	err = json.Unmarshal(bodyJobs, &jobs)
	if err != nil {
		responses.ERROR(w, http.StatusUnprocessableEntity, err)
		return
	}
	defer respJobs.Body.Close()

	// for each job, call exec
	for _, job := range jobs {
		// execute job -> creates manifestWork and deploy it, update UID, State, locker=false -> unlocked
		err := job.Execute()
		if err != nil {
			responses.ERROR(w, http.StatusUnprocessableEntity, err)
			return
		}
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
			return
		}
		defer reqState.Body.Close()
	}
	responses.JSON(w, http.StatusOK, jobs)
}
