package logs

import (
	"log"
	"os"
)

var Logger *log.Logger

func init() {

	Logger = log.New(os.Stdout, "[DEPLOY-MANAGER] ", log.Ldate|log.Ltime)

}
