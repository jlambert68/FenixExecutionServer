package testInstructionExecutionEngine

import "github.com/sirupsen/logrus"

// InitiateTestInstructionExecutionEngineCommandChannelReader
// Initiate the channel reader which is used for sending commands to TestInstruction Execution Engine
func (executionEngine *TestInstructionExecutionEngineStruct) InitiateTestInstructionExecutionEngineCommandChannelReader(executionEngineCommandChannel ExecutionEngineChannelType) {

	executionEngine.CommandChannelReference = &executionEngineCommandChannel
	go executionEngine.startCommandChannelReader()

	return
}

// SetLogger
// Set to use the same Logger reference as is used by central part of system
func (executionEngine *TestInstructionExecutionEngineStruct) SetLogger(logger *logrus.Logger) {

	executionEngine.logger = logger

	return

}
