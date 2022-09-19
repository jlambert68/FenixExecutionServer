package main

import (
	"FenixExecutionServer/common_config"
	"strconv"

	//"flag"
	"fmt"
	"log"
	"os"
)

// mustGetEnv is a helper function for getting environment variables.
// Displays a warning if the environment variable is not set.
func mustGetenv(k string) string {
	v := os.Getenv(k)
	if v == "" {
		log.Fatalf("Warning: %s environment variable not set.\n", k)
	}
	return v
}

func main() {
	//time.Sleep(15 * time.Second)
	fenixExecutionServerMain()
}

func init() {
	//executionLocation := flag.String("startupType", "0", "The application should be started with one of the following: LOCALHOST_NODOCKER, LOCALHOST_DOCKER, GCP")
	//flag.Parse()

	var err error

	// Get Environment variable to tell how this program was started
	var executionLocation = mustGetenv("ExecutionLocation")

	switch executionLocation {
	case "LOCALHOST_NODOCKER":
		common_config.ExecutionLocationForClient = common_config.LocalhostNoDocker

	case "LOCALHOST_DOCKER":
		common_config.ExecutionLocationForClient = common_config.LocalhostDocker

	case "GCP":
		common_config.ExecutionLocationForClient = common_config.GCP

	default:
		fmt.Println("Unknown Execution location for FenixGuiServer: " + executionLocation + ". Expected one of the following: LOCALHOST_NODOCKER, LOCALHOST_DOCKER, GCP")
		os.Exit(0)

	}

	// Address to GuiBuilderServer
	common_config.FenixGuiServerAddress = mustGetenv("FenixGuiBuilderServerAddress")

	// Port for GuiBuilderServer
	common_config.FenixExecutionGuiServerPort, err = strconv.Atoi(mustGetenv("FenixGuiBuilderServerPort"))
	if err != nil {
		fmt.Println("Couldn't convert environment variable 'FenixGuiBuilderServerPort' to an integer, error: ", err)
		os.Exit(0)

	}

}
