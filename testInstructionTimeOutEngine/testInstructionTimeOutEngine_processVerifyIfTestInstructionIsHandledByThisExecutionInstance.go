package testInstructionTimeOutEngine

import (
	"FenixExecutionServer/common_config"
	"github.com/sirupsen/logrus"
	"strconv"
)

// Verify if this executionServer-instance is the one responsible for incoming TestInstructionExecution
func (testInstructionExecutionTimeOutEngineObject *TestInstructionTimeOutEngineObjectStruct) processVerifyIfTestInstructionIsHandledByThisExecutionInstance(
	executionTrack int,
	incomingTimeOutChannelCommand *common_config.TimeOutChannelCommandStruct) {

	common_config.Logger.WithFields(logrus.Fields{
		"id":                            "18a05d8c-de9b-453d-999f-d584e90418ee",
		"incomingTimeOutChannelCommand": incomingTimeOutChannelCommand,
		"executionTrack":                executionTrack,
	}).Debug("Incoming 'processVerifyIfTestInstructionIsHandledByThisExecutionInstance'")

	defer common_config.Logger.WithFields(logrus.Fields{
		"id": "f55cca0d-4250-4152-b7db-89f775cac51b",
	}).Debug("Outgoing 'processVerifyIfTestInstructionIsHandledByThisExecutionInstance'")

	// Create Map-key for 'timedOutMap'
	var timeOutMapKey string

	var testInstructionExecutionVersionAsString string
	testInstructionExecutionVersionAsString = strconv.Itoa(
		int(incomingTimeOutChannelCommand.TimeOutChannelTestInstructionExecutions.TestInstructionExecutionVersion))

	timeOutMapKey = incomingTimeOutChannelCommand.TimeOutChannelTestInstructionExecutions.TestInstructionExecutionUuid +
		testInstructionExecutionVersionAsString

	// Object to be removed from TimeOut-timer
	var existsInMap bool

	// Check if TestInstructionExecution exits in map for allocations
	_, existsInMap = (AllocatedTimeOutTimerMapSlice[executionTrack])[timeOutMapKey]
	if existsInMap == true {

		common_config.Logger.WithFields(logrus.Fields{
			"id":             "e7836cbc-26a4-46be-8033-9ec7e69f2205",
			"executionTrack": executionTrack,
			"timeOutMapKey":  timeOutMapKey,
		}).Debug("Timer found in 'AllocatedTimeOutTimerMapSlice', so this instance is responsible for this TestInstructionExecution")

		// TestInstructionExecution is handled by this Execution-instance
		var testInstructionIsHandledByThisInstanceResponse common_config.TimeOutResponseChannelForVerifyIfTestInstructionIsHandledByThisInstanceStruct
		testInstructionIsHandledByThisInstanceResponse = common_config.TimeOutResponseChannelForVerifyIfTestInstructionIsHandledByThisInstanceStruct{
			TestInstructionIsHandledByThisExecutionInstance: true}

		// Send response on response channel
		*incomingTimeOutChannelCommand.TimeOutResponseChannelForVerifyIfTestInstructionIsHandledByThisInstance <- testInstructionIsHandledByThisInstanceResponse

		return
	}

	// Check if TestInstructionExecution exits in map for already timedOut TestInstructionExecutions
	_, existsInMap = (*timedOutMapSlice[executionTrack])[timeOutMapKey]
	if existsInMap == true {

		common_config.Logger.WithFields(logrus.Fields{
			"Id":            "2518ffb3-6dd4-4e8b-ac45-be80e63c5a42",
			"timeOutMapKey": timeOutMapKey,
		}).Debug("TestInstructionObject has already TimedOut, so this instance is responsible for this TestInstructionExecution")

		// TestInstructionExecution is handled by this Execution-instance
		var testInstructionIsHandledByThisInstanceResponse common_config.TimeOutResponseChannelForVerifyIfTestInstructionIsHandledByThisInstanceStruct
		testInstructionIsHandledByThisInstanceResponse = common_config.TimeOutResponseChannelForVerifyIfTestInstructionIsHandledByThisInstanceStruct{
			TestInstructionIsHandledByThisExecutionInstance: true}

		// Send response on response channel
		*incomingTimeOutChannelCommand.TimeOutResponseChannelForVerifyIfTestInstructionIsHandledByThisInstance <- testInstructionIsHandledByThisInstanceResponse

		return
	}

	// Check if TestInstructionExecution exists in timeOut-map
	_, existsInMap = (*timeOutMapSlice[executionTrack])[timeOutMapKey]
	if existsInMap == true {

		common_config.Logger.WithFields(logrus.Fields{
			"Id":            "6a7efe83-5ebd-4122-b870-e7cce13324c9",
			"timeOutMapKey": timeOutMapKey,
		}).Debug("TestInstructionObject exists in TimedOut-amp, so this instance is responsible for this TestInstructionExecution")

		// TestInstructionExecution is handled by this Execution-instance
		var testInstructionIsHandledByThisInstanceResponse common_config.TimeOutResponseChannelForVerifyIfTestInstructionIsHandledByThisInstanceStruct
		testInstructionIsHandledByThisInstanceResponse = common_config.TimeOutResponseChannelForVerifyIfTestInstructionIsHandledByThisInstanceStruct{
			TestInstructionIsHandledByThisExecutionInstance: true}

		// Send response on response channel
		*incomingTimeOutChannelCommand.TimeOutResponseChannelForVerifyIfTestInstructionIsHandledByThisInstance <- testInstructionIsHandledByThisInstanceResponse

		return
	}

	// TestInstructionExecution is not handled by this Execution-instance
	common_config.Logger.WithFields(logrus.Fields{
		"Id":            "0c909976-7319-416a-b57e-265d9215c9c4",
		"timeOutMapKey": timeOutMapKey,
	}).Debug("TestInstructionExecution is not handled by this Execution-instance")

	// TestInstructionExecution is handled by this Execution-instance
	var testInstructionIsHandledByThisInstanceResponse common_config.TimeOutResponseChannelForVerifyIfTestInstructionIsHandledByThisInstanceStruct
	testInstructionIsHandledByThisInstanceResponse = common_config.TimeOutResponseChannelForVerifyIfTestInstructionIsHandledByThisInstanceStruct{
		TestInstructionIsHandledByThisExecutionInstance: false}

	// Send response on response channel
	*incomingTimeOutChannelCommand.TimeOutResponseChannelForVerifyIfTestInstructionIsHandledByThisInstance <- testInstructionIsHandledByThisInstanceResponse

	return

}
