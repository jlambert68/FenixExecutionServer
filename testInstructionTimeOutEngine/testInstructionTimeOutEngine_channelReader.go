package testInstructionTimeOutEngine

import (
	"FenixExecutionServer/common_config"
	"github.com/sirupsen/logrus"
)

// Channel reader which is used for reading out commands to CommandEngine
func (testInstructionExecutionTimeOutEngineObject *TestInstructionTimeOutEngineObjectStruct) startTimeOutChannelReader() {

	var incomingTimeOutChannelCommand common_config.TimeOutChannelCommandStruct
	var channelSize int

	for {
		// Wait for incoming command over channel
		incomingTimeOutChannelCommand = <-TimeOutChannelEngineCommandChannel

		// If size of Channel > 'timeOutChannelWarningLevel' then log Warning message
		channelSize = len(TimeOutChannelEngineCommandChannel)
		if channelSize > timeOutChannelWarningLevel {
			common_config.Logger.WithFields(logrus.Fields{
				"Id":                         "7dafce7a-eed9-448c-81b6-1015733dc1cb",
				"channelSize":                channelSize,
				"timeOutChannelWarningLevel": timeOutChannelWarningLevel,
				"timeOutChannelSize":         timeOutChannelSize,
			}).Warning("Number of messages on queue for 'TimeOutChannel' has reached a critical level")
		}

		switch incomingTimeOutChannelCommand.TimeOutChannelCommand {

		case common_config.TimeOutChannelCommandAddTestInstructionExecutionToTimeOutTimer:
			testInstructionExecutionTimeOutEngineObject.addTestInstructionExecutionToTimeOutTimer(
				incomingTimeOutChannelCommand)

		case common_config.TimeOutChannelCommandRemoveTestInstructionExecutionFromTimeOutTimerDueToTimeOutFromTimer:
			testInstructionExecutionTimeOutEngineObject.removeTestInstructionExecutionFromTimeOutTimer(
				incomingTimeOutChannelCommand,
				common_config.TimeOutChannelCommandRemoveTestInstructionExecutionFromTimeOutTimerDueToTimeOutFromTimer)

		case common_config.TimeOutChannelCommandRemoveTestInstructionExecutionFromTimeOutTimerDueToExecutionResult:
			testInstructionExecutionTimeOutEngineObject.removeTestInstructionExecutionFromTimeOutTimer(
				incomingTimeOutChannelCommand,
				common_config.TimeOutChannelCommandRemoveTestInstructionExecutionFromTimeOutTimerDueToExecutionResult)

		case common_config.TimeOutChannelCommandExistsTestInstructionExecutionInTimeOutTimer:
			testInstructionExecutionTimeOutEngineObject.existsTestInstructionExecutionInTimeOutTimer(
				incomingTimeOutChannelCommand)

		// No other command is supported
		default:
			common_config.Logger.WithFields(logrus.Fields{
				"Id":                     "b18a28a0-7ff9-41fb-a9be-8990e58e4b78",
				"incomingChannelCommand": incomingTimeOutChannelCommand,
			}).Fatalln("Unknown command in TimeOutCommandChannel for TimeOutEngine")
		}
	}
}

// Add TestInstructionExecution to TimeOut-timer
func (testInstructionExecutionTimeOutEngineObject *TestInstructionTimeOutEngineObjectStruct) addTestInstructionExecutionToTimeOutTimer(
	incomingTimeOutChannelCommand common_config.TimeOutChannelCommandStruct) {

	testInstructionExecutionTimeOutEngineObject.processAddTestInstructionExecutionToTimeOutTimer(
		&incomingTimeOutChannelCommand)
}

// Remove TestInstructionExecution from TimeOut-timer
func (testInstructionExecutionTimeOutEngineObject *TestInstructionTimeOutEngineObjectStruct) removeTestInstructionExecutionFromTimeOutTimer(
	incomingTimeOutChannelCommand common_config.TimeOutChannelCommandStruct,
	timeOutChannelCommand common_config.TimeOutChannelCommandType) {

	testInstructionExecutionTimeOutEngineObject.processRemoveTestInstructionExecutionFromTimeOutTimer(
		&incomingTimeOutChannelCommand,
		timeOutChannelCommand)
}

// Check if TestInstructionExecution exists within TimeOut-timer
func (testInstructionExecutionTimeOutEngineObject *TestInstructionTimeOutEngineObjectStruct) existsTestInstructionExecutionInTimeOutTimer(
	incomingTimeOutChannelCommand common_config.TimeOutChannelCommandStruct) {

}
