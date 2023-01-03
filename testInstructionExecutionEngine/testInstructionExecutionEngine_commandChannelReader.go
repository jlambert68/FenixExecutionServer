package testInstructionExecutionEngine

import (
	"FenixExecutionServer/common_config"
	"fmt"
	fenixExecutionServerGrpcApi "github.com/jlambert68/FenixGrpcApi/FenixExecutionServer/fenixExecutionServerGrpcApi/go_grpc_api"
	"github.com/sirupsen/logrus"
)

// Channel reader which is used for reading out commands to CommandEngine
func (executionEngine *TestInstructionExecutionEngineStruct) startCommandChannelReader(executionTrackNumber int) {

	var incomingChannelCommand ChannelCommandStruct
	var channelSize int

	for {
		// Wait for incoming command over channel
		incomingChannelCommand = <-*executionEngine.CommandChannelReferenceSlice[executionTrackNumber]

		// If size of Channel > 'ExecutionEngineChannelWarningLevel' then log Warning message
		channelSize = len(*executionEngine.CommandChannelReferenceSlice[executionTrackNumber])
		if channelSize > ExecutionEngineChannelWarningLevel {
			common_config.Logger.WithFields(logrus.Fields{
				"Id":                                 "b2ee2644-d318-4a7f-882a-cddd80058608",
				"executionTrackNumber":               executionTrackNumber,
				"channelSize":                        channelSize,
				"ExecutionEngineChannelWarningLevel": ExecutionEngineChannelWarningLevel,
				"ExecutionEngineChannelSize":         ExecutionEngineChannelSize,
			}).Warning("Number of messages on queue for 'ExecutionEngineChannel' has reached a critical level")
		} else {
			common_config.Logger.WithFields(logrus.Fields{
				"Id":                                    "46ed3253-a3ac-47ea-9191-e18f5b30ef87",
				"executionTrackNumber":                  executionTrackNumber,
				"channelSize":                           channelSize,
				"ExecutionEngineChannelWarningLevel":    ExecutionEngineChannelWarningLevel,
				"ExecutionEngineChannelSize":            ExecutionEngineChannelSize,
				"incomingChannelCommand.ChannelCommand": incomingChannelCommand.ChannelCommand,
			}).Info("Incoming ExecutionEngineCommand")
		}

		switch incomingChannelCommand.ChannelCommand {

		case ChannelCommandCheckForTestInstructionExecutionWaitingOnQueue: //(A)
			executionEngine.moveTestInstructionExecutionsFromExecutionQueueToOngoingExecutions(
				executionTrackNumber,
				incomingChannelCommand.ChannelCommandTestCaseExecutions)

		case ChannelCommandCheckForTestInstructionExecutionsWaitingToBeSentToWorker: //(B)
			executionEngine.checkForTestInstructionsExecutionsWaitingToBeSentToWorker(
				executionTrackNumber,
				incomingChannelCommand.ChannelCommandTestCaseExecutions)

		case ChannelCommandCheckOngoingTestInstructionExecutions: // NOT USED FOR NOW
			executionEngine.checkOngoingExecutionsForTestInstructions(executionTrackNumber)

		case ChannelCommandUpdateExecutionStatusOnTestCaseExecutionExecutions: // (C)
			executionEngine.updateStatusOnTestCaseExecution(
				executionTrackNumber,
				incomingChannelCommand)

		case ChannelCommandLookForZombieTestInstructionExecutionsInUnderExecution:
			executionEngine.triggerLookForZombieTestInstructionExecutionsInUnderExecution(executionTrackNumber)

		case ChannelCommandProcessTestCaseExecutionsOnExecutionQueue:
			executionEngine.processTestCaseExecutionsOnExecutionQueue(
				executionTrackNumber,
				incomingChannelCommand.ChannelCommandTestCaseExecutions)

		case ChannelCommandSendZombieTestCaseExecutionThatAreStuckOnExecutionQueue:
			executionEngine.triggerLookForZombieTestCaseExecutionsOnExecutionQueue(executionTrackNumber)

		case ChannelCommandLookForZombieTestInstructionExecutionsOnExecutionQueue:
			executionEngine.triggerLookForZombieTestInstructionExecutionsOnExecutionQueue(executionTrackNumber)

		case ChannelCommandLookForZombieTestInstructionExecutionsThatHaveTimedOut:
			executionEngine.triggerLookForZombieTestInstructionExecutionsThatHaveTimedOut(executionTrackNumber)

		case ChannelCommandProcessTestInstructionExecutionsThatHaveTimedOut:
			executionEngine.triggerProcessTestInstructionExecutionsThatHaveTimedOut(
				executionTrackNumber,
				incomingChannelCommand.ChannelCommandTestInstructionExecutions)

		case ChannelCommandProcessFinalTestInstructionExecutionResultMessage:
			executionEngine.triggerProcessReportCompleteTestInstructionExecutionResultSaveToCloudDB(
				executionTrackNumber,
				incomingChannelCommand.FinalTestInstructionExecutionResultMessage)

		case ChannelCommandReCreateTimeOutTimersAtApplicationStartUp:
			executionEngine.triggerReCreateTimeOutTimersAtApplicationStartUp(executionTrackNumber)

		// No other command is supported
		default:
			executionEngine.logger.WithFields(logrus.Fields{
				"Id":                     "6bf37452-da99-4e7e-aa6a-4627b05d1bdb",
				"executionTrackNumber":   executionTrackNumber,
				"incomingChannelCommand": incomingChannelCommand,
			}).Fatalln("Unknown command in CommandChannel for TestInstructionEngine")
		}
	}
}

// Check ExecutionQueue for TestInstructions and move them to ongoing Executions-table
func (executionEngine *TestInstructionExecutionEngineStruct) moveTestInstructionExecutionsFromExecutionQueueToOngoingExecutions(
	executionTrackNumber int,
	channelCommandTestCasesExecution []ChannelCommandTestCaseExecutionStruct) {

	executionEngine.moveTestInstructionExecutionsFromExecutionQueueToOngoingExecutionsSaveToCloudDB(
		executionTrackNumber,
		channelCommandTestCasesExecution)
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
func (executionEngine *TestInstructionExecutionEngineStruct) updateStatusOnTestCaseExecution(
	executionTrackNumber int,
	incomingChannelCommand ChannelCommandStruct) {

	err := executionEngine.updateStatusOnTestCaseExecutionInCloudDB(
		executionTrackNumber,
		incomingChannelCommand.ChannelCommandTestCaseExecutions)

	// If there is a Channel reference present then send back Error-message from DB-update
	if incomingChannelCommand.ReturnChannelWithDBErrorReference != nil {

		var returnChannelWithDBError ReturnChannelWithDBErrorStruct
		returnChannelWithDBError = ReturnChannelWithDBErrorStruct{
			Err: err}

		*incomingChannelCommand.ReturnChannelWithDBErrorReference <- returnChannelWithDBError
	}
}

// Check for ongoing executions  for TestInstructions for change in status that should be propagated to other places
func (executionEngine *TestInstructionExecutionEngineStruct) checkOngoingExecutionsForTestInstructions(
	executionTrackNumber int) {
	fmt.Println("ÄÄÄÄÄÄÄÄÄÄÄÄÄ CODE IS MISSING FOR THIS ONE!!! ÖÖÖÖÖÖÖÖÖÖÖÖÖÖÖÖÖÖÖÖÖÖÖ")
}

// Update TestCaseExecutionStatus based on result on individual TestInstructionExecution-results
func (executionEngine *TestInstructionExecutionEngineStruct) checkForTestInstructionsExecutionsWaitingToBeSentToWorker(
	executionTrackNumber int,
	channelCommandTestCasesExecution []ChannelCommandTestCaseExecutionStruct) {

	executionEngine.sendNewTestInstructionsThatIsWaitingToBeSentWorker(
		executionTrackNumber,
		channelCommandTestCasesExecution)
}

// Look for Zombie-TransactionsExecutions that were sent to Worker, but was lost in some way
func (executionEngine *TestInstructionExecutionEngineStruct) triggerLookForZombieTestInstructionExecutionsInUnderExecution(
	executionTrackNumber int) {

	// Look for Zombie-TestInstructionExecutions in UnderExecution
	_ = executionEngine.sendAllZombieTestInstructionsUnderExecution(executionTrackNumber)

	// Trigger TestInstructionEngine to check if there are any Zombie-TestCaseExecutions stuck OnExecutionQueue
	channelCommandMessage := ChannelCommandStruct{
		ChannelCommand:                   ChannelCommandSendZombieTestCaseExecutionThatAreStuckOnExecutionQueue,
		ChannelCommandTestCaseExecutions: nil,
	}

	// Send Message on Channel
	*executionEngine.CommandChannelReferenceSlice[executionTrackNumber] <- channelCommandMessage

}

// Look for Zombie-TestCaseExecutions that are waiting on OnQueue, but was lost in some way
func (executionEngine *TestInstructionExecutionEngineStruct) triggerLookForZombieTestCaseExecutionsOnExecutionQueue(
	executionTrackNumber int) {

	_ = executionEngine.lookForZombieTestCaseExecutionsOnExecutionQueue(executionTrackNumber)

	// Trigger TestInstructionEngine to check if there are any Zombie-TestInstructionsExecutions stuck OnExecutionQueue
	channelCommandMessage := ChannelCommandStruct{
		ChannelCommand:                   ChannelCommandLookForZombieTestInstructionExecutionsOnExecutionQueue,
		ChannelCommandTestCaseExecutions: nil,
	}

	// Send Message on Channel
	*executionEngine.CommandChannelReferenceSlice[executionTrackNumber] <- channelCommandMessage

}

// Look for Zombie-TestCaseExecutions that are waiting on OnQueue, but was lost in some way
func (executionEngine *TestInstructionExecutionEngineStruct) processTestCaseExecutionsOnExecutionQueue(
	executionTrackNumber int,
	channelCommandTestCasesExecution []ChannelCommandTestCaseExecutionStruct) {

	_ = executionEngine.prepareInformThatThereAreNewTestCasesOnExecutionQueueSaveToCloudDB(
		channelCommandTestCasesExecution)

}

// Look for Zombie-TestInstructionExecutions that are waiting on OnQueue, but was lost in some way
func (executionEngine *TestInstructionExecutionEngineStruct) triggerLookForZombieTestInstructionExecutionsOnExecutionQueue(
	executionTrackNumber int) {

	_ = executionEngine.sendAllZombieTestInstructionsOnExecutionQueue(executionTrackNumber)

}

// Look for Zombie-TestInstructionExecutions that have timed out, but was lost in some way
func (executionEngine *TestInstructionExecutionEngineStruct) triggerLookForZombieTestInstructionExecutionsThatHaveTimedOut(
	executionTrackNumber int) {

	_ = executionEngine.findAllZombieTestInstructionExecutionsInTimeout(executionTrackNumber)

}

// Process for Zombie-TestInstructionExecutions that have timed out
func (executionEngine *TestInstructionExecutionEngineStruct) triggerProcessTestInstructionExecutionsThatHaveTimedOut(
	executionTrackNumber int,
	testInstructionExecutionsToProcess []ChannelCommandTestInstructionExecutionStruct) {

	executionEngine.timeoutHandlerForTestInstructionsUnderExecution(
		executionTrackNumber,
		testInstructionExecutionsToProcess)

}

// Saving the ongoing Execution of a new TestCaseExecutionUuid in the CloudDB
func (executionEngine *TestInstructionExecutionEngineStruct) triggerProcessReportCompleteTestInstructionExecutionResultSaveToCloudDB(
	executionTrackNumber int,
	finalTestInstructionExecutionResultMessage *fenixExecutionServerGrpcApi.FinalTestInstructionExecutionResultMessage) {

	go func() {
		_ = executionEngine.prepareReportCompleteTestInstructionExecutionResultSaveToCloudDB(
			executionTrackNumber,
			finalTestInstructionExecutionResultMessage)
	}()

}

// Load TimeOut-times for TestInstructionExecutions and create new TimeOut-Timers in TimeOut-Engine
func (executionEngine *TestInstructionExecutionEngineStruct) triggerReCreateTimeOutTimersAtApplicationStartUp(
	executionTrackNumber int) {

	executionEngine.reCreateTimeOutTimersAtApplicationStartUp(executionTrackNumber)

}
