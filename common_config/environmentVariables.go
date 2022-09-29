package common_config

// ***********************************************************************************************************
// The following variables receives their values from environment variables

// Where is the Worker running
var ExecutionLocationForWorker ExecutionLocationTypeType

// Where is Fenix Execution Server running
var ExecutionLocationForFenixExecutionServer ExecutionLocationTypeType

// Definitions for where client and Fenix Server is running
type ExecutionLocationTypeType int

// Constants used for where stuff is running
const (
	LocalhostNoDocker ExecutionLocationTypeType = iota
	LocalhostDocker
	GCP
)

// Execution Worker Port to use, will have its value from Environment variables at startup
var FenixExecutionWorkerServerPort int
