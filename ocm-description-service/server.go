package server

import (
	"fmt"
	"icos/server/ocm-description-service/controllers"
	"os"
)

var server = controllers.Server{}

func Init() {
	// loads values from .env into the system
	// if err := godotenv.Load(); err != nil {
	// 	log.Print("sad .env file found")
	// }
}

func Run() {
	server.Init()
	addr := fmt.Sprintf(":%s", os.Getenv(("SERVER_PORT")))
	server.Run(addr)

}
