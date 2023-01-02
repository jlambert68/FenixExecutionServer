package common_config

import "time"

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

// MinutesToShutDownWithOutAnyGrpcTraffic
// The number of minutes without any incoming gPRC-traffic before application is shut down
var MinutesToShutDownWithOutAnyGrpcTraffic time.Duration = 15 * time.Minute

// MaxMinutesLeftUntilNextTimeOutTimer
// Number of minutes that the application can wait for a TimeOutTimer, after waited 'MinutesToShutDownWithOutAnyGrpcTraffic'
var MaxMinutesLeftUntilNextTimeOutTimer = 2 * time.Minute
