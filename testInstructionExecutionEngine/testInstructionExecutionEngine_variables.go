package testInstructionExecutionEngine

import (
	"github.com/sirupsen/logrus"
)

// TestInstructionExecutionEngineStruct
// The struct for the object that hold all functions together within the executionEngine
type TestInstructionExecutionEngineStruct struct {
	logger                  *logrus.Logger
	CommandChannelReference *ExecutionEngineChannelType
}

// TestInstructionExecutionEngineObject
// The object that hold all functions together within the executionEngine
var TestInstructionExecutionEngineObject TestInstructionExecutionEngineStruct

// ExecutionEngineChannelSize
// The size of the channel
const ExecutionEngineChannelSize = 1000

// ExecutionEngineChannelWarningLevel
// The size of warning level for the channel
const ExecutionEngineChannelWarningLevel = 700

// ExecutionEngineCommandChannel
// The channel for the TestInstructionExecutionEngine
var ExecutionEngineCommandChannel ExecutionEngineChannelType

// ExecutionEngineChannelType
// The channel type
type ExecutionEngineChannelType chan ChannelCommandStruct

// ChannelCommandType
// The type for the constants used within the message sent in the channel
type ChannelCommandType uint8

const (
	ChannelCommandCheckForTestInstructionExecutionWaitingOnQueue ChannelCommandType = iota
	ChannelCommandCheckForTestInstructionExecutionsWaitingToBeSentToWorker
	ChannelCommandCheckOngoingTestInstructionExecutions
	ChannelCommandUpdateExecutionStatusOnTestCaseExecutionExecutions
	ChannelCommandLookForZombieTestInstructionExecutionsInUnderExecution
	ChannelCommandProcessTestCaseExecutionsOnExecutionQueue
	ChannelCommandSendZombieTestCaseExecutionThatAreStuckOnExecutionQueue
	ChannelCommandLookForZombieTestInstructionExecutionsOnExecutionQueue
	ChannelCommandLookForZombieTestInstructionExecutionsThatHaveTimedOut
	ChannelCommandProcessTestInstructionExecutionsThatHaveTimedOut
)

// ChannelCommandStruct
// The struct for the message that are sent over the channel to the executionEngine
type ChannelCommandStruct struct {
	ChannelCommand                          ChannelCommandType
	ChannelCommandTestCaseExecutions        []ChannelCommandTestCaseExecutionStruct
	ChannelCommandTestInstructionExecutions []ChannelCommandTestInstructionExecutionStruct
	ReturnChannelWithDBErrorReference       *ReturnChannelWithDBErrorType
}

// ChannelCommandTestCaseExecutionStruct
// Hold one TestCaseExecution that will be processed by the executionEngine
type ChannelCommandTestCaseExecutionStruct struct {
	TestCaseExecutionUuid    string
	TestCaseExecutionVersion int32
}

// ChannelCommandTestInstructionExecutionStruct
// Hold one TestInstructionExecution that will be processed by the executionEngine
type ChannelCommandTestInstructionExecutionStruct struct {
	TestCaseExecutionUuid                   string
	TestCaseExecutionVersion                int32
	TestInstructionExecutionUuid            string
	TestInstructionExecutionVersion         int32
	TestInstructionExecutionCanBeReExecuted bool
}

// ReturnChannelWithDBErrorType
// Channel used for response from ExecutionEngine when one command has finished and there is another command waiting for it to finsig
type ReturnChannelWithDBErrorType chan ReturnChannelWithDBErrorStruct

// ReturnChannelWithDBErrorStruct
// The struct for the message that are sent over the 'return-channel'
type ReturnChannelWithDBErrorStruct struct {
	Err error
}
