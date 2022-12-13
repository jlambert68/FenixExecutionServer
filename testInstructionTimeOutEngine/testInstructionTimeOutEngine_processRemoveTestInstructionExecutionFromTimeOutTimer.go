package testInstructionTimeOutEngine

import (
	"FenixExecutionServer/common_config"
	"github.com/sirupsen/logrus"
	"strconv"
)

// Add TestInstructionExecution to TimeOut-timer
func (testInstructionExecutionTimeOutEngineObject *TestInstructionTimeOutEngineObjectStruct) processRemoveTestInstructionExecutionToTimeOutTimer(
	incomingTimeOutChannelCommand *common_config.TimeOutChannelCommandStruct) {

	common_config.Logger.WithFields(logrus.Fields{
		"id":                            "265c6141-c3be-43a8-a4cf-970435b2eaa3",
		"incomingTimeOutChannelCommand": incomingTimeOutChannelCommand,
	}).Debug("Incoming 'processRemoveTestInstructionExecutionToTimeOutTimer'")

	defer common_config.Logger.WithFields(logrus.Fields{
		"id": "69054858-a8d5-4c1d-ad11-63c1d37453f4",
	}).Debug("Outgoing 'processRemoveTestInstructionExecutionToTimeOutTimer'")

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
	currentTimeOutObjectToRemove, existsInMap = timeOutMap[timeOutMapKeyToRemove]
	if existsInMap == false {

		common_config.Logger.WithFields(logrus.Fields{
			"id":                    "67a83431-0a18-458a-b0a1-032ad21de613",
			"timeOutMapKeyToRemove": timeOutMapKeyToRemove,
		}).Error("'timeOutMap' doesn't contain the object to be removed")

		//errorId := "c7e4917d-57d1-4a07-9c8e-0a793aa6174b"
		//err = errors.New(fmt.Sprintf("'timeOutMap' doesn't contain the object, '%s' to be removed [ErroId: %s]", timeOutMapKeyToRemove, errorId))

		return
	}

	// If the map only consist of 1 object then just remove it
	if len(timeOutMap) == 1 {

		delete(timeOutMap, timeOutMapKeyToRemove)

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

		// Delete current object
		delete(timeOutMap, timeOutMapKeyToRemove)

		// Update next object regarding previous object MapKey
		_ = testInstructionExecutionTimeOutEngineObject.updateTimeOutChannelCommandObject(
			nextTimeOutMapKey,
			nextTimeOutMapKey,
			"")

		return
	}

	// Check if the object, to be deleted, is between two objects
	if previousTimeOutMapKey != timeOutMapKeyToRemove &&
		timeOutMapKeyToRemove != nextTimeOutMapKey {

		// Delete current object
		delete(timeOutMap, timeOutMapKeyToRemove)

		// Update previous object regarding its next object
		_ = testInstructionExecutionTimeOutEngineObject.updateTimeOutChannelCommandObject(
			"",
			previousTimeOutMapKey,
			nextTimeOutMapKey)

		// Update next object regarding its previous object
		_ = testInstructionExecutionTimeOutEngineObject.updateTimeOutChannelCommandObject(
			previousTimeOutMapKey,
			nextTimeOutMapKey,
			"")

		return
	}

	// Check if the object, to be deleted, is last objects
	if timeOutMapKeyToRemove == nextTimeOutMapKey {

		// Delete current object
		delete(timeOutMap, timeOutMapKeyToRemove)

		// Update previous object regarding its next object
		_ = testInstructionExecutionTimeOutEngineObject.updateTimeOutChannelCommandObject(
			"",
			previousTimeOutMapKey,
			previousTimeOutMapKey)

		return
	}

}
