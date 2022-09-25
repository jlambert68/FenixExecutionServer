package testInstructionExecutionEngine

import "github.com/sirupsen/logrus"

type TestInstructionExecutionEngineStruct struct {
	logger                  *logrus.Logger
	CommandChannelReference *ExecutionEngineChannelType
}

// Parameters used for channel to trigger TestInstructionExecutionEngine
var ExecutionEngineCommandChannel ExecutionEngineChannelType

type ExecutionEngineChannelType chan ChannelCommandStruct

type ChannelCommandType uint8

const (
	ChannelCommandCheckTestInstructionExecutionQueue ChannelCommandType = iota
	ChannelCommandCheckOngoingTestInstructionExecutions
)

type ChannelCommandStruct struct {
	ChannelCommand ChannelCommandType
}
