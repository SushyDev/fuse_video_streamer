package flags

import (
	"flag"
)

var isDebug = flag.Bool("debug", false, "Enable debug mode")
var healthCheck = flag.Bool("health-check", false, "Run health check and exit")

func init() {
	flag.Parse()
}

func GetIsDebug() *bool {
	return isDebug
}

func GetHealthCheck() *bool {
	return healthCheck
}
