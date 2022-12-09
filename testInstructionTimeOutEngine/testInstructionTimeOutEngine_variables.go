package testInstructionTimeOutEngine

import "time"

// TestInstructionTimeOutEngineObjectStruct
// The struct for the object that hold all functions together within the TimeOutEngine
type TestInstructionTimeOutEngineObjectStruct struct {
}

// TestInstructionExecutionTimeOutEngineObject
// The object that hold all functions together within the TimeOutEngine
var TestInstructionExecutionTimeOutEngineObject TestInstructionTimeOutEngineObjectStruct


// timeOutMap,
var timeOutMap map[]


// TimeOutChannelEngineCommandChannel
// The channel for the TestInstructionExecutionEngine
var TimeOutChannelEngineCommandChannel TimeOutChannelEngineType

// TimeOutChannelEngineType
// The channel type
type TimeOutChannelEngineType chan TimeOutChannelCommandStruct

// TimeOutChannelCommandType
// The type for the constants used within the message sent in the TimeOutChannel
type TimeOutChannelCommandType uint8

const (
	TimeOutChannelCommandAddTestInstructionExecutionToTimeOutTimer TimeOutChannelCommandType = iota
	TimeOutChannelCommandRemoveTestInstructionExecutionFromTimeOutTimer
	TimeOutChannelCommandExistsTestInstructionExecutionInTimeOutTimer
)

// TimeOutChannelCommandStruct
// The struct for the message that are sent over the channel to the TimeOutEngine
type TimeOutChannelCommandStruct struct {
	TimeOutChannelCommand                                               TimeOutChannelCommandType
	TimeOutChannelTestInstructionExecutions                             TimeOutChannelCommandTestInstructionExecutionStruct
	TimeOutReturnChannelForTimeOutHasOccurred                           *TimeOutReturnChannelForTimeOutHasOccurredType
	TimeOutReturnChannelForExistsTestInstructionExecutionInTimeOutTimer *TimeOutReturnChannelForExistsTestInstructionExecutionWithinTimeOutTimerType
}

// TimeOutChannelCommandTestInstructionExecutionStruct
// Hold one TestInstructionExecution that to handled by TimeOutEngine
type TimeOutChannelCommandTestInstructionExecutionStruct struct {
	TestCaseExecutionUuid                   string
	TestCaseExecutionVersion                int32
	TestInstructionExecutionUuid            string
	TestInstructionExecutionVersion         int32
	TestInstructionExecutionCanBeReExecuted bool
	TimeOutTime                             time.Time
}

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
	TestInstructionExecution TimeOutChannelCommandTestInstructionExecutionStruct
}
