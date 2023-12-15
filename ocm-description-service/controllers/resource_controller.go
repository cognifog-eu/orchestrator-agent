package controllers

import (
	"errors"
	"fmt"
	"icos/server/ocm-description-service/models"
	"icos/server/ocm-description-service/responses"
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
