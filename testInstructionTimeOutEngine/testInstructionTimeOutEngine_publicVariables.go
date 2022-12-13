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

// TimeOutChannelEngineCommandChannel
// The channel for the TestInstructionExecutionEngine
var TimeOutChannelEngineCommandChannel common_config.TimeOutChannelEngineType

// TimeOutReturnChannelForTimeOutHasOccurredType
// Channel used for response from TimeOutEngine when TimeOut has occurred
type TimeOutReturnChannelForTimeOutHasOccurredType chan TimeOutReturnChannelForTimeOutHasOccurredStruct

// TimeOutReturnChannelForTimeOutHasOccurredStruct
// The struct for the message that are sent over the 'return-channel' when TimeOut has occurred
type TimeOutReturnChannelForTimeOutHasOccurredStruct struct {
	TimeOutWasTriggered bool
}

// TimeOutReturnChannelForExistsTestInstructionExecutionWithinTimeOutTimerType
// Channel used for response from TimeOutEngine when TimeOut has occurred
type TimeOutReturnChannelForExistsTestInstructionExecutionWithinTimeOutTimerType chan TimeOutReturnChannelForExistsTestInstructionExecutionInTimeOutTimerStruct

// TimeOutReturnChannelForExistsTestInstructionExecutionInTimeOutTimerStruct
// The struct for the message that are sent over a 'return-channel' when TimeOutEngine was asked for if TestInstructionExecution exists within TimeOutEngine
type TimeOutReturnChannelForExistsTestInstructionExecutionInTimeOutTimerStruct struct {
	TestInstructionExecution common_config.TimeOutChannelCommandTestInstructionExecutionStruct
}
