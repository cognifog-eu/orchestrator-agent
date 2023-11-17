package controllers

import (
	"icos/server/ocm-description-service/responses"
	"net/http"
)

func (server *Server) HealthCheck(w http.ResponseWriter, r *http.Request) {
	responses.JSON(w, http.StatusOK, "OCM Driver working properly!")
}
