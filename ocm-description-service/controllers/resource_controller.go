package controllers

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"icos/server/ocm-description-service/models"
	"icos/server/ocm-description-service/responses"
	"icos/server/ocm-description-service/utils/logs"
	"net/http"
	"time"

	"github.com/google/uuid"
	workv1 "open-cluster-management.io/api/work/v1"
)

func (server *Server) GetResourceStatus(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	stringUID := query.Get("uid")
	stringTarget := query.Get("node_target")
	stringManifestName := query.Get("manifest_name")
	if stringTarget == "" || stringUID == "" || stringManifestName == "" {
		err := errors.New("Job's uid, node_target or manifest name are empty")
		fmt.Println("JOB's uid: " + stringUID + " or node_target: " + stringTarget + " or manifest name: " + stringManifestName + " are empty")
		responses.ERROR(w, http.StatusBadRequest, err)
		return
	}

	var manifestWork *workv1.ManifestWork
	uid, err := uuid.Parse(stringUID)
	if err != nil {
		responses.ERROR(w, http.StatusUnprocessableEntity, err)
	}
	err = models.InClusterConfig()
	if err != nil {
		responses.ERROR(w, http.StatusForbidden, err)
	}
	manifestWork, err = models.GetManifestWork(stringTarget, stringManifestName)
	status := models.Status{
		Conditions: manifestWork.Status.Conditions,
	}
	resource := models.Resource{
		ID:           uid,
		ManifestName: stringManifestName,
		NodeTarget:   stringTarget,
		Status:       status,
		UpdatedAt:    time.Now(),
	}
	if uid.String() != string(manifestWork.UID) {
		err := errors.New("provided UID is different from the retrieved manifest")
		responses.ERROR(w, http.StatusBadRequest, err)
		return
	}
	responses.JSON(w, http.StatusOK, resource)
}

func (server *Server) StartSyncUp(w http.ResponseWriter, r *http.Request) {
	var resources []models.Resource
	err := models.InClusterConfig()
	if err != nil {
		logs.Logger.Println("Kubeconfig error occured", err)
	}
	resources, err = models.ResourceSync()
	if err != nil {
		responses.ERROR(w, http.StatusInternalServerError, err)
	}
	for _, resource := range resources {
		// update its status into JM (PUT)
		// HTTP PUT to update UUIDs, State into JOB MANAGER -> updateJob call
		logs.Logger.Println("Sending status details to Job Manager...")
		resourceBody, err := json.Marshal(resource)
		if err != nil {
			logs.Logger.Println("Could not unmarshall resource...", err)
		}
		reqState, err := http.NewRequest("PUT", jobmanagerBaseURL+"jobmanager/resources/status/"+resource.ID.String(), bytes.NewReader(resourceBody))
		if err != nil {
			responses.ERROR(w, http.StatusUnprocessableEntity, err)
			return
		}
		query := reqState.URL.Query()
		query.Add("uuid", resource.ID.String())
		reqState.Header.Add("Authorization", r.Header.Get("Authorization"))

		// do request
		client2 := &http.Client{}
		resp, err := client2.Do(reqState)
		if err != nil {
			logs.Logger.Println("Error occurred during Job details notification...")
			// responses.ERROR(w, resp.StatusCode, err)
			// keep executing
		}
		fmt.Println("Update Job Request " + logs.FormatRequest(reqState))
		logs.Logger.Println("Update Job Response " + resp.Status)
		defer reqState.Body.Close()
	}
	responses.JSON(w, http.StatusOK, nil)
}
