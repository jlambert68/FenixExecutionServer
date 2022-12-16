package testInstructionTimeOutEngine

import (
	"FenixExecutionServer/common_config"
	"github.com/sirupsen/logrus"
	"strconv"
)

// Allocate a Timer before starting it. Used to handle really fast responses for TestInstructionExecutions so stuff doesn't happen in wrong order
func (testInstructionExecutionTimeOutEngineObject *TestInstructionTimeOutEngineObjectStruct) processAllocateTestInstructionExecutionToTimeOutTimer(
	incomingTimeOutChannelCommand *common_config.TimeOutChannelCommandStruct) {

	common_config.Logger.WithFields(logrus.Fields{
		"id":                            "2118f722-4d0e-4ae0-8c30-3e0de37b312a",
		"incomingTimeOutChannelCommand": incomingTimeOutChannelCommand,
	}).Debug("Incoming 'processAllocateTestInstructionExecutionToTimeOutTimer'")

	defer common_config.Logger.WithFields(logrus.Fields{
		"id": "772d4001-f5a8-45e4-8515-03fd0301476a",
	}).Debug("Outgoing 'processAllocateTestInstructionExecutionToTimeOutTimer'")

	// Create Map-key for 'timedOutMap'
	var allocatedTimeOutTimerMapKey string
	var existInMap bool

	var testInstructionExecutionVersionAsString string
	testInstructionExecutionVersionAsString = strconv.Itoa(
		int(incomingTimeOutChannelCommand.TimeOutChannelTestInstructionExecutions.TestInstructionExecutionVersion))

	allocatedTimeOutTimerMapKey = incomingTimeOutChannelCommand.TimeOutChannelTestInstructionExecutions.TestInstructionExecutionUuid +
		testInstructionExecutionVersionAsString

	_, existInMap = allocatedTimeOutTimerMap[allocatedTimeOutTimerMapKey]

	// There should not be any existing TimeOut-timer allocated
	if existInMap == true {
		common_config.Logger.WithFields(logrus.Fields{
			"id":                          "7ac3686c-f5d1-45de-8c90-c7325193c0b4",
			"allocatedTimeOutTimerMapKey": allocatedTimeOutTimerMapKey,
		}).Debug("An TimeOut-timer has already been allocated which shouldn't happen")

		return
	}

	// Add to allocation-map for TimeOut-timers
	allocatedTimeOutTimerMap[allocatedTimeOutTimerMapKey] = allocatedTimeOutTimerMapKey

}
