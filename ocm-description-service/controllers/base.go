package controllers

import (
	"fmt"
	"net/http"

	"log"

	"github.com/gorilla/mux"
)

type Server struct {
	Router *mux.Router
}

func (server *Server) Init() {
	server.Router = mux.NewRouter()
	server.initializeRoutes()
}

func (server *Server) Run(addr string) {
	fmt.Printf("Listening to port %s", addr)
	log.Fatal(http.ListenAndServe(addr, server.Router))
}
