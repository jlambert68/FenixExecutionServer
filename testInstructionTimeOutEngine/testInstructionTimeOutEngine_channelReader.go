package testInstructionTimeOutEngine

import (
	"FenixExecutionServer/common_config"
	"github.com/sirupsen/logrus"
)

// Channel reader which is used for reading out commands to CommandEngine
func (testInstructionExecutionTimeOutEngineObject *TestInstructionTimeOutEngineObjectStruct) startTimeOutChannelReader() {

	var incomingTimeOutChannelCommand TimeOutChannelCommandStruct
	var channelSize int

	for {
		// Wait for incoming command over channel
		incomingTimeOutChannelCommand = <-TimeOutChannelEngineCommandChannel

		// If size of Channel > 'TimeOutChannelWarningLevel' then log Warning message
		channelSize = len(TimeOutChannelEngineCommandChannel)
		if channelSize > TimeOutChannelWarningLevel {
			common_config.Logger.WithFields(logrus.Fields{
				"Id":                         "7dafce7a-eed9-448c-81b6-1015733dc1cb",
				"channelSize":                channelSize,
				"TimeOutChannelWarningLevel": TimeOutChannelWarningLevel,
				"TimeOutChannelSize":         TimeOutChannelSize,
			}).Warning("Number of messages on queue for 'TimeOutChannel' has reached a critical level")
		}

		switch incomingTimeOutChannelCommand.TimeOutChannelCommand {

		case TimeOutChannelCommandAddTestInstructionExecutionToTimeOutTimer:
			testInstructionExecutionTimeOutEngineObject.addTestInstructionExecutionToTimeOutTimer(incomingTimeOutChannelCommand)

		case TimeOutChannelCommandRemoveTestInstructionExecutionFromTimeOutTimer:
			testInstructionExecutionTimeOutEngineObject.removeTestInstructionExecutionFromTimeOutTimer(incomingTimeOutChannelCommand)

		case TimeOutChannelCommandExistsTestInstructionExecutionInTimeOutTimer:
			testInstructionExecutionTimeOutEngineObject.existsTestInstructionExecutionInTimeOutTimer(incomingTimeOutChannelCommand)

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
func (testInstructionExecutionTimeOutEngineObject *TestInstructionTimeOutEngineObjectStruct) addTestInstructionExecutionToTimeOutTimer(incomingTimeOutChannelCommand TimeOutChannelCommandStruct) {

	testInstructionExecutionTimeOutEngineObject.processAddTestInstructionExecutionToTimeOutTimer(&incomingTimeOutChannelCommand)
}

// Remove TestInstructionExecution from TimeOut-timer
func (testInstructionExecutionTimeOutEngineObject *TestInstructionTimeOutEngineObjectStruct) removeTestInstructionExecutionFromTimeOutTimer(incomingTimeOutChannelCommand TimeOutChannelCommandStruct) {

}

// Check if TestInstructionExecution exists within TimeOut-timer
func (testInstructionExecutionTimeOutEngineObject *TestInstructionTimeOutEngineObjectStruct) existsTestInstructionExecutionInTimeOutTimer(incomingTimeOutChannelCommand TimeOutChannelCommandStruct) {

}
