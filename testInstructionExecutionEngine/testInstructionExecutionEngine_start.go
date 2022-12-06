package testInstructionExecutionEngine

import "github.com/sirupsen/logrus"

// InitiateTestInstructionExecutionEngineCommandChannelReader
// Initiate the channel reader which is used for sending commands to TestInstruction Execution Engine
func (executionEngine *TestInstructionExecutionEngineStruct) InitiateTestInstructionExecutionEngineCommandChannelReader(executionEngineCommandChannel ExecutionEngineChannelType) {

	executionEngine.CommandChannelReference = &executionEngineCommandChannel
	go executionEngine.startCommandChannelReader()

	// Trigger TestInstructionEngine to check if there are any Zombie-TestInstructionExecutions stuck in UnderExecutions
	channelCommandMessage := ChannelCommandStruct{
		ChannelCommand:                   ChannelCommandLookForZombieTestInstructionExecutionsInUnderExecution,
		ChannelCommandTestCaseExecutions: nil,
	}

	// Send Message on Channel
	*executionEngine.CommandChannelReference <- channelCommandMessage

	return
}

// SetLogger
// Set to use the same Logger reference as is used by central part of system
func (executionEngine *TestInstructionExecutionEngineStruct) SetLogger(logger *logrus.Logger) {

	executionEngine.logger = logger

	return

}
