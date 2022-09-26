package testInstructionExecutionEngine

import (
	"fmt"
	"github.com/sirupsen/logrus"
)

// Channel reader which is used for reading out commands to CommandEngine
func (executionEngine *TestInstructionExecutionEngineStruct) startCommandChannelReader() {

	var incomingChannelCommand ChannelCommandStruct

	for {
		// Wait for incoming command over channel
		incomingChannelCommand = <-*executionEngine.CommandChannelReference

		switch incomingChannelCommand.ChannelCommand {

		case ChannelCommandCheckTestInstructionExecutionQueue:
			executionEngine.initiateExecutionsForTestInstructionsOnExecutionQueue()

		case ChannelCommandCheckOngoingTestInstructionExecutions:
			executionEngine.checkOngoingExecutionsForTestInstructions()

		// No other command is supported
		default:
			executionEngine.logger.WithFields(logrus.Fields{
				"Id":                     "6bf37452-da99-4e7e-aa6a-4627b05d1bdb",
				"incomingChannelCommand": incomingChannelCommand,
			}).Fatalln("Unknown command in CommandChannel for TestInstructionEngine")
		}
	}

}

// Check ExecutionQueue for TestInstructions and move them to ongoing Executions-table
func (executionEngine *TestInstructionExecutionEngineStruct) initiateExecutionsForTestInstructionsOnExecutionQueue() {

	fmt.Println("initiateExecutionsForTestInstructionsOnExecutionQueue")
	executionEngine.prepareInitiateExecutionsForTestInstructionsOnExecutionQueueSaveToCloudDB()
}

// Check ongoing executions  for TestInstructions for change in status that should be propagated to other places
func (executionEngine *TestInstructionExecutionEngineStruct) checkOngoingExecutionsForTestInstructions() {

}
