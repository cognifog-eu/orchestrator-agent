package controllers

import (
	m "icos/server/ocm-description-service/middlewares"
)

func (s *Server) initializeRoutes() {

	// Home Route
	s.Router.HandleFunc("/ochrestration/description/", m.SetMiddlewareLog(m.SetMiddlewareJSON(s.Home))).Methods("GET")

	//ocm-descriptor routes
	// s.Router.HandleFunc("/deploy-manager/description/:id", m.SetMiddlewareLog(m.SetMiddlewareJSON(m.JWTValidation(s.GetJobByUUID)))).Methods("GET")
	// s.Router.HandleFunc("/ochrestration/description/:id", m.SetMiddlewareLog(m.SetMiddlewareJSON(m.JWTValidation(s.GetJobByUUID)))).Methods("POST")
}
