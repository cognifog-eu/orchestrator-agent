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
	"context"
	"etsn/server/ocm-description-service/utils/logs"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
	httpSwagger "github.com/swaggo/http-swagger/v2"
)

type Server struct {
	Router *mux.Router
}

func (server *Server) Init() {
	server.Router = mux.NewRouter()

	// swagger
	server.Router.PathPrefix("/deploy-manager/swagger/").Handler(httpSwagger.Handler(
		httpSwagger.URL("doc.json"), //The url pointing to API definition
		httpSwagger.DeepLinking(true),
		httpSwagger.DocExpansion("none"),
		httpSwagger.DomID("swagger-ui"),
	)).Methods(http.MethodGet)

	// enable JWT middleware
	enableJWT := os.Getenv("ENVIRONMENT") != "development"

	server.initializeRoutes(enableJWT)
}

func (server *Server) Run(addr string) {
	logs.Logger.Println("Listening on port " + addr + " ...")
	handler := cors.AllowAll().Handler(server.Router)

	httpServer := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	stop := make(chan os.Signal, 1)

	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logs.Logger.Fatalf("Could not listen on %s: %v\n", addr, err)
		}
	}()

	logs.Logger.Println("Server is ready to handle requests")

	<-stop

	logs.Logger.Println("Shutdown signal received")

	shutdownTimeout := flag.Duration("shutdown-timeout", 10*time.Second, "shutdown timeout (5s,5m,5h) before connections are cancelled")
	flag.Parse() // Ensure flags are parsed

	ctx, cancel := context.WithTimeout(context.Background(), *shutdownTimeout)
	defer cancel()

	logs.Logger.Println("Shutting down server gracefully...")
	if err := httpServer.Shutdown(ctx); err != nil {
		logs.Logger.Fatalf("Server Shutdown Failed:%+v", err)
	}

	logs.Logger.Println("Server gracefully stopped")
}
