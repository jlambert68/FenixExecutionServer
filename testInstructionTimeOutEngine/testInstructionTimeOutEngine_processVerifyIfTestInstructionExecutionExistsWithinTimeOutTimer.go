package testInstructionTimeOutEngine

import (
	"FenixExecutionServer/common_config"
	"github.com/sirupsen/logrus"
	"strconv"
)

// Check if the TestInstructionExecution is handled by this ExecutionServer-instance.
func (testInstructionExecutionTimeOutEngineObject *TestInstructionTimeOutEngineObjectStruct) processVerifyIfTestInstructionExecutionExistsWithinTimeOutTimer(
	executionTrack int,
	incomingTimeOutChannelCommand *common_config.TimeOutChannelCommandStruct) {

	common_config.Logger.WithFields(logrus.Fields{
		"id":                            "97fa771c-27c4-41fc-aa2e-48f4fab7b188",
		"incomingTimeOutChannelCommand": incomingTimeOutChannelCommand,
		"executionTrack":                executionTrack,
	}).Debug("Incoming 'processVerifyIfTestInstructionExecutionExistsWithinTimeOutTimer'")

	defer common_config.Logger.WithFields(logrus.Fields{
		"id": "4b30ce5b-1654-4eda-9274-f892b1372ee4",
	}).Debug("Outgoing 'processVerifyIfTestInstructionExecutionExistsWithinTimeOutTimer'")

	// Create Map-key for 'timedOutMap'
	var allocatedTimeOutTimerMapKey string
	var existInMap bool

	var testInstructionExecutionVersionAsString string
	testInstructionExecutionVersionAsString = strconv.Itoa(
		int(incomingTimeOutChannelCommand.TimeOutChannelTestInstructionExecutions.TestInstructionExecutionVersion))

	allocatedTimeOutTimerMapKey = incomingTimeOutChannelCommand.TimeOutChannelTestInstructionExecutions.TestInstructionExecutionUuid +
		testInstructionExecutionVersionAsString

	_, existInMap = (AllocatedTimeOutTimerMapSlice[executionTrack])[allocatedTimeOutTimerMapKey]

	// There should not be any existing TimeOut-timer allocated
	if existInMap == true {
		common_config.Logger.WithFields(logrus.Fields{
			"id":                          "7ac3686c-f5d1-45de-8c90-c7325193c0b4",
			"allocatedTimeOutTimerMapKey": allocatedTimeOutTimerMapKey,
		}).Debug("An TimeOut-timer has already been allocated which shouldn't happen")

		return
	}

	// Add to allocation-map for TimeOut-timers
	(AllocatedTimeOutTimerMapSlice[executionTrack])[allocatedTimeOutTimerMapKey] = allocatedTimeOutTimerMapKey

}
