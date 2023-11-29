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
	m "cognifog/server/ocm-description-service/middlewares"
)

func (s *Server) initializeRoutes() {

	// Home Route
	s.Router.HandleFunc("/deploy-manager", m.SetMiddlewareLog(m.SetMiddlewareJSON(s.Home))).Methods("GET")
	//healthcheck
	s.Router.HandleFunc("/deploy-manager/healthz", s.HealthCheck).Methods("GET")
	//ocm-descriptor routes
	s.Router.HandleFunc("/deploy-manager/execute", m.SetMiddlewareLog(m.SetMiddlewareJSON(m.JWTValidation(s.PullJobs)))).Methods("GET")
}
