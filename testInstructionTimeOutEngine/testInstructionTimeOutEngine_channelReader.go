package testInstructionTimeOutEngine

import (
	"FenixExecutionServer/common_config"
	"github.com/sirupsen/logrus"
)

// Channel reader which is used for reading out commands to CommandEngine
func (testInstructionExecutionTimeOutEngineObject *TestInstructionTimeOutEngineObjectStruct) startTimeOutChannelReader(
	executionTrack int) {

	var incomingTimeOutChannelCommand common_config.TimeOutChannelCommandStruct
	var channelSize int

	for {
		// Wait for incoming command over channel
		incomingTimeOutChannelCommand = <-TimeOutChannelEngineCommandChannelSlice[executionTrack]

		common_config.Logger.WithFields(logrus.Fields{
			"Id":                            "0d02712a-d27b-4be4-b168-f2d670010990",
			"incomingTimeOutChannelCommand": incomingTimeOutChannelCommand,
			"TimeOutChannelCommand":         common_config.TimeOutChannelCommandsForDebugPrinting[incomingTimeOutChannelCommand.TimeOutChannelCommand],
			"executionTrack":                executionTrack,
		}).Debug("Message received on 'TimeOutChannel'")

		// If size of Channel > 'timeOutChannelWarningLevel' then log Warning message
		channelSize = len(TimeOutChannelEngineCommandChannelSlice[executionTrack])
		if channelSize > timeOutChannelWarningLevel {
			common_config.Logger.WithFields(logrus.Fields{
				"Id":                         "7dafce7a-eed9-448c-81b6-1015733dc1cb",
				"channelSize":                channelSize,
				"timeOutChannelWarningLevel": timeOutChannelWarningLevel,
				"timeOutChannelSize":         timeOutChannelSize,
				"executionTrack":             executionTrack,
			}).Warning("Number of messages on queue for 'TimeOutChannel' has reached a critical level")
		} else {
			common_config.Logger.WithFields(logrus.Fields{
				"Id":                         "c97f1405-9347-4995-b625-0afc36f6dc04",
				"channelSize":                channelSize,
				"timeOutChannelWarningLevel": timeOutChannelWarningLevel,
				"timeOutChannelSize":         timeOutChannelSize,
				"executionTrack":             executionTrack,
				"incomingTimeOutChannelCommand.TimeOutChannelCommand": incomingTimeOutChannelCommand.TimeOutChannelCommand,
			}).Info("Incoming TimeOutEngine-command")
		}

		switch incomingTimeOutChannelCommand.TimeOutChannelCommand {

		case common_config.TimeOutChannelCommandAddTestInstructionExecutionToTimeOutTimer:
			testInstructionExecutionTimeOutEngineObject.addTestInstructionExecutionToTimeOutTimer(
				executionTrack,
				incomingTimeOutChannelCommand)

		case common_config.TimeOutChannelCommandRemoveTestInstructionExecutionFromTimeOutTimerDueToTimeOutFromTimer:
			testInstructionExecutionTimeOutEngineObject.removeTestInstructionExecutionFromTimeOutTimer(
				executionTrack,
				incomingTimeOutChannelCommand,
				common_config.TimeOutChannelCommandRemoveTestInstructionExecutionFromTimeOutTimerDueToTimeOutFromTimer)

		case common_config.TimeOutChannelCommandRemoveTestInstructionExecutionFromTimeOutTimerDueToExecutionResult:
			testInstructionExecutionTimeOutEngineObject.removeTestInstructionExecutionFromTimeOutTimer(
				executionTrack,
				incomingTimeOutChannelCommand,
				common_config.TimeOutChannelCommandRemoveTestInstructionExecutionFromTimeOutTimerDueToExecutionResult)

		case common_config.TimeOutChannelCommandExistsTestInstructionExecutionInTimeOutTimer:
			testInstructionExecutionTimeOutEngineObject.existsTestInstructionExecutionInTimeOutTimer(
				executionTrack,
				incomingTimeOutChannelCommand)

		case common_config.TimeOutChannelCommandHasTestInstructionExecutionAlreadyTimedOut:
			testInstructionExecutionTimeOutEngineObject.hasTestInstructionExecutionAlreadyTimedOut(
				executionTrack,
				incomingTimeOutChannelCommand)

		case common_config.TimeOutChannelCommandAllocateTestInstructionExecutionToTimeOutTimer:
			testInstructionExecutionTimeOutEngineObject.allocateTestInstructionExecutionToTimeOutTimer(
				executionTrack,
				incomingTimeOutChannelCommand)

		case common_config.TimeOutChannelCommandTimeUntilNextTimeOutTimerToFires:
			testInstructionExecutionTimeOutEngineObject.timeUntilNextTimeOutTimerToFires(
				executionTrack,
				incomingTimeOutChannelCommand)

		case common_config.TimeOutChannelCommandRemoveAllocationForTestInstructionExecutionToTimeOutTimer:
			testInstructionExecutionTimeOutEngineObject.removeAllocationForTestInstructionExecutionToTimeOutTimer(
				executionTrack,
				incomingTimeOutChannelCommand)

		// No other command is supported
		default:
			common_config.Logger.WithFields(logrus.Fields{
				"Id":                     "b18a28a0-7ff9-41fb-a9be-8990e58e4b78",
				"incomingChannelCommand": incomingTimeOutChannelCommand,
				"executionTrack":         executionTrack,
			}).Fatalln("Unknown command in TimeOutCommandChannel for TimeOutEngine")
		}
	}
}

// Add TestInstructionExecution to TimeOut-timer
func (testInstructionExecutionTimeOutEngineObject *TestInstructionTimeOutEngineObjectStruct) addTestInstructionExecutionToTimeOutTimer(
	executionTrack int,
	incomingTimeOutChannelCommand common_config.TimeOutChannelCommandStruct) {

	testInstructionExecutionTimeOutEngineObject.processAddTestInstructionExecutionToTimeOutTimer(
		executionTrack,
		&incomingTimeOutChannelCommand)
}

// Remove TestInstructionExecution from TimeOut-timer
func (testInstructionExecutionTimeOutEngineObject *TestInstructionTimeOutEngineObjectStruct) removeTestInstructionExecutionFromTimeOutTimer(
	executionTrack int,
	incomingTimeOutChannelCommand common_config.TimeOutChannelCommandStruct,
	timeOutChannelCommand common_config.TimeOutChannelCommandType) {

	testInstructionExecutionTimeOutEngineObject.processRemoveTestInstructionExecutionFromTimeOutTimer(
		executionTrack,
		&incomingTimeOutChannelCommand,
		timeOutChannelCommand)
}

// Check if TestInstructionExecution exists within TimeOut-timer
func (testInstructionExecutionTimeOutEngineObject *TestInstructionTimeOutEngineObjectStruct) existsTestInstructionExecutionInTimeOutTimer(
	executionTrack int,
	incomingTimeOutChannelCommand common_config.TimeOutChannelCommandStruct) {

}

// Check if TestInstructionExecution already had TimedOut
func (testInstructionExecutionTimeOutEngineObject *TestInstructionTimeOutEngineObjectStruct) hasTestInstructionExecutionAlreadyTimedOut(
	executionTrack int,
	incomingTimeOutChannelCommand common_config.TimeOutChannelCommandStruct) {

	testInstructionExecutionTimeOutEngineObject.processHasTestInstructionExecutionAlreadyTimedOut(
		executionTrack,
		&incomingTimeOutChannelCommand)

}

// Allocate a Timer before starting it. Used to handle really fast responses for TestInstructionExecutions so stuff doesn't happen in wrong order
func (testInstructionExecutionTimeOutEngineObject *TestInstructionTimeOutEngineObjectStruct) allocateTestInstructionExecutionToTimeOutTimer(
	executionTrack int,
	incomingTimeOutChannelCommand common_config.TimeOutChannelCommandStruct) {

	testInstructionExecutionTimeOutEngineObject.processAllocateTestInstructionExecutionToTimeOutTimer(
		executionTrack,
		&incomingTimeOutChannelCommand)

}

// Check the duration until next TimeOut-timer to fire
func (testInstructionExecutionTimeOutEngineObject *TestInstructionTimeOutEngineObjectStruct) timeUntilNextTimeOutTimerToFires(
	executionTrack int,
	incomingTimeOutChannelCommand common_config.TimeOutChannelCommandStruct) {

	testInstructionExecutionTimeOutEngineObject.processTimeUntilNextTimeOutTimerToFires(
		executionTrack,
		&incomingTimeOutChannelCommand)

}

// Remove Allocation for a Timer.
func (testInstructionExecutionTimeOutEngineObject *TestInstructionTimeOutEngineObjectStruct) removeAllocationForTestInstructionExecutionToTimeOutTimer(
	executionTrack int,
	incomingTimeOutChannelCommand common_config.TimeOutChannelCommandStruct) {

	testInstructionExecutionTimeOutEngineObject.processRemoveAllocationForTestInstructionExecutionToTimeOutTimer(
		executionTrack,
		&incomingTimeOutChannelCommand)

}
