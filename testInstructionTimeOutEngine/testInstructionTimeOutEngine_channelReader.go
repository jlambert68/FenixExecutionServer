package testInstructionTimeOutEngine

import (
	"FenixExecutionServer/common_config"
	"github.com/sirupsen/logrus"
)

// Channel reader which is used for reading out commands to CommandEngine
func (testInstructionExecutionTimeOutEngineObject *TestInstructionTimeOutEngineObjectStruct) startTimeOutChannelReader() {

	var incomingTimeOutChannelCommand TimeOutChannelCommandStruct

	for {
		// Wait for incoming command over channel
		incomingTimeOutChannelCommand = <-TimeOutChannelEngineCommandChannel

		switch incomingTimeOutChannelCommand.TimeOutChannelCommand {

		case TimeOutChannelCommandAddTestInstructionExecutionToTimeOutTimer: //(A)
			executionEngine.moveTestInstructionExecutionsFromExecutionQueueToOngoingExecutions(incomingChannelCommand.ChannelCommandTestCaseExecutions)

		case TimeOutChannelCommandRemoveTestInstructionExecutionFromTimeOutTimer: //(B)
			executionEngine.checkForTestInstructionsExecutionsWaitingToBeSentToWorker(incomingChannelCommand.ChannelCommandTestCaseExecutions)

		case TimeOutChannelCommandExistsTestInstructionExecutionInTimeOutTimer: // NOT USED FOR NOW
			executionEngine.checkOngoingExecutionsForTestInstructions()

		// No other command is supported
		default:
			common_config.Logger.WithFields(logrus.Fields{
				"Id":                     "b18a28a0-7ff9-41fb-a9be-8990e58e4b78",
				"incomingChannelCommand": incomingTimeOutChannelCommand,
			}).Fatalln("Unknown command in TimeOutCommandChannel for TimeOutEngine")
		}
	}

}
