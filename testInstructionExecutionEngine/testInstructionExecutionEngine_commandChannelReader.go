package testInstructionExecutionEngine

import (
	"FenixExecutionServer/common_config"
	"fmt"
	"github.com/sirupsen/logrus"
)

// Channel reader which is used for reading out commands to CommandEngine
func (executionEngine *TestInstructionExecutionEngineStruct) startCommandChannelReader() {

	var incomingChannelCommand ChannelCommandStruct
	var channelSize int

	for {
		// Wait for incoming command over channel
		incomingChannelCommand = <-*executionEngine.CommandChannelReference

		// If size of Channel > 'ExecutionEngineChannelWarningLevel' then log Warning message
		channelSize = len(*executionEngine.CommandChannelReference)
		if channelSize > ExecutionEngineChannelWarningLevel {
			common_config.Logger.WithFields(logrus.Fields{
				"Id":                                 "b2ee2644-d318-4a7f-882a-cddd80058608",
				"channelSize":                        channelSize,
				"ExecutionEngineChannelWarningLevel": ExecutionEngineChannelWarningLevel,
				"ExecutionEngineChannelSize":         ExecutionEngineChannelSize,
			}).Warning("Number of messages on queue for 'ExecutionEngineChannel' has reached a critical level")
		}

		switch incomingChannelCommand.ChannelCommand {

		case ChannelCommandCheckForTestInstructionExecutionWaitingOnQueue: //(A)
			executionEngine.moveTestInstructionExecutionsFromExecutionQueueToOngoingExecutions(incomingChannelCommand.ChannelCommandTestCaseExecutions)

		case ChannelCommandCheckForTestInstructionExecutionsWaitingToBeSentToWorker: //(B)
			executionEngine.checkForTestInstructionsExecutionsWaitingToBeSentToWorker(incomingChannelCommand.ChannelCommandTestCaseExecutions)

		case ChannelCommandCheckOngoingTestInstructionExecutions: // NOT USED FOR NOW
			executionEngine.checkOngoingExecutionsForTestInstructions()

		case ChannelCommandUpdateExecutionStatusOnTestCaseExecutionExecutions: // (C)
			executionEngine.updateStatusOnTestCaseExecution(incomingChannelCommand)

		case ChannelCommandLookForZombieTestInstructionExecutionsInUnderExecution:
			executionEngine.triggerLookForZombieTestInstructionExecutionsInUnderExecution()

		case ChannelCommandProcessTestCaseExecutionsOnExecutionQueue:
			executionEngine.processTestCaseExecutionsOnExecutionQueue(incomingChannelCommand.ChannelCommandTestCaseExecutions)

		case ChannelCommandSendZombieTestCaseExecutionThatAreStuckOnExecutionQueue:
			executionEngine.triggerLookForZombieTestCaseExecutionsOnExecutionQueue()

		case ChannelCommandLookForZombieTestInstructionExecutionsOnExecutionQueue:
			executionEngine.triggerLookForZombieTestInstructionExecutionsOnExecutionQueue()

		case ChannelCommandLookForZombieTestInstructionExecutionsThatHaveTimedOut:
			executionEngine.triggerLookForZombieTestInstructionExecutionsThatHaveTimedOut()

		case ChannelCommandProcessTestInstructionExecutionsThatHaveTimedOut:
			executionEngine.triggerProcessTestInstructionExecutionsThatHaveTimedOut(incomingChannelCommand.ChannelCommandTestInstructionExecutions)

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
func (executionEngine *TestInstructionExecutionEngineStruct) moveTestInstructionExecutionsFromExecutionQueueToOngoingExecutions(channelCommandTestCasesExecution []ChannelCommandTestCaseExecutionStruct) {

	executionEngine.moveTestInstructionExecutionsFromExecutionQueueToOngoingExecutionsSaveToCloudDB(channelCommandTestCasesExecution)
	/*
		// Trigger TestInstructionEngine to check if there are TestInstructions that should be sent to workers
		channelCommandMessage := ChannelCommandStruct{
			ChannelCommand:                   ChannelCommandCheckForTestInstructionExecutionsWaitingToBeSentToWorker,
			ChannelCommandTestCaseExecutions: channelCommandTestCasesExecution,
		}

		// Send Message on Channel
		*executionEngine.CommandChannelReference <- channelCommandMessage
	*/
}

// Check for new executions for TestInstructions that should be sent to workers
func (executionEngine *TestInstructionExecutionEngineStruct) updateStatusOnTestCaseExecution(incomingChannelCommand ChannelCommandStruct) {

	err := executionEngine.updateStatusOnTestCaseExecutionInCloudDB(incomingChannelCommand.ChannelCommandTestCaseExecutions)

	// If there is a Channel reference present then send back Error-message from DB-update
	if incomingChannelCommand.ReturnChannelWithDBErrorReference != nil {

		var returnChannelWithDBError ReturnChannelWithDBErrorStruct
		returnChannelWithDBError = ReturnChannelWithDBErrorStruct{
			Err: err}

		*incomingChannelCommand.ReturnChannelWithDBErrorReference <- returnChannelWithDBError
	}
}

// Check for ongoing executions  for TestInstructions for change in status that should be propagated to other places
func (executionEngine *TestInstructionExecutionEngineStruct) checkOngoingExecutionsForTestInstructions() {
	fmt.Println("ÄÄÄÄÄÄÄÄÄÄÄÄÄ CODE IS MISSING FOR THIS ONE!!! ÖÖÖÖÖÖÖÖÖÖÖÖÖÖÖÖÖÖÖÖÖÖÖ")
}

// Update TestCaseExecutionStatus based on result on individual TestInstructionExecution-results
func (executionEngine *TestInstructionExecutionEngineStruct) checkForTestInstructionsExecutionsWaitingToBeSentToWorker(channelCommandTestCasesExecution []ChannelCommandTestCaseExecutionStruct) {

	executionEngine.sendNewTestInstructionsThatIsWaitingToBeSentWorker(channelCommandTestCasesExecution)
}

// Look for Zombie-TransactionsExecutions that were sent to Worker, but was lost in some way
func (executionEngine *TestInstructionExecutionEngineStruct) triggerLookForZombieTestInstructionExecutionsInUnderExecution() {

	// Look for Zombie-TestInstructionExecutions in UnderExecution
	_ = executionEngine.sendAllZombieTestInstructionsUnderExecution()

	// Trigger TestInstructionEngine to check if there are any Zombie-TestCaseExecutions stuck OnExecutionQueue
	channelCommandMessage := ChannelCommandStruct{
		ChannelCommand:                   ChannelCommandSendZombieTestCaseExecutionThatAreStuckOnExecutionQueue,
		ChannelCommandTestCaseExecutions: nil,
	}

	// Send Message on Channel
	*executionEngine.CommandChannelReference <- channelCommandMessage

}

// Look for Zombie-TestCaseExecutions that are waiting on OnQueue, but was lost in some way
func (executionEngine *TestInstructionExecutionEngineStruct) triggerLookForZombieTestCaseExecutionsOnExecutionQueue() {

	_ = executionEngine.lookForZombieTestCaseExecutionsOnExecutionQueue()

	// Trigger TestInstructionEngine to check if there are any Zombie-TestInstructionsExecutions stuck OnExecutionQueue
	channelCommandMessage := ChannelCommandStruct{
		ChannelCommand:                   ChannelCommandLookForZombieTestInstructionExecutionsOnExecutionQueue,
		ChannelCommandTestCaseExecutions: nil,
	}

	// Send Message on Channel
	*executionEngine.CommandChannelReference <- channelCommandMessage

}

// Look for Zombie-TestCaseExecutions that are waiting on OnQueue, but was lost in some way
func (executionEngine *TestInstructionExecutionEngineStruct) processTestCaseExecutionsOnExecutionQueue(channelCommandTestCasesExecution []ChannelCommandTestCaseExecutionStruct) {

	_ = executionEngine.prepareInformThatThereAreNewTestCasesOnExecutionQueueSaveToCloudDB(channelCommandTestCasesExecution)

}

// Look for Zombie-TestInstructionExecutions that are waiting on OnQueue, but was lost in some way
func (executionEngine *TestInstructionExecutionEngineStruct) triggerLookForZombieTestInstructionExecutionsOnExecutionQueue() {

	_ = executionEngine.sendAllZombieTestInstructionsOnExecutionQueue()

}

// Look for Zombie-TestInstructionExecutions that have timed out, but was lost in some way
func (executionEngine *TestInstructionExecutionEngineStruct) triggerLookForZombieTestInstructionExecutionsThatHaveTimedOut() {

	_ = executionEngine.findAllZombieTestInstructionExecutionsInTimeout()

}

// Process for Zombie-TestInstructionExecutions that have timed out
func (executionEngine *TestInstructionExecutionEngineStruct) triggerProcessTestInstructionExecutionsThatHaveTimedOut(testInstructionExecutionsToProcess []ChannelCommandTestInstructionExecutionStruct) {

	executionEngine.timeoutHandlerForTestInstructionsUnderExecution(testInstructionExecutionsToProcess)

}
