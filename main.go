package main

import (
	"FenixExecutionServer/common_config"
	uuidGenerator "github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"net/http"
	"strconv"

	//"flag"
	"fmt"
	"log"
	_ "net/http/pprof"
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
	// Profiling
	go func() {
		fmt.Println(http.ListenAndServe("localhost:6060", nil))
	}()
	/*
		config := metrics.DefaultConfig
		config.Username = "jlambert"
		config.Password = "jonasaradmin"
		config.Database = "stats"

		err := metrics.RunCollector(config)
		if err != nil {
			log.Fatalf(err.Error())
		}
	*/

	fenixExecutionServerMain()
}

func init() {

	// Create Unique Uuid for run time instance
	common_config.ApplicationRuntimeUuid = uuidGenerator.New().String()
	fmt.Println("ApplicationRuntimeUuid: " + common_config.ApplicationRuntimeUuid)

	var err error

	// Get Environment variable to tell how/were this worker is  running
	var executionLocationForWorker = mustGetenv("ExecutionLocationForWorker")

	switch executionLocationForWorker {
	case "LOCALHOST_NODOCKER":
		common_config.ExecutionLocationForWorker = common_config.LocalhostNoDocker

	case "LOCALHOST_DOCKER":
		common_config.ExecutionLocationForWorker = common_config.LocalhostDocker

	case "GCP":
		common_config.ExecutionLocationForWorker = common_config.GCP

	default:
		fmt.Println("Unknown Execution location for Worker: " + executionLocationForWorker + ". Expected one of the following: 'LOCALHOST_NODOCKER', 'LOCALHOST_DOCKER', 'GCP'")
		os.Exit(0)

	}

	// Get Environment variable to tell were Fenix Execution Server is running
	var executionLocationForExecutionServer = mustGetenv("ExecutionLocationForFenixTestExecutionServer")

	switch executionLocationForExecutionServer {
	case "LOCALHOST_NODOCKER":
		common_config.ExecutionLocationForFenixExecutionServer = common_config.LocalhostNoDocker

	case "LOCALHOST_DOCKER":
		common_config.ExecutionLocationForFenixExecutionServer = common_config.LocalhostDocker

	case "GCP":
		common_config.ExecutionLocationForFenixExecutionServer = common_config.GCP

	default:
		fmt.Println("Unknown Execution location for Fenix Execution Server: " + executionLocationForWorker + ". Expected one of the following: 'LOCALHOST_NODOCKER', 'LOCALHOST_DOCKER', 'GCP'")
		os.Exit(0)

	}

	// Address to Fenix Execution Server
	//common_config.FenixExecutionWorkerServerAddress = mustGetenv("FenixExecutionServerAddress")

	// Port for Fenix Execution Server
	common_config.FenixExecutionExecutionServerPort, err = strconv.Atoi(mustGetenv("FenixExecutionExecutionServerPort"))
	if err != nil {
		fmt.Println("Couldn't convert environment variable 'FenixExecutionWorkerServerPort' to an integer, error: ", err)
		os.Exit(0)

	}

	// Address to Worker, when not run in cloud because then the address is coming from DB
	common_config.FenixExecutionWorkerAddress = mustGetenv("FenixExecutionWorkerServerAddress")

	// Port for Fenix Execution Worker Server
	common_config.FenixExecutionWorkerServerPort, err = strconv.Atoi(mustGetenv("FenixExecutionWorkerServerPort"))
	if err != nil {
		fmt.Println("Couldn't convert environment variable 'FenixExecutionWorkerServerPort' to an integer, error: ", err)
		os.Exit(0)
	}

	// Extract Debug level
	var loggingLevel = mustGetenv("LoggingLevel")

	switch loggingLevel {

	case "DebugLevel":
		common_config.LoggingLevel = logrus.DebugLevel

	case "InfoLevel":
		common_config.LoggingLevel = logrus.InfoLevel

	default:
		fmt.Println("Unknown LoggingLevel '" + loggingLevel + "'. Expected one of the following: 'DebugLevel', 'InfoLevel'")
		os.Exit(0)

	}
}
