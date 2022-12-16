package testInstructionTimeOutEngine

import (
	"FenixExecutionServer/common_config"
	"github.com/sirupsen/logrus"
	"strconv"
	"time"
)

// Add TestInstructionExecution to TimeOut-timer
func (testInstructionExecutionTimeOutEngineObject *TestInstructionTimeOutEngineObjectStruct) processHasTestInstructionExecutionAlreadyTimedOut(
	incomingTimeOutChannelCommand *common_config.TimeOutChannelCommandStruct) {

	common_config.Logger.WithFields(logrus.Fields{
		"id":                            "5297b508-3d00-4930-b1b5-0e75fe0dcc45",
		"incomingTimeOutChannelCommand": incomingTimeOutChannelCommand,
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

	// Check if TestInstructionExecution has already timedOut
	currentTimeOutdObject, existsInMap = timedOutMap[timeOutdMapKey]
	if existsInMap == true {

		// TestInstructionExecution has already timed out so create response to caller
		var timedOutResponse common_config.TimeOutResponseChannelForTimeOutHasOccurredStruct
		timedOutResponse = common_config.TimeOutResponseChannelForTimeOutHasOccurredStruct{
			TimeOutWasTriggered: true}

		// Send response on response channel
		*incomingTimeOutChannelCommand.TimeOutReturnChannelForTimeOutHasOccurred <- timedOutResponse
	}

	// Extract when Timer will time out
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
	}

}
