package controllers

import (
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
	jobmanagerBaseURL  = ""
)

func (server *Server) PullJobs(w http.ResponseWriter, r *http.Request) {

	jobs := []models.Job{}
	// get jobs with specific state
	req, err := http.NewRequest("GET", jobmanagerBaseURL+"/jobs", http.NoBody)
	if err != nil {
		responses.ERROR(w, http.StatusUnprocessableEntity, err)
		return
	}

	// do request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		responses.ERROR(w, http.StatusUnprocessableEntity, err)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		responses.ERROR(w, http.StatusUnprocessableEntity, err)
	}

	err = json.Unmarshal(body, &jobs)
	if err != nil {
		responses.ERROR(w, http.StatusUnprocessableEntity, err)
		return
	}
	defer resp.Body.Close()

	// for each job, call exec
	for _, job := range jobs {
		job.Execute()
	}

	responses.JSON(w, resp.StatusCode, jobs)

}
