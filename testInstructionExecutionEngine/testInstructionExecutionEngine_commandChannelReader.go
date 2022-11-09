package testInstructionExecutionEngine

import (
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
			executionEngine.initiateExecutionsForTestInstructionsOnExecutionQueue(incomingChannelCommand.ChannelCommandTestCaseExecutions)

		case ChannelCommandCheckNewTestInstructionExecutions:
			executionEngine.checkNewExecutionsForTestInstructions(incomingChannelCommand.ChannelCommandTestCaseExecutions)

		case ChannelCommandCheckOngoingTestInstructionExecutions:
			executionEngine.checkOngoingExecutionsForTestInstructions()

		case ChannelCommandUpdateFinalExecutionStatusOnTestCaseExecutionExecutions:
			executionEngine.updateStatusOnTestCaseExecution(incomingChannelCommand.ChannelCommandTestCaseExecutions)

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
func (executionEngine *TestInstructionExecutionEngineStruct) initiateExecutionsForTestInstructionsOnExecutionQueue(channelCommandTestCasesExecution []ChannelCommandTestCaseExecutionStruct) {

	executionEngine.prepareInitiateExecutionsForTestInstructionsOnExecutionQueueSaveToCloudDB(channelCommandTestCasesExecution)
	/*
		// Trigger TestInstructionEngine to check if there are TestInstructions that should be sent to workers
		channelCommandMessage := ChannelCommandStruct{
			ChannelCommand:                   ChannelCommandCheckNewTestInstructionExecutions,
			ChannelCommandTestCaseExecutions: channelCommandTestCasesExecution,
		}

		// Send Message on Channel
		*executionEngine.CommandChannelReference <- channelCommandMessage
	*/
}

// Check for new executions for TestInstructions that should be sent to workers
func (executionEngine *TestInstructionExecutionEngineStruct) updateStatusOnTestCaseExecution(channelCommandTestCasesExecution []ChannelCommandTestCaseExecutionStruct) {

	_ = executionEngine.updateStatusOnTestCaseExecutionInCloudDB(channelCommandTestCasesExecution)
}

// Check for ongoing executions  for TestInstructions for change in status that should be propagated to other places
func (executionEngine *TestInstructionExecutionEngineStruct) checkOngoingExecutionsForTestInstructions() {

}

// Update TestCaseExecutionStatus based on result on individual TestInstructionExecution-results
func (executionEngine *TestInstructionExecutionEngineStruct) checkNewExecutionsForTestInstructions(channelCommandTestCasesExecution []ChannelCommandTestCaseExecutionStruct) {

	executionEngine.sendNewTestInstructionsThatIsWaitingToBeSentWorker(channelCommandTestCasesExecution)
}
