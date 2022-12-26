package testInstructionTimeOutEngine

import (
	"FenixExecutionServer/common_config"
)

// TestInstructionTimeOutEngineObjectStruct
// The struct for the object that hold all functions together within the TimeOutEngine
type TestInstructionTimeOutEngineObjectStruct struct {
}

// TestInstructionExecutionTimeOutEngineObject
// The object that hold all functions together within the TimeOutEngine
var TestInstructionExecutionTimeOutEngineObject TestInstructionTimeOutEngineObjectStruct

// AllocatedTimeOutTimerMapSlice, a slice with the maps that keeps track of all allocated TimeOut-timers which are created before sending TestInstructionExecution to Worker
// Each position in slice represents the execution track
// The reason for keeping track of these are when the execution-response is very fast and Timer had no time to start. Probably only a problem in development-tests
var AllocatedTimeOutTimerMapSlice []map[string]string

// TimeOutChannelEngineCommandChannel
// The channel for the TestInstructionExecutionEngine
//var TimeOutChannelEngineCommandChannel common_config.TimeOutChannelEngineType

// TimeOutChannelEngineCommandChannelSlice
// A Slice with all parallell channel for the TestInstructionExecutionEngine, one per execution track
var TimeOutChannelEngineCommandChannelSlice []common_config.TimeOutChannelEngineType

// TimeOutReturnChannelForExistsTestInstructionExecutionWithinTimeOutTimerType
// Channel used for response from TimeOutEngine when TimeOut has occurred
type TimeOutReturnChannelForExistsTestInstructionExecutionWithinTimeOutTimerType chan TimeOutReturnChannelForExistsTestInstructionExecutionInTimeOutTimerStruct

// TimeOutReturnChannelForExistsTestInstructionExecutionInTimeOutTimerStruct
// The struct for the message that are sent over a 'return-channel' when TimeOutEngine was asked for if TestInstructionExecution exists within TimeOutEngine
type TimeOutReturnChannelForExistsTestInstructionExecutionInTimeOutTimerStruct struct {
	TestInstructionExecution common_config.TimeOutChannelCommandTestInstructionExecutionStruct
}
