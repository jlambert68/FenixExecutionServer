package common_config

import "time"

// TimeOutChannelEngineCommandChannelReference
// A reference to  channel for the TestInstructionExecutionEngine
//var TimeOutChannelEngineCommandChannelReference *TimeOutChannelEngineType

// TimeOutChannelEngineCommandChannelReferenceSlice
// A slice with references to  channels for the TestInstructionExecutionEngine
// Each position in slice represents one execution track
var TimeOutChannelEngineCommandChannelReferenceSlice []*TimeOutChannelEngineType

// TimeOutChannelEngineType
// The channel type
type TimeOutChannelEngineType chan TimeOutChannelCommandStruct

// TimeOutChannelCommandType
// The type for the constants used within the message sent in the TimeOutChannel
type TimeOutChannelCommandType uint8

const (
	TimeOutChannelCommandAddTestInstructionExecutionToTimeOutTimer TimeOutChannelCommandType = iota
	TimeOutChannelCommandRemoveTestInstructionExecutionFromTimeOutTimerDueToTimeOutFromTimer
	TimeOutChannelCommandRemoveTestInstructionExecutionFromTimeOutTimerDueToExecutionResult
	TimeOutChannelCommandExistsTestInstructionExecutionInTimeOutTimer
	TimeOutChannelCommandHasTestInstructionExecutionAlreadyTimedOut
	TimeOutChannelCommandTimeOutTimerTriggered
	TimeOutChannelCommandAllocateTestInstructionExecutionToTimeOutTimer
	TimeOutChannelCommandTimeUntilNextTimeOutTimerToFires
	TimeOutChannelCommandRemoveAllocationForTestInstructionExecutionToTimeOutTimer
)

var TimeOutChannelCommandsForDebugPrinting []string = []string{
	"TimeOutChannelCommandAddTestInstructionExecutionToTimeOutTimer",
	"TimeOutChannelCommandRemoveTestInstructionExecutionFromTimeOutTimerDueToTimeOutFromTimer",
	"TimeOutChannelCommandRemoveTestInstructionExecutionFromTimeOutTimerDueToExecutionResult",
	"TimeOutChannelCommandExistsTestInstructionExecutionInTimeOutTimer",
	"TimeOutChannelCommandHasTestInstructionExecutionAlreadyTimedOut",
	"TimeOutChannelCommandTimeOutTimerTriggered",
	"TimeOutChannelCommandAllocateTestInstructionExecutionToTimeOutTimer",
	"TimeOutChannelCommandTimeUntilNextTimeOutTimerToFires",
	"TimeOutChannelCommandRemoveAllocationForTestInstructionExecutionToTimeOutTimer",
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

// TimeOutChannelCommandStruct
// The struct for the message that are sent over the channel to the TimeOutEngine
type TimeOutChannelCommandStruct struct {
	TimeOutChannelCommand                               TimeOutChannelCommandType
	TimeOutChannelTestInstructionExecutions             TimeOutChannelCommandTestInstructionExecutionStruct
	TimeOutReturnChannelForTimeOutHasOccurred           *TimeOutResponseChannelForTimeOutHasOccurredType
	TimeOutResponseChannelForDurationUntilTimeOutOccurs *TimeOutResponseChannelForDurationUntilTimeOutOccursType
	SendID                                              string
	//TimeOutReturnChannelForExistsTestInstructionExecutionInTimeOutTimer *TimeOutReturnChannelForExistsTestInstructionExecutionWithinTimeOutTimerType
}

// TimeOutResponseChannelForTimeOutHasOccurredType
// Channel used for response from TimeOutEngine when TimeOut has occurred
type TimeOutResponseChannelForTimeOutHasOccurredType chan TimeOutResponseChannelForTimeOutHasOccurredStruct

// TimeOutResponseChannelForTimeOutHasOccurredStruct
// The struct for the message that are sent over the 'return-channel' when TimeOut has occurred
type TimeOutResponseChannelForTimeOutHasOccurredStruct struct {
	TimeOutWasTriggered bool
}

// TimeOutResponseChannelForDurationUntilTimeOutOccursType
// Channel used for response from TimeOutEngine when extracting duration until TimeOut occurs
type TimeOutResponseChannelForDurationUntilTimeOutOccursType chan TimeOutResponseChannelForDurationUntilTimeOutOccursStruct

// TimeOutResponseChannelForDurationUntilTimeOutOccursStruct
// The struct for the message that are sent over the 'return-channel' when responding duration until TimeOut occurs
type TimeOutResponseChannelForDurationUntilTimeOutOccursStruct struct {
	DurationUntilTimeOutOccurs time.Duration
}
