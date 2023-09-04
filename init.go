package main

import (
	"FenixExecutionServer/common_config"
	"fmt"
	uuidGenerator "github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"log"
	"os"
	"strconv"
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

	// GCP Project to where the application is deployed
	common_config.GCPProjectId = mustGetenv("ProjectId")

	// Should all SQL-queries be logged before executed
	var tempBoolAsString string
	var tempBool bool
	tempBoolAsString = mustGetenv("LogAllSQLs")
	tempBool, err = strconv.ParseBool(tempBoolAsString)
	if err != nil {
		fmt.Println("Couldn't convert environment variable 'LogAllSQLs' to a boolean, error: ", err)
		os.Exit(0)
	}
	common_config.LogAllSQLs = tempBool

	// Max number of DB-connection from Pool. Not stored because it is re-read when connecting the DB-pool
	_ = mustGetenv("DB_POOL_MAX_CONNECTIONS")

	_, err = strconv.Atoi(mustGetenv("DB_POOL_MAX_CONNECTIONS"))
	if err != nil {
		fmt.Println("Couldn't convert environment variable 'DB_POOL_MAX_CONNECTIONS' to an integer, error: ", err)
		os.Exit(0)

	}

	// Extract environment variable 'WorkerIsUsingPubSubWhenSendingTestInstructionExecutions'
	common_config.WorkerIsUsingPubSubWhenSendingTestInstructionExecutions, err = strconv.ParseBool(mustGetenv("WorkerIsUsingPubSubWhenSendingTestInstructionExecutions"))
	if err != nil {
		fmt.Println("Couldn't convert environment variable 'WorkerIsUsingPubSubWhenSendingTestInstructionExecutions' to an boolean, error: ", err)
		os.Exit(0)
	}

	// Should PubSub be used for sending 'ExecutionsStatus-update-message' to GuiExecutionServer
	common_config.UsePubSubWhenSendingExecutionStatusToGuiExecutionServer, err = strconv.ParseBool(mustGetenv("UsePubSubWhenSendingExecutionStatusToGuiExecutionServer"))
	if err != nil {
		fmt.Println("Couldn't convert environment variable 'UsePubSubWhenSendingExecutionStatusToGuiExecutionServer' to a boolean, error: ", err)
		os.Exit(0)
	}

	// Extract PubSub-Topic for where to send 'ExecutionStatus-messages'
	common_config.ExecutionStatusPubSubTopic = mustGetenv("ExecutionStatusPubSubTopic")

	// Extract local path to Service-Account file
	common_config.LocalServiceAccountPath = mustGetenv("LocalServiceAccountPath")
	// The only way have an OK space is to replace an existing character
	if common_config.LocalServiceAccountPath == "#" {
		common_config.LocalServiceAccountPath = ""
	}

	// Set the environment variable that Google-client-libraries look for
	os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", common_config.LocalServiceAccountPath)

	// Extract PubSub-Topic-DeadLettering-subscription for where to send 'ExecutionStatus-messages' end up when no one reads them
	common_config.ExecutionStatusPubSubDeatLetteringSubscription = mustGetenv("ExecutionStautsPubSubTopic-DeadLettering-Subscription")

}
