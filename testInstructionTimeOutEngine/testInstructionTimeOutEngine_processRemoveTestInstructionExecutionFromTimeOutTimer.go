package testInstructionTimeOutEngine

import (
	"FenixExecutionServer/common_config"
	"github.com/sirupsen/logrus"
	"strconv"
	"time"
)

// Add TestInstructionExecution to TimeOut-timer
func (testInstructionExecutionTimeOutEngineObject *TestInstructionTimeOutEngineObjectStruct) processRemoveTestInstructionExecutionFromTimeOutTimer(
	executionTrack int,
	incomingTimeOutChannelCommand *common_config.TimeOutChannelCommandStruct,
	timeOutChannelCommand common_config.TimeOutChannelCommandType) {

	common_config.Logger.WithFields(logrus.Fields{
		"id":                            "265c6141-c3be-43a8-a4cf-970435b2eaa3",
		"incomingTimeOutChannelCommand": incomingTimeOutChannelCommand,
		"executionTrack":                executionTrack,
	}).Debug("Incoming 'processRemoveTestInstructionExecutionFromTimeOutTimer'")

	defer common_config.Logger.WithFields(logrus.Fields{
		"id": "69054858-a8d5-4c1d-ad11-63c1d37453f4",
	}).Debug("Outgoing 'processRemoveTestInstructionExecutionFromTimeOutTimer'")

	// Create Map-key for 'timeOutMap'
	var timeOutMapKeyToRemove string

	var testInstructionExecutionVersionAsString string
	testInstructionExecutionVersionAsString = strconv.Itoa(
		int(incomingTimeOutChannelCommand.TimeOutChannelTestInstructionExecutions.TestInstructionExecutionVersion))

	timeOutMapKeyToRemove = incomingTimeOutChannelCommand.TimeOutChannelTestInstructionExecutions.TestInstructionExecutionUuid +
		testInstructionExecutionVersionAsString

	// Object to be removed from TimeOut-timer
	var currentTimeOutObjectToRemove *timeOutMapStruct
	var existsInMap bool

	// Extract object from TimeOut.timerMap
	currentTimeOutObjectToRemove, existsInMap = (*timeOutMapSlice[executionTrack])[timeOutMapKeyToRemove]
	if existsInMap == false {

		// Check if Remove-command comes from that an TestInstructionExecution was done or
		// if it was triggered by the TimeOut-timer
		if timeOutChannelCommand == common_config.TimeOutChannelCommandRemoveTestInstructionExecutionFromTimeOutTimerDueToExecutionResult {

			// Only log Warning when Executions is done.
			// TODO fix this so when it times out then it is put in a certain bucket that Execution-Remove will look into when it's not found in main-map
			common_config.Logger.WithFields(logrus.Fields{
				"id":                    "67a83431-0a18-458a-b0a1-032ad21de613",
				"timeOutMapKeyToRemove": timeOutMapKeyToRemove,
				"timeOutChannelCommand": timeOutChannelCommand,
			}).Warning("'timeOutMap' doesn't contain the object to be removed")

		} else {

			// Check if Timer already TimedOut
			_, existsInMap = (*timedOutMapSlice[executionTrack])[timeOutMapKeyToRemove]
			if existsInMap == false {
				// log Error when the TimeOut occurred
				common_config.Logger.WithFields(logrus.Fields{
					"id":                    "ce9d00d7-e245-4687-bca6-028f2e059dc9",
					"timeOutMapKeyToRemove": timeOutMapKeyToRemove,
					"timeOutChannelCommand": timeOutChannelCommand,
				}).Error("'timeOutMap' doesn't contain the object to be removed")
			}
		}

		//errorId := "c7e4917d-57d1-4a07-9c8e-0a793aa6174b"
		//err = errors.New(fmt.Sprintf("'timeOutMap' doesn't contain the object, '%s' to be removed [ErroId: %s]", timeOutMapKeyToRemove, errorId))

		return
	}

	// If timer TimedOut then add object to map with TimedOut Objects
	if timeOutChannelCommand == common_config.TimeOutChannelCommandRemoveTestInstructionExecutionFromTimeOutTimerDueToTimeOutFromTimer {

		_, existsInMap = (*timedOutMapSlice[executionTrack])[timeOutMapKeyToRemove]
		if existsInMap == true {
			common_config.Logger.WithFields(logrus.Fields{
				"id":                    "54f95fa7-f016-4c19-8232-40be29124a6b",
				"timeOutMapKeyToRemove": timeOutMapKeyToRemove,
				"timeOutChannelCommand": timeOutChannelCommand,
			}).Error("'timedOutMap' already contain the object")

			return
		}

		// Add Object to Map with the Timed out Objects
		(*timedOutMapSlice[executionTrack])[timeOutMapKeyToRemove] = currentTimeOutObjectToRemove

	}

	// If the map only consist of 1 object then just remove it
	if len((*timeOutMapSlice[executionTrack])) == 1 {

		// Only Cancel Timer when TestInstructionExecution was finished, otherwise Timer has finished
		if timeOutChannelCommand == common_config.TimeOutChannelCommandRemoveTestInstructionExecutionFromTimeOutTimerDueToExecutionResult {

			// Cancel timer
			currentTimeOutObjectToRemove.cancellableTimer.Cancel()

			// Wait for Cancel timer is complete
			<-currentTimeOutObjectToRemove.cancellableTimer.TimerHasBeenClosed

		}

		// Delete  current object from map
		delete((*timeOutMapSlice[executionTrack]), timeOutMapKeyToRemove)

		// Clear reference to first MapKey
		nextUpcomingObjectMapKeyWithTimeOutSlice[executionTrack] = ""

		return
	}

	// Extract Previous and Next MapKey
	var (
		previousTimeOutMapKey string
		nextTimeOutMapKey     string
	)

	previousTimeOutMapKey = currentTimeOutObjectToRemove.previousTimeOutMapKey
	nextTimeOutMapKey = currentTimeOutObjectToRemove.nextTimeOutMapKey

	// Current object is the first object
	if previousTimeOutMapKey == timeOutMapKeyToRemove &&
		timeOutMapKeyToRemove != nextTimeOutMapKey {

		// Only Cancel Timer when TestInstructionExecution was finished, otherwise Timer has finished
		if timeOutChannelCommand == common_config.TimeOutChannelCommandRemoveTestInstructionExecutionFromTimeOutTimerDueToExecutionResult {

			// Cancel timer
			currentTimeOutObjectToRemove.cancellableTimer.Cancel()

			// Wait for Cancel timer to be complete
			<-currentTimeOutObjectToRemove.cancellableTimer.TimerHasBeenClosed

		}
		// Delete current object
		delete((*timeOutMapSlice[executionTrack]), timeOutMapKeyToRemove)

		// Set new reference to first MapKey
		nextUpcomingObjectMapKeyWithTimeOutSlice[executionTrack] = nextTimeOutMapKey

		// Extract newTimeOutMapObject
		var nextTimeOutMapObject *timeOutMapStruct
		nextTimeOutMapObject, existsInMap = (*timeOutMapSlice[executionTrack])[nextTimeOutMapKey]
		if existsInMap == false {

			common_config.Logger.WithFields(logrus.Fields{
				"id":                "78bc4963-daf1-4624-89d6-dbdba32e5d31",
				"nextTimeOutMapKey": nextTimeOutMapKey,
			}).Error("'timeOutMap' doesn't contain the next object in line")

			return

		}

		// Extract TimeOut-time
		var timeOutTime time.Time
		timeOutTime = nextTimeOutMapObject.currentTimeOutChannelCommandObject.TimeOutChannelTestInstructionExecutions.TimeOutTime

		// Store TimeOut-time in 'incomingTimeOutChannelCommand' because for Remove that information doesn't exist in that object
		incomingTimeOutChannelCommand.TimeOutChannelTestInstructionExecutions.TimeOutTime = timeOutTime

		// Start new TimeOut-timer for next TestInstructionExecution
		go testInstructionExecutionTimeOutEngineObject.startTimeOutTimerTestInstructionExecution(
			executionTrack,
			nextTimeOutMapObject,
			incomingTimeOutChannelCommand,
			nextTimeOutMapKey,
			"082203e6-0136-4d03-a37e-8b0913d8add3",
		)

		// Update next object regarding previous object MapKey
		_ = testInstructionExecutionTimeOutEngineObject.updateTimeOutChannelCommandObject(
			executionTrack,
			nextTimeOutMapKey,
			nextTimeOutMapKey,
			"")

		return
	}

	// Check if the object, to be deleted, is between two objects
	if previousTimeOutMapKey != timeOutMapKeyToRemove &&
		timeOutMapKeyToRemove != nextTimeOutMapKey {

		/* Timer only exist if it is the first object
		// Cancel timer
		currentTimeOutObjectToRemove.cancellableTimer.Cancel()

		// Wait for Cancel timer is complete
		<-currentTimeOutObjectToRemove.cancellableTimer.TimerHasBeenClosed
		*/
		// Delete current object
		delete((*timeOutMapSlice[executionTrack]), timeOutMapKeyToRemove)

		// Update previous object regarding its next object
		_ = testInstructionExecutionTimeOutEngineObject.updateTimeOutChannelCommandObject(
			executionTrack,
			"",
			previousTimeOutMapKey,
			nextTimeOutMapKey)

		// Update next object regarding its previous object
		_ = testInstructionExecutionTimeOutEngineObject.updateTimeOutChannelCommandObject(
			executionTrack,
			previousTimeOutMapKey,
			nextTimeOutMapKey,
			"")

		return
	}

	// Check if the object, to be deleted, is last objects
	if timeOutMapKeyToRemove == nextTimeOutMapKey {

		/* Timer only exist if it is the first object
		// Cancel timer
		currentTimeOutObjectToRemove.cancellableTimer.Cancel()

		// Wait for Cancel timer is complete
		<-currentTimeOutObjectToRemove.cancellableTimer.TimerHasBeenClosed
		*/

		// Delete current object
		delete((*timeOutMapSlice[executionTrack]), timeOutMapKeyToRemove)

		// Update previous object regarding its next object
		_ = testInstructionExecutionTimeOutEngineObject.updateTimeOutChannelCommandObject(
			executionTrack,
			"",
			previousTimeOutMapKey,
			previousTimeOutMapKey)

		return
	}

}
