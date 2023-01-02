package testInstructionTimeOutEngine

import (
	"FenixExecutionServer/common_config"
	"github.com/sirupsen/logrus"
	"time"
)

// Check the duration until next TimeOut-timer to fire
func (testInstructionExecutionTimeOutEngineObject *TestInstructionTimeOutEngineObjectStruct) processTimeUntilNextTimeOutTimerToFires(
	executionTrack int,
	incomingTimeOutChannelCommand *common_config.TimeOutChannelCommandStruct) {

	common_config.Logger.WithFields(logrus.Fields{
		"id":                            "4d655ea6-ffba-45bb-91bd-b23ea3859cae",
		"incomingTimeOutChannelCommand": incomingTimeOutChannelCommand.TimeOutChannelCommand,
		"executionTrack":                executionTrack,
	}).Debug("Incoming 'processTimeUntilNextTimeOutTimerToFire'")

	defer common_config.Logger.WithFields(logrus.Fields{
		"id": "7801069e-565f-418a-8a4b-c239291e7d71",
	}).Debug("Outgoing 'processTimeUntilNextTimeOutTimerToFire'")

	// Create Map-key for 'timedOutMap' for next upcoming timeout
	var timeOutdMapKey string
	timeOutdMapKey = nextUpcomingObjectMapKeyWithTimeOutSlice[executionTrack]

	// When there is no TimeOut-timer for this track
	if timeOutdMapKey == "" {

		// Create response to caller
		var timedOutResponse common_config.TimeOutResponseChannelForDurationUntilTimeOutOccursStruct
		timedOutResponse = common_config.TimeOutResponseChannelForDurationUntilTimeOutOccursStruct{
			DurationUntilTimeOutOccurs: time.Duration(-1)}

		// Send response on response channel
		*incomingTimeOutChannelCommand.TimeOutResponseChannelForDurationUntilTimeOutOccurs <- timedOutResponse

		return
	}

	// Object to be removed from TimeOut-timer
	var currentTimeOutdObject *timeOutMapStruct
	var existsInMap bool

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
		var timedOutResponse common_config.TimeOutResponseChannelForDurationUntilTimeOutOccursStruct
		timedOutResponse = common_config.TimeOutResponseChannelForDurationUntilTimeOutOccursStruct{
			DurationUntilTimeOutOccurs: time.Duration(-1)}

		// Send response on response channel
		*incomingTimeOutChannelCommand.TimeOutResponseChannelForDurationUntilTimeOutOccurs <- timedOutResponse

		return
	}

	// Extract when Timer will time out
	var durationToTimeOut time.Duration
	durationToTimeOut, _ = currentTimeOutdObject.cancellableTimer.WhenWillTimerTimeOut()

	// Create response to caller
	var timedOutResponse common_config.TimeOutResponseChannelForDurationUntilTimeOutOccursStruct
	timedOutResponse = common_config.TimeOutResponseChannelForDurationUntilTimeOutOccursStruct{
		DurationUntilTimeOutOccurs: durationToTimeOut}

	// Send response on response channel
	*incomingTimeOutChannelCommand.TimeOutResponseChannelForDurationUntilTimeOutOccurs <- timedOutResponse

	return
}
