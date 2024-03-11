package testInstructionTimeOutEngine

import (
	"FenixExecutionServer/common_config"
	"time"
)

// Variable holding a MapKey to 'timeOutMap' for the TestInstructionExecution that has the closest upcoming TimeOut
// var nextUpcomingObjectMapKeyWithTimeOut string

// A slice have Variables holding MapKeys to 'timeOutMap' for the TestInstructionExecution that has the closest upcoming TimeOut
// Each position in slice represents the execution track
var nextUpcomingObjectMapKeyWithTimeOutSlice []string

// Slice with Timers used to keep track of when next TestInstructionExecutions TimeOut-time occurs, per Execution Track
var cancellableTimerSlice []*common_config.CancellableTimerStruct

// The time that is added to the TimeOut-timer running on ExecutionServer
const extractTimerMarginalBeforeTimeOut time.Duration = time.Second * 60 * 2

// The time that is removed from when TestInstructionExecution finished quite near the end of TimeOut
const extractTimerMarginalBeforeTimeOut_25percent time.Duration = time.Second * 180
const extractTimerMarginalBeforeTimeOut_15Seconds time.Duration = time.Second * 15

// allocatedTimeOutTimerMap, the map that keeps track of all allocated TimeOut-timers which are created before sending TestInstructionExecution to Worker
// The reason for keeping track of these are when the execution-response is very fast and Timer had no time to start. Probably only a problem in development-tests
//var allocatedTimeOutTimerMap map[string]string // map[TestInstructionExecutionKey]TestInstructionExecutionKey

// timeOutMap, the map that keeps track of all TestInstructionExecutions with ongoing TimeOut-timers
// var timeOutMap map[string]*timeOutMapStruct // map[TestInstructionExecutionKey]*timeOutMapStruct

// timeOutMapSlice, a slice with the maps that keeps track of all TestInstructionExecutions with ongoing TimeOut-timers
// Each position in slice represents the execution track
var timeOutMapSlice []*map[string]*timeOutMapStruct

// timedOutMap, the map that keeps track of all TestInstructionExecutions that has TimedOut
// var timedOutMap map[string]*timeOutMapStruct // map[TestInstructionExecutionKey]*timeOutMapStruct

// timedOutMapSlice,  a slice with the maps that keeps track of all TestInstructionExecutions that has TimedOut
var timedOutMapSlice []*map[string]*timeOutMapStruct

// The struct holding one map-object with references to previous and next objects regarding their TimeOut-time
type timeOutMapStruct struct {
	currentTimeOutMapKey               string
	previousTimeOutMapKey              string
	nextTimeOutMapKey                  string
	currentTimeOutChannelCommandObject *common_config.TimeOutChannelCommandStruct
	cancellableTimer                   *common_config.CancellableTimerStruct
}

// timeOutChannelSize
// The size of the channel
const timeOutChannelSize = 100

// timeOutChannelWarningLevel
// The size of warning level for the channel
const timeOutChannelWarningLevel = 90
