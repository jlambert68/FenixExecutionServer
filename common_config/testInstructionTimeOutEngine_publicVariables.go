package common_config

import "time"

// TimeOutChannelEngineCommandChannelReference
// A reference to  channel for the TestInstructionExecutionEngine
var TimeOutChannelEngineCommandChannelReference *TimeOutChannelEngineType

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
	TimeOutChannelCommandTimeOutTimerTriggered
)

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

// TimeOutChannelCommandStruct
// The struct for the message that are sent over the channel to the TimeOutEngine
type TimeOutChannelCommandStruct struct {
	TimeOutChannelCommand                   TimeOutChannelCommandType
	TimeOutChannelTestInstructionExecutions TimeOutChannelCommandTestInstructionExecutionStruct
	//TimeOutReturnChannelForTimeOutHasOccurred                           *TimeOutReturnChannelForTimeOutHasOccurredType
	//TimeOutReturnChannelForExistsTestInstructionExecutionInTimeOutTimer *TimeOutReturnChannelForExistsTestInstructionExecutionWithinTimeOutTimerType
}
