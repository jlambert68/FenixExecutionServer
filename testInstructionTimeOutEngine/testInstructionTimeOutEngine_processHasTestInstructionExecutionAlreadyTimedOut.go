package testInstructionTimeOutEngine

import (
	"FenixExecutionServer/common_config"
	"github.com/sirupsen/logrus"
	"strconv"
	"time"
)

// Add TestInstructionExecution to TimeOut-timer
func (testInstructionExecutionTimeOutEngineObject *TestInstructionTimeOutEngineObjectStruct) processHasTestInstructionExecutionAlreadyTimedOut(
	executionTrack int,
	incomingTimeOutChannelCommand *common_config.TimeOutChannelCommandStruct) {

	common_config.Logger.WithFields(logrus.Fields{
		"id":                            "5297b508-3d00-4930-b1b5-0e75fe0dcc45",
		"incomingTimeOutChannelCommand": incomingTimeOutChannelCommand,
		"executionTrack":                executionTrack,
	}).Debug("Incoming 'processHasTestInstructionExecutionAlreadyTimedOut'")

	defer common_config.Logger.WithFields(logrus.Fields{
		"id": "772d4001-f5a8-45e4-8515-03fd0301476a",
	}).Debug("Outgoing 'processHasTestInstructionExecutionAlreadyTimedOut'")

	// Create Map-key for 'timedOutMap'
	var timeOutdMapKey string

	var testInstructionExecutionVersionAsString string
	testInstructionExecutionVersionAsString = strconv.Itoa(
		int(incomingTimeOutChannelCommand.TimeOutChannelTestInstructionExecutions.TestInstructionExecutionVersion))

	timeOutdMapKey = incomingTimeOutChannelCommand.TimeOutChannelTestInstructionExecutions.TestInstructionExecutionUuid +
		testInstructionExecutionVersionAsString

	// Object to be removed from TimeOut-timer
	var currentTimeOutdObject *timeOutMapStruct
	var existsInMap bool

	// If there are an ongoing allocation then the Timer hasn't started yet so then NO Timer has TimedOut
	_, existsInMap = (AllocatedTimeOutTimerMapSlice[executionTrack])[timeOutdMapKey]
	if existsInMap == true {

		// TestInstructionExecution hasn't timed out due to Timer  hasn't started yet
		var timedOutResponse common_config.TimeOutResponseChannelForTimeOutHasOccurredStruct
		timedOutResponse = common_config.TimeOutResponseChannelForTimeOutHasOccurredStruct{
			TimeOutWasTriggered: false}

		// Send response on response channel
		*incomingTimeOutChannelCommand.TimeOutReturnChannelForTimeOutHasOccurred <- timedOutResponse

		return
	}

	// Check if TestInstructionExecution has already timedOut
	currentTimeOutdObject, existsInMap = (*timedOutMapSlice[executionTrack])[timeOutdMapKey]
	if existsInMap == true {

		common_config.Logger.WithFields(logrus.Fields{
			"Id":             "3ff8adb7-f3d3-4bf3-8c05-8a0711cf7bcd",
			"timeOutdMapKey": timeOutdMapKey,
		}).Debug("TestInstructionObject has already TimedOut")

		// TestInstructionExecution has already timed out so create response to caller
		var timedOutResponse common_config.TimeOutResponseChannelForTimeOutHasOccurredStruct
		timedOutResponse = common_config.TimeOutResponseChannelForTimeOutHasOccurredStruct{
			TimeOutWasTriggered: true}

		// Send response on response channel
		*incomingTimeOutChannelCommand.TimeOutReturnChannelForTimeOutHasOccurred <- timedOutResponse

		return
	}

	// Get current TestInstructionExecution from timeOut-map
	currentTimeOutdObject, existsInMap = (*timeOutMapSlice[executionTrack])[timeOutdMapKey]
	if existsInMap == false {

		common_config.Logger.WithFields(logrus.Fields{
			"Id":             "25124f1b-6402-4eb2-a27b-e3b41798a952",
			"timeOutdMapKey": timeOutdMapKey,
		}).Error("couldn't find the TestInstructionObject in TimeOut-map, something is very wrong")

		// Sending is over channel is not necessary, but I will keep the program running to have more log data.
		// The most correct would be to end program, but....
		// Set that TestInstructionExecution has already timed out so create response to caller
		var timedOutResponse common_config.TimeOutResponseChannelForTimeOutHasOccurredStruct
		timedOutResponse = common_config.TimeOutResponseChannelForTimeOutHasOccurredStruct{
			TimeOutWasTriggered: true}

		// Send response on response channel
		*incomingTimeOutChannelCommand.TimeOutReturnChannelForTimeOutHasOccurred <- timedOutResponse

		return
	}

	// If 'cancellableTimer' = nil and 'timeOutdMapKey' != 'currentTimeOutdObject.currentTimeOutMapKey'
	// then this is not the first object in line to TimeOut, just return that it has not TimedOut yet
	if currentTimeOutdObject.cancellableTimer == nil &&
		currentTimeOutdObject.currentTimeOutMapKey != nextUpcomingObjectMapKeyWithTimeOutSlice[executionTrack] {

		var timedOutResponse common_config.TimeOutResponseChannelForTimeOutHasOccurredStruct
		timedOutResponse = common_config.TimeOutResponseChannelForTimeOutHasOccurredStruct{
			TimeOutWasTriggered: false}

		// Send response on response channel
		*incomingTimeOutChannelCommand.TimeOutReturnChannelForTimeOutHasOccurred <- timedOutResponse

		return
	}

	// Nothing had timed outs so extract when Timer will time out
	var durationToTimeOut time.Duration
	durationToTimeOut, _ = currentTimeOutdObject.cancellableTimer.WhenWillTimerTimeOut()

	// remove 25% of TimerOut-time safety margin
	var timeDurationUntilTimerSignal time.Duration
	timeDurationUntilTimerSignal = durationToTimeOut - extractTimerMarginalBeforeTimeOut_25percent

	// When 'timeDurationUntilTimerSignal' gets negative then the Timer is close to Time out and therefor counts as a TimeOut
	if timeDurationUntilTimerSignal < 0 {

		// TestInstructionExecution has, or is close to, time out so create response to caller
		var timedOutResponse common_config.TimeOutResponseChannelForTimeOutHasOccurredStruct
		timedOutResponse = common_config.TimeOutResponseChannelForTimeOutHasOccurredStruct{
			TimeOutWasTriggered: true}

		// Send response on response channel
		*incomingTimeOutChannelCommand.TimeOutReturnChannelForTimeOutHasOccurred <- timedOutResponse

		return
	}

	// TestInstructionExecution has NOT timed out so create response to caller
	var timedOutResponse common_config.TimeOutResponseChannelForTimeOutHasOccurredStruct
	timedOutResponse = common_config.TimeOutResponseChannelForTimeOutHasOccurredStruct{
		TimeOutWasTriggered: false}

	// Send response on response channel
	*incomingTimeOutChannelCommand.TimeOutReturnChannelForTimeOutHasOccurred <- timedOutResponse

	return
}
