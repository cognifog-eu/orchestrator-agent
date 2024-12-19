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
	m "etsn/server/ocm-description-service/middlewares"
	"etsn/server/ocm-description-service/utils/logs"
	"net/http"
)

func (s *Server) initializeRoutes(enableJWT bool) {

	middlewares := []func(http.HandlerFunc) http.HandlerFunc{
		m.SetMiddlewareLog,
		m.SetMiddlewareJSON,
	}

	if enableJWT {
		logs.Logger.Println("JWT Middleware is enabled")
		middlewares = append(middlewares, m.JWTValidation)
	} else {
		logs.Logger.Println("JWT Middleware is disabled")
	}

	// Home Route - Applies all middlewares
	s.Router.HandleFunc("/deploy-manager", applyMiddlewares(s.Home, middlewares...)).Methods("GET")

	// Healthcheck Route - Typically no middlewares applied
	s.Router.HandleFunc("/deploy-manager/healthz", s.HealthCheck).Methods("GET")

	// OCM Descriptor Routes - Applies all middlewares
	s.Router.HandleFunc("/deploy-manager/execute", applyMiddlewares(s.PullJobs, middlewares...)).Methods("GET")

	// Get Resource (Status) Route - Applies all middlewares
	s.Router.HandleFunc("/deploy-manager/resource", applyMiddlewares(s.GetResourceStatus, middlewares...)).Methods("GET")

	// Trigger Resource Syncup Route - Applies all middlewares
	s.Router.HandleFunc("/deploy-manager/resource/sync", applyMiddlewares(s.StartSyncUp, middlewares...)).Methods("GET")
}

func applyMiddlewares(handler http.HandlerFunc, middlewares ...func(http.HandlerFunc) http.HandlerFunc) http.HandlerFunc {
	for _, middleware := range middlewares {
		handler = middleware(handler)
	}
	return handler
}
