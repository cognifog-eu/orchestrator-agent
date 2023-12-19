package controllers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"icos/server/ocm-description-service/models"
	"icos/server/ocm-description-service/responses"
	"icos/server/ocm-description-service/utils/logs"
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
	logs.Logger.Println("Requesting Jobs...")
	reqJobs, err := http.NewRequest("GET", jobmanagerBaseURL+"/jobmanager/jobs/executable", http.NoBody)
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
		job, err := models.Execute(&job)
		if err != nil {
			// responses.ERROR(w, http.StatusUnprocessableEntity, err)
			logs.Logger.Println("Error occurred during Job execution...")
			// keep executing
		} else {
			// HTTP PUT to update UUIDs, State into JOB MANAGER -> updateJob call
			logs.Logger.Println("Job executed, sending details to Job Manager...")
			jobBody, err := json.Marshal(job)
			reqState, err := http.NewRequest("PUT", jobmanagerBaseURL+"jobmanager/jobs/"+job.ID.String(), bytes.NewReader(jobBody))
			query := reqState.URL.Query()
			query.Add("uuid", job.UUID.String())
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
