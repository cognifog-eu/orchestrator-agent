package server

import (
	"etsn/server/ocm-description-service/controllers"
	"fmt"
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
	// addr := fmt.Sprintf(":%s", os.Getenv(("SERVER_PORT")))
	addr := fmt.Sprintf(":%s", "8083")
	server.Run(addr)

}
