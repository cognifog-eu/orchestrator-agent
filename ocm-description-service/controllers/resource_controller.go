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
	"encoding/json"
	"errors"
	"etsn/server/ocm-description-service/models"
	"etsn/server/ocm-description-service/responses"
	"etsn/server/ocm-description-service/utils/logs"
	"fmt"
	"net/http"

	workv1 "open-cluster-management.io/api/work/v1"
)

// GetResourceStatus example
//
// @Summary		Get resource status by id
// @Description	get resource status by id
// @Tags			resources
// @Accept			json
// @Produce			json
// @Param			uid				path		string	true	"Resource ID"
// @Param			resource_name	query		string	true	"Resource name"
// @Param			node_target		query		string	true	"Node target"
// @Success		200				{object}	models.Resource
// @Failure		400				{object}	string	"Resource UID is required"
// @Failure		400				{object}	string	"provided UID is different from the retrieved manifest"
// @Failure		422				{object}	string	"Can not parse UID"
// @Failure		404				{object}	string	"Can not find Resource"
// @Router			/deploy-manager/resource [get]
func (server *Server) GetResourceStatus(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	stringUID := query.Get("uid")
	stringTarget := query.Get("node_target")
	stringManifestName := query.Get("resource_name")
	if stringTarget == "" || stringUID == "" || stringManifestName == "" {
		err := errors.New("job's uid, node_target or manifest name are empty")
		fmt.Println("JOB's uid: " + stringUID + " or node_target: " + stringTarget + " or manifest name: " + stringManifestName + " are empty")
		responses.ERROR(w, http.StatusBadRequest, err)
		return
	}

	var manifestWork *workv1.ManifestWork

	err := models.InClusterConfig()
	if err != nil {
		responses.ERROR(w, http.StatusForbidden, err)
	}
	manifestWork, err = models.GetManifestWork(stringTarget, stringManifestName)
	if err != nil {
		logs.Logger.Println("Error during Manifest retrieval...", err)
	}

	conditions := manifestWork.Status.Conditions

	resource := models.Resource{
		ResourceUID:  stringUID,
		ResourceName: stringManifestName,
		Conditions:   conditions,
	}
	if stringUID != string(manifestWork.UID) {
		err := errors.New("provided UID is different from the retrieved manifest")
		responses.ERROR(w, http.StatusBadRequest, err)
		return
	}
	responses.JSON(w, http.StatusOK, resource)
}

// StartSyncUp example
//
// @Summary		Start sync-up
// @Description	start sync-up
// @Tags			resources
// @Accept			json
// @Produce			json
// @Param			Authorization	header		string	true	"Authentication header"
// @Success		200				{string}	string	"Ok"
// @Failure		500				{object}	string "Internal Server Error"
// @Router			/deploy-manager/resource/sync [get]
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
		// logs.Logger.Println("Creating Status Request for Job Manager...")
		// logs.Logger.Println("Resource Status: ")
		// logs.Logger.Printf("%#v", resource)
		resourceBody, err := json.Marshal(resource)
		if err != nil {
			logs.Logger.Println("Could not unmarshall resource...", err)
		}
		reqState, err := http.NewRequest("PUT", jobmanagerBaseURL+"jobmanager/resources/status", bytes.NewReader(resourceBody))
		if err != nil {
			logs.Logger.Println("Error creating resource status update request...", err)
		}
		// logs.Logger.Println("PUT Request to Job Manager being created: ")
		// logs.Logger.Println(reqState.URL)

		reqState.Header.Add("Authorization", r.Header.Get("Authorization"))

		// do request
		client2 := &http.Client{}
		_, err = client2.Do(reqState)
		if err != nil {
			logs.Logger.Println("Error occurred during resource status update request, resource ID: " + resource.ResourceUID)
		}
		defer reqState.Body.Close()
		// logs.Logger.Println("Resource status update request sent, resource ID: " + resource.ResourceUID)
		// logs.Logger.Println("HTTP Response Status:", res.StatusCode, http.StatusText(res.StatusCode))
	}
	responses.JSON(w, http.StatusOK, nil)
}
