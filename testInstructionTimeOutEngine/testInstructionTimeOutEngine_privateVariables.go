package testInstructionTimeOutEngine

import (
	"FenixExecutionServer/common_config"
	"time"
)

// Variable holding a MapKey to 'timeOutMap' for the TestInstructionExecution that has the closest upcoming TimeOut
var nextUpcomingObjectMapKeyWithTimeOut string

// Timer used to keep track of when next TestInstructionExecutions TimeOut-time occurs
var cancellableTimer *common_config.CancellableTimerStruct

// The time that is added to the TimeOut-timer running on ExecutionServer
const extractTimerMarginalBeforeTimeOut time.Duration = time.Second * 60

// timeOutMap, the map that keeps track of all TestInstructionExecutions
var timeOutMap map[string]*timeOutMapStruct // map[TestInstructionExecutionKey]*timeOutMapStruct

// The struct holding one map-object with references to previous and next objects regarding their TimeOut-time
type timeOutMapStruct struct {
	currentTimeOutMapKey               string
	previousTimeOutMapKey              string
	nextTimeOutMapKey                  string
	currentTimeOutChannelCommandObject *common_config.TimeOutChannelCommandStruct
}

// timeOutChannelSize
// The size of the channel
const timeOutChannelSize = 100

// timeOutChannelWarningLevel
// The size of warning level for the channel
const timeOutChannelWarningLevel = 90