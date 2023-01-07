package testInstructionTimeOutEngine

import (
	"FenixExecutionServer/common_config"
	"github.com/sirupsen/logrus"
	"strconv"
)

// Remove Allocation for a Timer
func (testInstructionExecutionTimeOutEngineObject *TestInstructionTimeOutEngineObjectStruct) processRemoveAllocationForTestInstructionExecutionToTimeOutTimer(
	executionTrack int,
	incomingTimeOutChannelCommand *common_config.TimeOutChannelCommandStruct) {

	common_config.Logger.WithFields(logrus.Fields{
		"id":                            "74f6be2e-41cc-4e8f-916b-9e6a5c2a82f4",
		"incomingTimeOutChannelCommand": incomingTimeOutChannelCommand,
		"executionTrack":                executionTrack,
	}).Debug("Incoming 'processRemoveAllocationForTestInstructionExecutionToTimeOutTimer'")

	defer common_config.Logger.WithFields(logrus.Fields{
		"id": "7b21734f-d944-49ca-bf83-4079c44d05d9",
	}).Debug("Outgoing 'processRemoveAllocationForTestInstructionExecutionToTimeOutTimer'")

	// Create Map-key for 'timedOutMap'
	var allocatedTimeOutTimerMapKey string
	var existInMap bool

	var testInstructionExecutionVersionAsString string
	testInstructionExecutionVersionAsString = strconv.Itoa(
		int(incomingTimeOutChannelCommand.TimeOutChannelTestInstructionExecutions.TestInstructionExecutionVersion))

	allocatedTimeOutTimerMapKey = incomingTimeOutChannelCommand.TimeOutChannelTestInstructionExecutions.TestInstructionExecutionUuid +
		testInstructionExecutionVersionAsString

	_, existInMap = (AllocatedTimeOutTimerMapSlice[executionTrack])[allocatedTimeOutTimerMapKey]

	// There should be an existing TimeOut-timer allocated
	if existInMap == false {
		common_config.Logger.WithFields(logrus.Fields{
			"id":                          "41698ed4-dae0-428b-ab89-2801527dbd2b",
			"allocatedTimeOutTimerMapKey": allocatedTimeOutTimerMapKey,
		}).Error("An TimeOut-timer was NOT allocated which should not happen")

		return
	}

	// Remove Allocation from allocation-map for TimeOut-timers
	delete(AllocatedTimeOutTimerMapSlice[executionTrack], allocatedTimeOutTimerMapKey)

}
