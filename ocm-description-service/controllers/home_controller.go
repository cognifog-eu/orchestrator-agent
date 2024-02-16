package controllers

import (
	"etsn/server/ocm-description-service/responses"
	"net/http"
)

func (server *Server) Home(w http.ResponseWriter, r *http.Request) {
	responses.JSON(w, http.StatusOK, "Welcome To ICOS OCM Description Service")

}
