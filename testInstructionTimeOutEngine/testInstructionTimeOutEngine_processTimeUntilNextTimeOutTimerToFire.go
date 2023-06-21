package testInstructionTimeOutEngine

import (
	"FenixExecutionServer/common_config"
	"fmt"
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
			"Id":             "342d8b62-ca31-4459-8ac7-7020ad34a392",
			"timeOutdMapKey": timeOutdMapKey,
		}).Error("couldn't find the TestInstructionObject in TimeOut-map, something is very wrong")

		for timeOutMapSliceCounter, timeOutMap := range timeOutMapSlice {
			for timeOutMapKey, timeOutMapObject := range *timeOutMap {

				timeOutMapValues := fmt.Sprintf(
					"timeOutMapSliceCounter:%s, "+
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
