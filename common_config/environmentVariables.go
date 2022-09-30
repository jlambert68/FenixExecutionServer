package common_config

// ***********************************************************************************************************
// The following variables receives their values from environment variables

// ExecutionLocationForWorker
// Where is the Worker running
var ExecutionLocationForWorker ExecutionLocationTypeType

// ExecutionLocationForFenixExecutionServer
// Where is Fenix Execution Server running
var ExecutionLocationForFenixExecutionServer ExecutionLocationTypeType

// ExecutionLocationTypeType
// Definitions for where client and Fenix Server is running
type ExecutionLocationTypeType int

// Constants used for where stuff is running
const (
	LocalhostNoDocker ExecutionLocationTypeType = iota
	LocalhostDocker
	GCP
)

// FenixExecutionWorkerServerPort
// Execution Worker Port to use, will have its value from Environment variables at startup
var FenixExecutionWorkerServerPort int

// Address to use when not run locally, not on GCP/Cloud which gets its address from DB
var FenixExecutionWorkerAddress string

// FenixExecutionExecutionServerPort
// Execution Server Port to use, will have its value from Environment variables at startup
var FenixExecutionExecutionServerPort int
