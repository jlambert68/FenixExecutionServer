package testInstructionTimeOutEngine

import (
	"FenixExecutionServer/common_config"
	"errors"
	"fmt"
	"github.com/sirupsen/logrus"
	"strconv"
	"time"
)

// Add TestInstructionExecution to TimeOut-timer
func (testInstructionExecutionTimeOutEngineObject *TestInstructionTimeOutEngineObjectStruct) processAddTestInstructionExecutionToTimeOutTimer(incomingTimeOutChannelCommand *TimeOutChannelCommandStruct) {

	common_config.Logger.WithFields(logrus.Fields{
		"id":                            "bb9dcac7-ef0d-4559-b710-9ac04b3b4c6a",
		"incomingTimeOutChannelCommand": incomingTimeOutChannelCommand,
	}).Debug("Incoming 'processAddTestInstructionExecutionToTimeOutTimer'")

	defer common_config.Logger.WithFields(logrus.Fields{
		"id": "3d82a2e0-a8ba-4d01-95ec-aab154b227d3",
	}).Debug("Outgoing 'processAddTestInstructionExecutionToTimeOutTimer'")

	// If the map is empty then this TestInstructionExecution has the next, and only, upcoming TimeOut
	if len(timeOutMap) == 0 {

		// Create Map-key for 'timeOutMap'
		var timeOutMapKey string

		var testInstructionExecutionVersionAsString string
		testInstructionExecutionVersionAsString = strconv.Itoa(int(incomingTimeOutChannelCommand.TimeOutChannelTestInstructionExecutions.TestInstructionExecutionVersion))

		timeOutMapKey = strconv.Itoa(int(
			incomingTimeOutChannelCommand.TimeOutChannelTestInstructionExecutions.TestInstructionExecutionVersion)) +
			testInstructionExecutionVersionAsString

		// Create object to be stored
		var timeOutMapObject *timeOutMapStruct
		timeOutMapObject = &timeOutMapStruct{
			currentTimeOutMapKey:               timeOutMapKey,
			previousTimeOutMapKey:              timeOutMapKey,
			nextTimeOutMapKey:                  timeOutMapKey,
			currentTimeOutChannelCommandObject: incomingTimeOutChannelCommand,
		}

		// Store object in 'timeOutMap'
		timeOutMap[timeOutMapKey] = timeOutMapObject

		// Set variable that keeps track of next object with upcoming Timeout
		nextUpcomingObjectMapKeyWithTimeOut = timeOutMapKey

		//TODO Call TIMER

		return
	}

	// If there are at least one 'TimeOutChannelCommandObject' in 'timeOutMap' then process in a recursive way.
	// Start with object that has next upcoming TimeOut
	_ = testInstructionExecutionTimeOutEngineObject.recursiveAddTestInstructionExecutionToTimeOutTimer(incomingTimeOutChannelCommand, nextUpcomingObjectMapKeyWithTimeOut)

}

func (testInstructionExecutionTimeOutEngineObject *TestInstructionTimeOutEngineObjectStruct) recursiveAddTestInstructionExecutionToTimeOutTimer(newIncomingTimeOutChannelCommandObject *TimeOutChannelCommandStruct, currentTimeOutMapKey string) (err error) {

	var existsInMap bool

	// Extract TimeOut-time from Objects
	var newObjectsTimeOutTime time.Time
	var currentProcessedObjectsTimeOutTime time.Time

	// Extract TimeOut-time from new object
	newObjectsTimeOutTime = newIncomingTimeOutChannelCommandObject.TimeOutChannelTestInstructionExecutions.TimeOutTime

	// Extract TimeOut-time from current processed object
	var currentProcessedObjects *timeOutMapStruct
	currentProcessedObjects, existsInMap = timeOutMap[currentTimeOutMapKey]
	if existsInMap == false {
		common_config.Logger.WithFields(logrus.Fields{
			"id":                   "9d6f51c0-9a68-477c-844c-61ab5f6416bb",
			"currentTimeOutMapKey": currentTimeOutMapKey,
		}).Error("'timeOutMap' doesn't contain any object for for the 'currentTimeOutMapKey'")

		errorId := "af257654-4034-451d-9d1e-8385e2497264"
		err = errors.New(fmt.Sprintf("'timeOutMap' doesn't contain any object for for the 'currentTimeOutMapKey': '%s' [ErroId: %s]", currentTimeOutMapKey, errorId))

		return err

	}

	currentProcessedObjectsTimeOutTime = currentProcessedObjects.currentTimeOutChannelCommandObject.TimeOutChannelTestInstructionExecutions.TimeOutTime

	// Check if new object should be inserted before current object
	if newObjectsTimeOutTime.Before(currentProcessedObjectsTimeOutTime) == true {

		// Rearrange chain of objects
		var previousObject *timeOutMapStruct
		var nextObject *timeOutMapStruct
		var objectPositionFound bool

		// Current object is both first and last object
		if len(timeOutMapStruct) == 1 {
			objectPositionFound = true

			return err
		}

		// Current object is first object
		if currentProcessedObjects.previousTimeOutMapKey = currentProcessedObjects.currentTimeOutMapKey {

			// Insert new object before current object

			// Do recursive call to next object
			err = testInstructionExecutionTimeOutEngineObject.recursiveAddTestInstructionExecutionToTimeOutTimer(newIncomingTimeOutChannelCommandObject, currentProcessedObjects.nextTimeOutMapKey)

			return err
		}

		// Current object is not the first or last object

		// Insert new object before current object

		// Do recursive call to next object
		err = testInstructionExecutionTimeOutEngineObject.recursiveAddTestInstructionExecutionToTimeOutTimer(newIncomingTimeOutChannelCommandObject, currentProcessedObjects.nextTimeOutMapKey)

		return err

		// Current object is last object

	}

}
