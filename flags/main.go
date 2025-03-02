package flags

import (
	"flag"
)

var isDebug = flag.Bool("debug", false, "Enable debug mode")

func init() {
	flag.Parse()
}

func GetIsDebug() *bool {
	return isDebug
}
