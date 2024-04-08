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
	"errors"
	"etsn/server/ocm-description-service/models"
	"etsn/server/ocm-description-service/responses"
	"etsn/server/ocm-description-service/utils/logs"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	workv1 "open-cluster-management.io/api/work/v1"
)

// GetResourceStatus example
//
//	@Description	get resource status by id
//	@ID				get-resource-status-by-id
//	@Accept			json
//	@Produce		json
//	@Param			Authorization	header		string	true	"Authentication header"
//	@Param			uid				path		string	true	"Resource ID"
//	@Success		200				{object}	string	"Ok"
//	@Failure		400				{string}	string	"Resource UID is required"
//	@Failure		400				{string}	string	"provided UID is different from the retrieved manifest"
//	@Failure		422				{string}	string	"Can not parse UID"
//	@Failure		404				{string}	string	"Can not find Resource"
//	@Router			/deploy-manager/resource [get]
func (server *Server) GetResourceStatus(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	stringUID := query.Get("uid")
	stringTarget := query.Get("node_target")
	stringManifestName := query.Get("manifest_name")
	if stringTarget == "" || stringUID == "" || stringManifestName == "" {
		err := errors.New("job's uid, node_target or manifest name are empty")
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
	if err != nil {
		logs.Logger.Println("Error during Manifest retrieval...", err)
	}

	conditions := manifestWork.Status.Conditions

	resource := models.Resource{
		ID:           uid,
		ManifestName: stringManifestName,
		NodeTarget:   stringTarget,
		Conditions:   conditions,
		UpdatedAt:    time.Now(),
	}
	if uid.String() != string(manifestWork.UID) {
		err := errors.New("provided UID is different from the retrieved manifest")
		responses.ERROR(w, http.StatusBadRequest, err)
		return
	}
	responses.JSON(w, http.StatusOK, resource)
}

// StartSyncUp example
//
//	@Description	start sync-up
//	@ID				start-sync-up
//	@Accept			json
//	@Produce		json
//	@Param			Authorization	header		string	true	"Authentication header"
//	@Success		200				{string}	string	"Ok"
//	@Router			/deploy-manager/resource/sync [get]
func (server *Server) StartSyncUp(w http.ResponseWriter, r *http.Request) {
	var resources []models.Resource
	err := models.InClusterConfig()
	if err != nil {
		logs.Logger.Println("Kubeconfig error occured", err)
	}
	resources, err = models.ResourceSync()
	if err != nil {
		logs.Logger.Println("Error during resource sync...", err)
	}
	for _, resource := range resources {
		// HTTP PUT to update UUIDs, State into JOB MANAGER -> updateJob call
		logs.Logger.Println("Creating Status Request for Job Manager...")
		logs.Logger.Println("Resource Status: ")
		logs.Logger.Printf("%#v", resource)
		resourceBody, err := json.Marshal(resource)
		if err != nil {
			logs.Logger.Println("Could not unmarshall resource...", err)
		}
		reqState, err := http.NewRequest("PUT", jobmanagerBaseURL+"/jobmanager/resources/status/"+resource.ID.String(), bytes.NewReader(resourceBody))
		if err != nil {
			logs.Logger.Println("Error creating resource status update request...", err)
		}
		logs.Logger.Println("PUT Request to Job Manager being created: ")
		logs.Logger.Println(reqState.URL)
		query := reqState.URL.Query()
		query.Add("uuid", resource.ID.String())
		reqState.Header.Add("Authorization", r.Header.Get("Authorization"))

		// do request
		client2 := &http.Client{}
		res, err := client2.Do(reqState)
		if err != nil {
			logs.Logger.Println("Error occurred during resource status update request, resource ID: " + resource.ID.String())
			// keep executing
		}
		defer reqState.Body.Close()
		logs.Logger.Println("Resource status update request sent, resource ID: " + resource.ID.String())
		logs.Logger.Println("HTTP Response Status:", res.StatusCode, http.StatusText(res.StatusCode))
	}
	responses.JSON(w, http.StatusOK, nil)
}
