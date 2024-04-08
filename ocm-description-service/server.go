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
