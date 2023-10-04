package controllers

import (
	"icos/server/ocm-description-service/models"
	"icos/server/ocm-description-service/responses"
	"net/http"
)

const (
	lighthouseBaseURL  = "http://lighthouse.icos-project.eu:8080"
	apiV3              = "/api/v3"
	matchmackerBaseURL = ""
	jobmanagerBaseURL  = ""
)

func (server *Server) PullJobs(w http.ResponseWriter, r *http.Request) {

	jobs := models.Jobs{}
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
	defer resp.Body.Close()

	responses.JSON(w, resp.StatusCode, jobs)

}
