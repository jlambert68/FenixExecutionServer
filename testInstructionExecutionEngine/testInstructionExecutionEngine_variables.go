package testInstructionExecutionEngine

import (
	"github.com/sirupsen/logrus"
)

type TestInstructionExecutionEngineStruct struct {
	logger                  *logrus.Logger
	CommandChannelReference *ExecutionEngineChannelType
}

// TestInstructionExecutionEngineObject
var TestInstructionExecutionEngineObject TestInstructionExecutionEngineStruct

// Parameters used for channel to trigger TestInstructionExecutionEngine
var ExecutionEngineCommandChannel ExecutionEngineChannelType

type ExecutionEngineChannelType chan ChannelCommandStruct

type ChannelCommandType uint8

const (
	ChannelCommandCheckForTestInstructionExecutionWaitingOnQueue ChannelCommandType = iota
	ChannelCommandCheckForTestInstructionExecutionsWaitingToBeSentToWorker
	ChannelCommandCheckOngoingTestInstructionExecutions
	ChannelCommandUpdateExecutionStatusOnTestCaseExecutionExecutions
	ChannelCommandLookForZombieTestInstructionExecutionsInUnderExecution
	ChannelCommandProcessTestCaseExecutionsOnExecutionQueue
	ChannelCommandSendZombieTestCaseExecutionThatAreStuckOnExecutionQueue
)

type ChannelCommandStruct struct {
	ChannelCommand                    ChannelCommandType
	ChannelCommandTestCaseExecutions  []ChannelCommandTestCaseExecutionStruct
	ReturnChannelWithDBErrorReference *ReturnChannelWithDBErrorType
}

type ChannelCommandTestCaseExecutionStruct struct {
	TestCaseExecutionUuid    string
	TestCaseExecutionVersion int32
}

// Channel used for response from ExecutionEngine when one command has finished and there is another command waiting for it to finsig
type ReturnChannelWithDBErrorType chan ReturnChannelWithDBErrorStruct
type ReturnChannelWithDBErrorStruct struct {
	Err error
}
