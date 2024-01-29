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
var MinutesToShutDownWithOutAnyGrpcTraffic time.Duration = 10 * time.Minute

// MaxMinutesLeftUntilNextTimeOutTimer
// Number of minutes that the application can wait for a TimeOutTimer, after waited 'MinutesToShutDownWithOutAnyGrpcTraffic'
// If TimeOutTimer > MaxMinutesLeftUntilNextTimeOutTimer then application shuts down and saves the timer value in FireStore-DB
// A Crone Job in GCP autostarts the application 'NumberOfMinutesBeforeNextTimeOutTimerToStart' before TimeOut is expected
var MaxMinutesLeftUntilNextTimeOutTimer = 5 * time.Minute

// NumberOfMinutesBeforeNextTimeOutTimerToStart
// Number of minutes that the application will start up before next TimeOut-Timer will fire
var NumberOfMinutesBeforeNextTimeOutTimerToStart = -2 * time.Minute

// GCPProjectId is the GCP project where the application will be deployed into
var GCPProjectId string

// WorkerIsUsingPubSubWhenSendingTestInstructionExecutions decides which gRPC-call to worker that will be used
var WorkerIsUsingPubSubWhenSendingTestInstructionExecutions bool

var LogAllSQLs bool

// UsePubSubWhenSendingExecutionStatusToGuiExecutionServer
// Should PubSub be used for sending 'ExecutionsStatus-update-message' to GuiExecutionServer
var UsePubSubWhenSendingExecutionStatusToGuiExecutionServer bool

// ExecutionStatusPubSubTopic
// PubSub-Topic for where to send 'ExecutionStatus-messages'
var ExecutionStatusPubSubTopic string

// LocalServiceAccountPath
// Local path to Service-Account file
var LocalServiceAccountPath string

// ExecutionStatusPubSubDeadLetteringSubscription
// The PubSub-DeadLettering-subscription where all messages that no one reads end up
var ExecutionStatusPubSubDeadLetteringSubscription string

// SleepTimeInSecondsBeforeClaimingTestInstructionExecutionFromDatabase
// The number of seconds before this ExecutionServer tries to claim a TestInstructionExecution from database
var SleepTimeInSecondsBeforeClaimingTestInstructionExecutionFromDatabase int
