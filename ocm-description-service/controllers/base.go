package controllers

import (
	"context"
	"flag"
	"icos/server/ocm-description-service/utils/logs"
	"net/http"
	"os"
	"os/signal"
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

	server.initializeRoutes()
}

func (server *Server) Run(addr string) {
	logs.Logger.Println("Listening to port " + addr + " ...")
	handler := cors.AllowAll().Handler(server.Router)

	stop := make(chan os.Signal)
	signal.Notify(stop, os.Interrupt)

	go func() {
		// init server
		if err := http.ListenAndServe(addr, handler); err != nil {
			if err != http.ErrServerClosed {
				logs.Logger.Fatal(err)
			}
		}
	}()

	<-stop

	// after stopping server
	logs.Logger.Println("Closing connections ...")

	var shutdownTimeout = flag.Duration("shutdown-timeout", 10*time.Second, "shutdown timeout (5s,5m,5h) before connections are cancelled")
	_, cancel := context.WithTimeout(context.Background(), *shutdownTimeout)
	defer cancel()
}
