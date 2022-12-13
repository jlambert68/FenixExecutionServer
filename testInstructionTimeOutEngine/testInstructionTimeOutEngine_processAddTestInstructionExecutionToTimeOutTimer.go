package testInstructionTimeOutEngine

import (
	"FenixExecutionServer/common_config"
	"FenixExecutionServer/testInstructionExecutionEngine"
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

		timeOutMapKey = incomingTimeOutChannelCommand.TimeOutChannelTestInstructionExecutions.TestInstructionExecutionUuid +
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

		// Check if TimeOutTime has occurred
		if incomingTimeOutChannelCommand.TimeOutChannelTestInstructionExecutions.TimeOutTime.Before(time.Now().Add(extractTimerMarginalBeforeTimeOut)) == true {
			// TimeOutTime has already occurred, so act now
			var executionEngineChannelCommand testInstructionExecutionEngine.ChannelCommandStruct
			var executionEngineChannelCommandTestInstructionExecution testInstructionExecutionEngine.ChannelCommandTestInstructionExecutionStruct
			var executionEngineChannelCommandTestInstructionExecutions []testInstructionExecutionEngine.ChannelCommandTestInstructionExecutionStruct
			executionEngineChannelCommandTestInstructionExecution = testInstructionExecutionEngine.ChannelCommandTestInstructionExecutionStruct{
				TestCaseExecutionUuid:                   incomingTimeOutChannelCommand.TimeOutChannelTestInstructionExecutions.TestCaseExecutionUuid,
				TestCaseExecutionVersion:                incomingTimeOutChannelCommand.TimeOutChannelTestInstructionExecutions.TestCaseExecutionVersion,
				TestInstructionExecutionUuid:            incomingTimeOutChannelCommand.TimeOutChannelTestInstructionExecutions.TestInstructionExecutionUuid,
				TestInstructionExecutionVersion:         incomingTimeOutChannelCommand.TimeOutChannelTestInstructionExecutions.TestInstructionExecutionVersion,
				TestInstructionExecutionCanBeReExecuted: incomingTimeOutChannelCommand.TimeOutChannelTestInstructionExecutions.TestInstructionExecutionCanBeReExecuted,
			}
			executionEngineChannelCommandTestInstructionExecutions = []testInstructionExecutionEngine.ChannelCommandTestInstructionExecutionStruct{executionEngineChannelCommandTestInstructionExecution}

			executionEngineChannelCommand = testInstructionExecutionEngine.ChannelCommandStruct{
				ChannelCommand:                          testInstructionExecutionEngine.ChannelCommandProcessTestInstructionExecutionsThatHaveTimedOut,
				ChannelCommandTestCaseExecutions:        nil,
				ChannelCommandTestInstructionExecutions: executionEngineChannelCommandTestInstructionExecutions,
				ReturnChannelWithDBErrorReference:       nil,
			}

			// Send command to ExecutionsEngine
			testInstructionExecutionEngine.ExecutionEngineCommandChannel <- executionEngineChannelCommand
		}

		// Start new TimeOut-timer for this TestInstructionExecution as go-routine
		go func() {
			// Initiate the CancellableTimer
			var tempCancellableTimer *common_config.CancellableTimerStruct
			tempCancellableTimer = common_config.NewCancellableTimer()

			// Create Timer Response Channel
			var cancellableTimerReturnChannelReference common_config.CancellableTimerReturnChannelType
			var cancellableTimerReturnChannelResponseValue common_config.CancellableTimerEndStatusType

			// Initiate response channel for Timer
			cancellableTimerReturnChannelReference = make(chan common_config.CancellableTimerEndStatusType)

			// Start Timer
			var sleepDuration time.Duration
			sleepDuration = incomingTimeOutChannelCommand.TimeOutChannelTestInstructionExecutions.TimeOutTime.Sub(
				time.Now().Add(extractTimerMarginalBeforeTimeOut))
			common_config.StartCancellableTimer(tempCancellableTimer, sleepDuration, &cancellableTimerReturnChannelReference)

			// Wait for Timer to TimeOut or to be cancelled
			cancellableTimerReturnChannelResponseValue = <-cancellableTimerReturnChannelReference

			// Check if the Timer TimedOut or was cancelled
			// Only act when Timer did TimedOut
			if cancellableTimerReturnChannelResponseValue == common_config.CancellableTimerEndStatusTimedOut {

				// TimeOutTime has occurred
				var executionEngineChannelCommand testInstructionExecutionEngine.ChannelCommandStruct
				var executionEngineChannelCommandTestInstructionExecution testInstructionExecutionEngine.ChannelCommandTestInstructionExecutionStruct
				var executionEngineChannelCommandTestInstructionExecutions []testInstructionExecutionEngine.ChannelCommandTestInstructionExecutionStruct
				executionEngineChannelCommandTestInstructionExecution = testInstructionExecutionEngine.ChannelCommandTestInstructionExecutionStruct{
					TestCaseExecutionUuid:                   incomingTimeOutChannelCommand.TimeOutChannelTestInstructionExecutions.TestCaseExecutionUuid,
					TestCaseExecutionVersion:                incomingTimeOutChannelCommand.TimeOutChannelTestInstructionExecutions.TestCaseExecutionVersion,
					TestInstructionExecutionUuid:            incomingTimeOutChannelCommand.TimeOutChannelTestInstructionExecutions.TestInstructionExecutionUuid,
					TestInstructionExecutionVersion:         incomingTimeOutChannelCommand.TimeOutChannelTestInstructionExecutions.TestInstructionExecutionVersion,
					TestInstructionExecutionCanBeReExecuted: incomingTimeOutChannelCommand.TimeOutChannelTestInstructionExecutions.TestInstructionExecutionCanBeReExecuted,
				}
				executionEngineChannelCommandTestInstructionExecutions = []testInstructionExecutionEngine.ChannelCommandTestInstructionExecutionStruct{executionEngineChannelCommandTestInstructionExecution}

				executionEngineChannelCommand = testInstructionExecutionEngine.ChannelCommandStruct{
					ChannelCommand:                          testInstructionExecutionEngine.ChannelCommandProcessTestInstructionExecutionsThatHaveTimedOut,
					ChannelCommandTestCaseExecutions:        nil,
					ChannelCommandTestInstructionExecutions: executionEngineChannelCommandTestInstructionExecutions,
					ReturnChannelWithDBErrorReference:       nil,
				}

				// Send command to ExecutionsEngine that TestInstructionExecution TimedOut
				testInstructionExecutionEngine.ExecutionEngineCommandChannel <- executionEngineChannelCommand

				// Remove This TestInstructionExecution from Timer-queue (and set a new Timer for next TestInstructionExecution)
				var tempTimeOutChannelTestInstructionExecutions TimeOutChannelCommandTestInstructionExecutionStruct
				tempTimeOutChannelTestInstructionExecutions = TimeOutChannelCommandTestInstructionExecutionStruct{
					TestCaseExecutionUuid:                   incomingTimeOutChannelCommand.TimeOutChannelTestInstructionExecutions.TestCaseExecutionUuid,
					TestCaseExecutionVersion:                incomingTimeOutChannelCommand.TimeOutChannelTestInstructionExecutions.TestCaseExecutionVersion,
					TestInstructionExecutionUuid:            incomingTimeOutChannelCommand.TimeOutChannelTestInstructionExecutions.TestInstructionExecutionUuid,
					TestInstructionExecutionVersion:         incomingTimeOutChannelCommand.TimeOutChannelTestInstructionExecutions.TestInstructionExecutionVersion,
					TestInstructionExecutionCanBeReExecuted: incomingTimeOutChannelCommand.TimeOutChannelTestInstructionExecutions.TestInstructionExecutionCanBeReExecuted,
					TimeOutTime:                             incomingTimeOutChannelCommand.TimeOutChannelTestInstructionExecutions.TimeOutTime,
				}

				var tempTimeOutChannelCommand TimeOutChannelCommandStruct
				tempTimeOutChannelCommand = TimeOutChannelCommandStruct{
					TimeOutChannelCommand:                                               TimeOutChannelCommandRemoveTestInstructionExecutionFromTimeOutTimer,
					TimeOutChannelTestInstructionExecutions:                             tempTimeOutChannelTestInstructionExecutions,
					TimeOutReturnChannelForTimeOutHasOccurred:                           nil,
					TimeOutReturnChannelForExistsTestInstructionExecutionInTimeOutTimer: nil,
				}

				// Send message on TimeOutEngineChannel to remove TestInstructionExecution from Timer-queue
				TimeOutChannelEngineCommandChannel <- tempTimeOutChannelCommand
			}

			return
		}()

		return
	}

	// If there are at least one 'TimeOutChannelCommandObject' in 'timeOutMap' then process in a recursive way.
	// Start with object that has next upcoming TimeOut
	_ = testInstructionExecutionTimeOutEngineObject.recursiveAddTestInstructionExecutionToTimeOutTimer(incomingTimeOutChannelCommand, nextUpcomingObjectMapKeyWithTimeOut)

}

func (testInstructionExecutionTimeOutEngineObject *TestInstructionTimeOutEngineObjectStruct) recursiveAddTestInstructionExecutionToTimeOutTimer(
	newIncomingTimeOutChannelCommandObject *TimeOutChannelCommandStruct, currentTimeOutMapKey string) (err error) {

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

	// Extract current object timeout time
	currentProcessedObjectsTimeOutTime = currentProcessedObjects.currentTimeOutChannelCommandObject.TimeOutChannelTestInstructionExecutions.TimeOutTime

	// Create Map-key for 'timeOutMap' for new object
	var newObjectsTimeOutMapKey string

	var testInstructionExecutionVersionAsString string
	testInstructionExecutionVersionAsString = strconv.Itoa(
		int(newIncomingTimeOutChannelCommandObject.TimeOutChannelTestInstructionExecutions.TestInstructionExecutionVersion))

	newObjectsTimeOutMapKey = newIncomingTimeOutChannelCommandObject.TimeOutChannelTestInstructionExecutions.TestInstructionExecutionUuid +
		testInstructionExecutionVersionAsString

	// Check if new object should be inserted before current object
	if newObjectsTimeOutTime.Before(currentProcessedObjectsTimeOutTime) == true {

		// Current object is first object
		if currentProcessedObjects.previousTimeOutMapKey == currentProcessedObjects.currentTimeOutMapKey {

			// Insert new object before current object
			err = testInstructionExecutionTimeOutEngineObject.storeNewTimeOutChannelCommandObject(
				newObjectsTimeOutMapKey,
				newObjectsTimeOutMapKey,
				currentTimeOutMapKey,
				newIncomingTimeOutChannelCommandObject)

			if err != nil {
				return err
			}

			// Update current object regarding previous and next object MapKey
			err = testInstructionExecutionTimeOutEngineObject.updateTimeOutChannelCommandObject(
				newObjectsTimeOutMapKey,
				currentTimeOutMapKey,
				"")

			return err
		}

		// Current object is not the first object
		if currentProcessedObjects.previousTimeOutMapKey != currentProcessedObjects.currentTimeOutMapKey &&
			currentProcessedObjects.currentTimeOutMapKey != currentProcessedObjects.nextTimeOutMapKey {

			// Insert new object before current object
			err = testInstructionExecutionTimeOutEngineObject.storeNewTimeOutChannelCommandObject(
				currentProcessedObjects.previousTimeOutMapKey,
				newObjectsTimeOutMapKey,
				currentTimeOutMapKey,
				newIncomingTimeOutChannelCommandObject)

			if err != nil {
				return err
			}

			// Update current object regarding previous and next object MapKey
			err = testInstructionExecutionTimeOutEngineObject.updateTimeOutChannelCommandObject(
				newObjectsTimeOutMapKey,
				currentTimeOutMapKey,
				"")

			return err
		}
	}

	// Check if current object is the last object
	if currentProcessedObjects.currentTimeOutMapKey == currentProcessedObjects.nextTimeOutMapKey {

		// Insert new object after current object
		err = testInstructionExecutionTimeOutEngineObject.storeNewTimeOutChannelCommandObject(
			currentProcessedObjects.currentTimeOutMapKey,
			newObjectsTimeOutMapKey,
			newObjectsTimeOutMapKey,
			newIncomingTimeOutChannelCommandObject)

		if err != nil {
			return err
		}

		// Update current object regarding previous and next object MapKey
		err = testInstructionExecutionTimeOutEngineObject.updateTimeOutChannelCommandObject(
			"",
			currentTimeOutMapKey,
			newObjectsTimeOutMapKey)

		return err
	}

	// Do recursive call to "next object"
	err = testInstructionExecutionTimeOutEngineObject.recursiveAddTestInstructionExecutionToTimeOutTimer(
		newIncomingTimeOutChannelCommandObject,
		currentProcessedObjects.nextTimeOutMapKey)

	return err
}

// Insert New object into map
func (testInstructionExecutionTimeOutEngineObject *TestInstructionTimeOutEngineObjectStruct) storeNewTimeOutChannelCommandObject(
	previousTimeOutMapKey string,
	currentTimeOutMapKey string,
	nextTimeOutMapKey string,
	newIncomingTimeOutChannelCommandObject *TimeOutChannelCommandStruct) (err error) {

	var existsInMap bool

	// Create Map-key for 'timeOutMap'
	var timeOutMapKey string

	var testInstructionExecutionVersionAsString string
	testInstructionExecutionVersionAsString = strconv.Itoa(int(newIncomingTimeOutChannelCommandObject.TimeOutChannelTestInstructionExecutions.TestInstructionExecutionVersion))

	timeOutMapKey = strconv.Itoa(int(
		newIncomingTimeOutChannelCommandObject.TimeOutChannelTestInstructionExecutions.TestInstructionExecutionVersion)) +
		testInstructionExecutionVersionAsString

	// Create object to be stored
	var timeOutMapObject *timeOutMapStruct
	timeOutMapObject = &timeOutMapStruct{
		currentTimeOutMapKey:               currentTimeOutMapKey,
		previousTimeOutMapKey:              previousTimeOutMapKey,
		nextTimeOutMapKey:                  nextTimeOutMapKey,
		currentTimeOutChannelCommandObject: newIncomingTimeOutChannelCommandObject,
	}

	// Verify that Object doesn't exist in Map
	_, existsInMap = timeOutMap[currentTimeOutMapKey]
	if existsInMap == true {
		common_config.Logger.WithFields(logrus.Fields{
			"id":                   "ac69e72d-b124-43c8-a7e9-8760da369bec",
			"currentTimeOutMapKey": currentTimeOutMapKey,
		}).Error("'timeOutMap' does already contain an object for for the 'currentTimeOutMapKey'")

		errorId := "af257654-4034-451d-9d1e-8385e2497264"
		err = errors.New(fmt.Sprintf("'timeOutMap' does already contain an object for for the 'currentTimeOutMapKey': '%s' [ErroId: %s]", currentTimeOutMapKey, errorId))

		return err

	}

	// Store object in 'timeOutMap'
	timeOutMap[timeOutMapKey] = timeOutMapObject

	return err
}

// Update Previous and Next MapKey for Object
func (testInstructionExecutionTimeOutEngineObject *TestInstructionTimeOutEngineObjectStruct) updateTimeOutChannelCommandObject(
	previousTimeOutMapKey string,
	currentTimeOutMapKey string,
	nextTimeOutMapKey string) (err error) {

	// Create object to be stored
	var timeOutMapObject *timeOutMapStruct
	var existsInMap bool

	// Verify that Object does exist in Map
	timeOutMapObject, existsInMap = timeOutMap[currentTimeOutMapKey]
	if existsInMap == false {
		common_config.Logger.WithFields(logrus.Fields{
			"id":                   "015c9dca-211d-47d9-bd4e-0a30ed144032",
			"currentTimeOutMapKey": currentTimeOutMapKey,
		}).Error("'timeOutMap' doesn't have any object for 'currentTimeOutMapKey'")

		errorId := "cc40f9db-fd3f-4963-af85-c6d3a63da58a"
		err = errors.New(fmt.Sprintf("'timeOutMap'doesn't have any object for 'currentTimeOutMapKey': '%s' [ErroId: %s]", currentTimeOutMapKey, errorId))

		return err

	}

	// Update object in 'timeOutMap'
	if previousTimeOutMapKey != "" {
		timeOutMapObject.previousTimeOutMapKey = previousTimeOutMapKey
	}
	if previousTimeOutMapKey != "" {
		timeOutMapObject.nextTimeOutMapKey = nextTimeOutMapKey
	}

	return err
}
