package testInstructionTimeOutEngine

import (
	"FenixExecutionServer/common_config"
	"fmt"
	"github.com/sirupsen/logrus"
	"strconv"
	"time"
)

// Check if TestInstructionExecution already had TimedOut
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

	// When FinalTestInstructionResult is claimed from database due to that other ExecutionServer then the one sent
	// the TestInstructionExecution then no timer is set in this instance
	if incomingTimeOutChannelCommand.MessageReceivedByOtherExecutionServerWasClaimedFromDatabase == true {

		common_config.Logger.WithFields(logrus.Fields{
			"id": "69a5236f-b76f-40da-9b91-7118ad7189c7",
		}).Debug("This TestInstructionExecution was claimed due no other is processing it")

		// TestInstructionExecution hasn't timed out due to Timer was not set
		var timedOutResponse common_config.TimeOutResponseChannelForTimeOutHasOccurredStruct
		timedOutResponse = common_config.TimeOutResponseChannelForTimeOutHasOccurredStruct{
			TimeOutWasTriggered: false}

		// Send response on response channel
		*incomingTimeOutChannelCommand.TimeOutReturnChannelForTimeOutHasOccurred <- timedOutResponse

		return
	}

	// Create Map-key for 'timedOutMap'
	var timeOutMapKey string

	var testInstructionExecutionVersionAsString string
	testInstructionExecutionVersionAsString = strconv.Itoa(
		int(incomingTimeOutChannelCommand.TimeOutChannelTestInstructionExecutions.TestInstructionExecutionVersion))

	timeOutMapKey = incomingTimeOutChannelCommand.TimeOutChannelTestInstructionExecutions.TestInstructionExecutionUuid +
		testInstructionExecutionVersionAsString

	// Object to be removed from TimeOut-timer
	var existsInMap bool

	// If there are an ongoing allocation then the Timer hasn't started yet so then NO Timer has TimedOut
	_, existsInMap = (AllocatedTimeOutTimerMapSlice[executionTrack])[timeOutMapKey]
	if existsInMap == true {

		common_config.Logger.WithFields(logrus.Fields{
			"id":             "373a1899-145d-451e-acc4-e6c5bbf516fe",
			"executionTrack": executionTrack,
			"timeOutMapKey":  timeOutMapKey,
		}).Debug("Timer found in 'AllocatedTimeOutTimerMapSlice' so Timer can't be TimedOut yet")

		// TestInstructionExecution hasn't timed out due to Timer hasn't started yet
		var timedOutResponse common_config.TimeOutResponseChannelForTimeOutHasOccurredStruct
		timedOutResponse = common_config.TimeOutResponseChannelForTimeOutHasOccurredStruct{
			TimeOutWasTriggered: false}

		// Send response on response channel
		*incomingTimeOutChannelCommand.TimeOutReturnChannelForTimeOutHasOccurred <- timedOutResponse

		return
	}

	// Check if TestInstructionExecution has already timedOut
	_, existsInMap = (*timedOutMapSlice[executionTrack])[timeOutMapKey]
	if existsInMap == true {

		common_config.Logger.WithFields(logrus.Fields{
			"Id":            "3ff8adb7-f3d3-4bf3-8c05-8a0711cf7bcd",
			"timeOutMapKey": timeOutMapKey,
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
	var currentTimeOutObject *timeOutMapStruct
	currentTimeOutObject, existsInMap = (*timeOutMapSlice[executionTrack])[timeOutMapKey]
	if existsInMap == false {

		common_config.Logger.WithFields(logrus.Fields{
			"Id":            "25124f1b-6402-4eb2-a27b-e3b41798a952",
			"timeOutMapKey": timeOutMapKey,
		}).Error("couldn't find the TestInstructionObject in TimeOut-map, something is very wrong")

		for timeOutMapSliceCounter, timeOutMap := range timeOutMapSlice {
			for timeOutMapKey, timeOutMapObject := range *timeOutMap {

				timeOutMapValues := fmt.Sprintf(
					"timeOutMapSliceCounter:%d, "+
						"timeOutMapKey:%s, "+
						"timeOutMapObject.previousTimeOutMapKey:%s, "+
						"timeOutMapObject.currentTimeOutMapKey:%s, "+
						"timeOutMapObject.nextTimeOutMapKey:%s",
					timeOutMapSliceCounter,
					timeOutMapKey,
					timeOutMapObject.previousTimeOutMapKey,
					timeOutMapObject.currentTimeOutMapKey,
					timeOutMapObject.nextTimeOutMapKey)
				fmt.Print(timeOutMapValues)
			}
		}

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

	// If 'cancellableTimer' = nil and 'timeOutMapKey' != 'currentTimeOutObject.currentTimeOutMapKey'
	// then this is not the first object in line to TimeOut, just return that it has not TimedOut yet
	if currentTimeOutObject.cancellableTimer == nil &&
		currentTimeOutObject.currentTimeOutMapKey != nextUpcomingObjectMapKeyWithTimeOutSlice[executionTrack] {

		var timedOutResponse common_config.TimeOutResponseChannelForTimeOutHasOccurredStruct
		timedOutResponse = common_config.TimeOutResponseChannelForTimeOutHasOccurredStruct{
			TimeOutWasTriggered: false}

		// Send response on response channel
		*incomingTimeOutChannelCommand.TimeOutReturnChannelForTimeOutHasOccurred <- timedOutResponse

		return
	}

	// Nothing had timed outs so extract when Timer will time out
	var durationToTimeOut time.Duration
	durationToTimeOut, _ = currentTimeOutObject.cancellableTimer.WhenWillTimerTimeOut()

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
