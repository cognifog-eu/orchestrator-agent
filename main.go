package main

import (
	_ "etsn/server/docs"
	ocm_description_service "etsn/server/ocm-description-service"
)

//	@title			Swagger Deployment Manager API
//	@version		1.0
//	@description	ICOS Deployment Manager Microservice.
//	@termsOfService	http://swagger.io/terms/

//	@contact.name	API Support
//	@contact.url	http://www.swagger.io/support
//	@contact.email	support@swagger.io

//	@license.name	Apache 2.0
//	@license.url	http://www.apache.org/licenses/LICENSE-2.0.html

//	@host		localhost:8083
//	@BasePath	/

//	@securityDefinitions.basic	OAuth 2.0

// @externalDocs.description	OpenAPI
// @externalDocs.url			https://swagger.io/resources/open-api/
func main() {

	ocm_description_service.Run()

}
