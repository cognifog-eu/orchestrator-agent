package controllers

import (
	m "etsn/server/ocm-description-service/middlewares"
)

func (s *Server) initializeRoutes() {

	// Home Route
	s.Router.HandleFunc("/deploy-manager", m.SetMiddlewareLog(m.SetMiddlewareJSON(s.Home))).Methods("GET")
	//healthcheck
	s.Router.HandleFunc("/deploy-manager/healthz", s.HealthCheck).Methods("GET")
	//ocm-descriptor routes
	s.Router.HandleFunc("/deploy-manager/execute", m.SetMiddlewareLog(m.SetMiddlewareJSON(m.JWTValidation(s.PullJobs)))).Methods("GET")
	// get resource (status)
	s.Router.HandleFunc("/deploy-manager/resource", m.SetMiddlewareLog(m.SetMiddlewareJSON(m.JWTValidation(s.GetResourceStatus)))).Methods("GET")
	// trigger resource syncup
	s.Router.HandleFunc("/deploy-manager/resource/sync", m.SetMiddlewareLog(m.SetMiddlewareJSON(m.JWTValidation(s.StartSyncUp)))).Methods("GET")
}
