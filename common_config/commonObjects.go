package common_config

// ***********************************************************************************************************
// The following variables receives their values from environment variables

// Where is the client running
var ExecutionLocationForClient ExecutionLocationTypeType

// Where is the Fenix TestDataSync server running
// LocationForFenixExecutionServer
var ExecutionLocationForFenixExecutionServer ExecutionLocationTypeType

// Definitions for where client and Fenix Server is running
type ExecutionLocationTypeType int

// Constants used for where stuff is running
const (
	LocalhostNoDocker ExecutionLocationTypeType = iota
	LocalhostDocker
	GCP
)

// FenixGuiBuilderServer
var LocationForFenixGuiBuilderServerTypeMapping = map[ExecutionLocationTypeType]string{
	LocalhostNoDocker: "LOCALHOST_NODOCKER",
	LocalhostDocker:   "LOCALHOST_DOCKER",
	GCP:               "GCP",
}

// Address to Fenix TestData Server & Client, will have their values from Environment variables at startup
var (
	FenixTestDataSyncServerAddress string // TODO remove, but is referenced by code that is not removed yet
	FenixTestDataSyncServerPort    int    // TODO remove,
	FenixGuiServerAddress          string
	FenixExecutionGuiServerPort    int
)

var highestFenixProtoFileVersion int32 = -1
var highestClientProtoFileVersion int32 = -1

// ***********************************************************************************************************
