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
func (testInstructionExecutionTimeOutEngineObject *TestInstructionTimeOutEngineObjectStruct) processAddTestInstructionExecutionToTimeOutTimer(
	incomingTimeOutChannelCommand *common_config.TimeOutChannelCommandStruct) {

	common_config.Logger.WithFields(logrus.Fields{
		"id":                            "bb9dcac7-ef0d-4559-b710-9ac04b3b4c6a",
		"incomingTimeOutChannelCommand": incomingTimeOutChannelCommand,
		"len(timeOutMap)":               len(timeOutMap),
		"len(timedOutMap),":             len(timedOutMap),
	}).Debug("Incoming 'processAddTestInstructionExecutionToTimeOutTimer'")

	defer common_config.Logger.WithFields(logrus.Fields{
		"id": "3d82a2e0-a8ba-4d01-95ec-aab154b227d3",
	}).Debug("Outgoing 'processAddTestInstructionExecutionToTimeOutTimer'")

	// Force GC to clear up, should see a memory drop
	//runtime.GC()
	common_config.PrintMemUsage()

	// Check if TimeOutTime has occurred
	/*
		if incomingTimeOutChannelCommand.TimeOutChannelTestInstructionExecutions.TimeOutTime.After(
			time.Now().Add(extractTimerMarginalBeforeTimeOut)) == true {

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

			return
		}
	*/

	// Variables needed
	var existsInMap bool

	// Create Map-key for 'timeOutMap'
	var timeOutMapKey string

	var testInstructionExecutionVersionAsString string
	testInstructionExecutionVersionAsString = strconv.Itoa(int(
		incomingTimeOutChannelCommand.TimeOutChannelTestInstructionExecutions.TestInstructionExecutionVersion))

	timeOutMapKey = incomingTimeOutChannelCommand.TimeOutChannelTestInstructionExecutions.TestInstructionExecutionUuid +
		testInstructionExecutionVersionAsString

	// There must be an allocation before a TimeOut-timer can be added
	// The reason is that sometime a very fast response, from sent TestInstructionExecutions, then the TimerRemove-command comes in before the TimerAdd-command
	_, existsInMap = allocatedTimeOutTimerMap[timeOutMapKey]
	if existsInMap == false {

		common_config.Logger.WithFields(logrus.Fields{
			"id":                            "da1ea775-627d-4856-961c-1da9f20c9751",
			"incomingTimeOutChannelCommand": incomingTimeOutChannelCommand,
			"timeOutMapKey":                 timeOutMapKey,
		}).Debug("There exist no allocated TimeOut-timer so we should NOT create a new TimeOut-timer")

		return
	}

	// Secure that the TimeOut-timer doesn't already exist
	_, existsInMap = timeOutMap[timeOutMapKey]
	if existsInMap == true {
		common_config.Logger.WithFields(logrus.Fields{
			"id":            "efeba77a-d36d-4c32-a27f-8b7bbe2e3855",
			"timeOutMapKey": timeOutMapKey,
		}).Error("'timeOutMap' does already contain an object for for the 'timeOutMapKey'")

		return
	}

	// If the map is empty then this TestInstructionExecution has the next, and only, upcoming TimeOut
	if len(timeOutMap) == 0 {

		// Create object to be stored
		var timeOutMapObject *timeOutMapStruct
		timeOutMapObject = &timeOutMapStruct{
			currentTimeOutMapKey:               timeOutMapKey,
			previousTimeOutMapKey:              timeOutMapKey,
			nextTimeOutMapKey:                  timeOutMapKey,
			currentTimeOutChannelCommandObject: incomingTimeOutChannelCommand,
		}

		if timeOutMapKey == "11" {
			common_config.Logger.WithFields(logrus.Fields{
				"id":                            "da1ea775-627d-4856-961c-1da9f20c9751",
				"incomingTimeOutChannelCommand": incomingTimeOutChannelCommand,
				"timeOutMapKey":                 timeOutMapKey,
			}).Debug("THIS IS WRONG")
		}

		// Store object in 'timeOutMap'
		timeOutMap[timeOutMapKey] = timeOutMapObject

		// Set variable that keeps track of next object with upcoming Timeout
		nextUpcomingObjectMapKeyWithTimeOut = timeOutMapKey

		// Remove Allocation for TimeOut-timer
		delete(allocatedTimeOutTimerMap, timeOutMapKey)

		// Start new TimeOut-timer for this TestInstructionExecution as go-routine
		go testInstructionExecutionTimeOutEngineObject.startTimeOutTimerTestInstructionExecution(
			timeOutMapObject,
			incomingTimeOutChannelCommand,
			timeOutMapKey,
			"cb66ee1d-9500-41fe-90d3-14aa30d233d5")

		return
	}

	// If there are at least one 'TimeOutChannelCommandObject' in 'timeOutMap' then process in a recursive way.
	// Start with object that has next upcoming TimeOut
	_ = testInstructionExecutionTimeOutEngineObject.recursiveAddTestInstructionExecutionToTimeOutTimer(
		incomingTimeOutChannelCommand,
		nextUpcomingObjectMapKeyWithTimeOut)

}

func (testInstructionExecutionTimeOutEngineObject *TestInstructionTimeOutEngineObjectStruct) recursiveAddTestInstructionExecutionToTimeOutTimer(
	newIncomingTimeOutChannelCommandObject *common_config.TimeOutChannelCommandStruct,
	currentTimeOutMapKey string) (err error) {

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
		err = errors.New(fmt.Sprintf(
			"'timeOutMap' doesn't contain any object for for the 'currentTimeOutMapKey': '%s' [ErroId: %s]",
			currentTimeOutMapKey,
			errorId))

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
			var newTimeOutMapObject *timeOutMapStruct
			newTimeOutMapObject, err = testInstructionExecutionTimeOutEngineObject.storeNewTimeOutChannelCommandObject(
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

			// Set variable that keeps track of first object with upcoming Timeout
			nextUpcomingObjectMapKeyWithTimeOut = newObjectsTimeOutMapKey

			// Cancel timer of previous first object
			currentProcessedObjects.cancellableTimer.Cancel()

			// Start new TimeOut-timer for this TestInstructionExecution as go-routine
			go testInstructionExecutionTimeOutEngineObject.startTimeOutTimerTestInstructionExecution(
				newTimeOutMapObject,
				newIncomingTimeOutChannelCommandObject,
				newObjectsTimeOutMapKey,
				"41a72efa-bdd3-4933-a913-a96e69a7c9c6")

			return err

		}

		// Current object is not the first object or the last object
		if currentProcessedObjects.previousTimeOutMapKey != currentProcessedObjects.currentTimeOutMapKey &&
			currentProcessedObjects.currentTimeOutMapKey != currentProcessedObjects.nextTimeOutMapKey {

			// Insert new object before current object
			_, err = testInstructionExecutionTimeOutEngineObject.storeNewTimeOutChannelCommandObject(
				currentProcessedObjects.previousTimeOutMapKey,
				newObjectsTimeOutMapKey,
				currentTimeOutMapKey,
				newIncomingTimeOutChannelCommandObject)

			if err != nil {
				return err
			}

			// Update previous object regarding next object MapKey
			err = testInstructionExecutionTimeOutEngineObject.updateTimeOutChannelCommandObject(
				"",
				currentProcessedObjects.previousTimeOutMapKey,
				newObjectsTimeOutMapKey)

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
	} else {

		// Verify if current object is the last object
		if currentProcessedObjects.currentTimeOutMapKey == currentProcessedObjects.nextTimeOutMapKey {

			// Insert new object after current object
			_, err = testInstructionExecutionTimeOutEngineObject.storeNewTimeOutChannelCommandObject(
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

	return err
}

// Insert New object into map
func (testInstructionExecutionTimeOutEngineObject *TestInstructionTimeOutEngineObjectStruct) storeNewTimeOutChannelCommandObject(
	previousTimeOutMapKey string,
	currentTimeOutMapKey string,
	nextTimeOutMapKey string,
	newIncomingTimeOutChannelCommandObject *common_config.TimeOutChannelCommandStruct) (timeOutMapObject *timeOutMapStruct, err error) {

	var existsInMap bool

	// Create Map-key for 'timeOutMap'
	var timeOutMapKey string

	var testInstructionExecutionVersionAsString string
	testInstructionExecutionVersionAsString = strconv.Itoa(
		int(newIncomingTimeOutChannelCommandObject.TimeOutChannelTestInstructionExecutions.TestInstructionExecutionVersion))

	timeOutMapKey = newIncomingTimeOutChannelCommandObject.TimeOutChannelTestInstructionExecutions.TestInstructionExecutionUuid +
		testInstructionExecutionVersionAsString

	// Create object to be stored
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

		return nil, err

	}

	if timeOutMapKey == "11" {
		common_config.Logger.WithFields(logrus.Fields{
			"id":                            "795aad00-3b5a-40fe-8ace-49dd511dc24e",
			"incomingTimeOutChannelCommand": newIncomingTimeOutChannelCommandObject,
			"timeOutMapKey":                 timeOutMapKey,
		}).Debug("THIS IS WRONG")
	}

	// Store object in 'timeOutMap'
	timeOutMap[timeOutMapKey] = timeOutMapObject

	// Is this the first object, to Time Out
	if currentTimeOutMapKey == previousTimeOutMapKey {

		// Set variable that keeps track of next object with upcoming Timeout
		nextUpcomingObjectMapKeyWithTimeOut = timeOutMapKey
	}

	// Remove Allocation for TimeOut-timer
	delete(allocatedTimeOutTimerMap, timeOutMapKey)

	return timeOutMapObject, err
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
	if nextTimeOutMapKey != "" {
		timeOutMapObject.nextTimeOutMapKey = nextTimeOutMapKey
	}

	return err
}

// Start new TimeOut-timer for this TestInstructionExecution as go-routine
func (testInstructionExecutionTimeOutEngineObject *TestInstructionTimeOutEngineObjectStruct) startTimeOutTimerTestInstructionExecution(
	timeOutMapObjectReference *timeOutMapStruct,
	incomingTimeOutChannelCommand *common_config.TimeOutChannelCommandStruct,
	timeOutMapKey string,
	senderId string) {

	common_config.Logger.WithFields(logrus.Fields{
		"id":                            "2834d828-f3b7-4f7d-a33e-f12c86e89432",
		"incomingTimeOutChannelCommand": incomingTimeOutChannelCommand,
		"timeOutMapKey":                 timeOutMapKey,
		"senderId":                      senderId,
	}).Debug("Incoming 'startTimeOutTimerTestInstructionExecution'")

	defer common_config.Logger.WithFields(logrus.Fields{
		"id": "fd15766c-fce1-4fb6-82f4-e4825d14935b",
	}).Debug("Outgoing 'startTimeOutTimerTestInstructionExecution'")

	// Initiate the CancellableTimer
	var tempCancellableTimer *common_config.CancellableTimerStruct
	tempCancellableTimer = common_config.NewCancellableTimer()

	// Create Timer Response Channel
	var cancellableTimerReturnChannel common_config.CancellableTimerReturnChannelType
	var cancellableTimerReturnChannelResponseValue common_config.CancellableTimerEndStatusType

	// Initiate response channel for Timer
	cancellableTimerReturnChannel = make(chan common_config.CancellableTimerEndStatusType)

	// Add reference to CancellableTimer in 'timeOutMap-object'
	timeOutMapObjectReference.cancellableTimer = tempCancellableTimer

	// Calculate Timer-time
	var sleepDuration time.Duration
	sleepDuration = incomingTimeOutChannelCommand.TimeOutChannelTestInstructionExecutions.TimeOutTime.Add(extractTimerMarginalBeforeTimeOut).Sub(time.Now())

	// Initiate a Timer
	go common_config.StartCancellableTimer(
		tempCancellableTimer,
		sleepDuration,
		&cancellableTimerReturnChannel,
		timeOutMapKey)

	// Wait for Timer to TimeOut or to be cancelled
	cancellableTimerReturnChannelResponseValue = <-cancellableTimerReturnChannel

	common_config.Logger.WithFields(logrus.Fields{
		"id": "e25b7b8e-3ce5-4428-8f90-d13fc81182b3",
		"cancellableTimerReturnChannelResponseValue": cancellableTimerReturnChannelResponseValue,
		"timeOutMapKey": timeOutMapKey,
	}).Debug("Response from 'cancellableTimerReturnChannel'")

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
		var tempTimeOutChannelTestInstructionExecutions common_config.TimeOutChannelCommandTestInstructionExecutionStruct
		tempTimeOutChannelTestInstructionExecutions = common_config.TimeOutChannelCommandTestInstructionExecutionStruct{
			TestCaseExecutionUuid:                   incomingTimeOutChannelCommand.TimeOutChannelTestInstructionExecutions.TestCaseExecutionUuid,
			TestCaseExecutionVersion:                incomingTimeOutChannelCommand.TimeOutChannelTestInstructionExecutions.TestCaseExecutionVersion,
			TestInstructionExecutionUuid:            incomingTimeOutChannelCommand.TimeOutChannelTestInstructionExecutions.TestInstructionExecutionUuid,
			TestInstructionExecutionVersion:         incomingTimeOutChannelCommand.TimeOutChannelTestInstructionExecutions.TestInstructionExecutionVersion,
			TestInstructionExecutionCanBeReExecuted: incomingTimeOutChannelCommand.TimeOutChannelTestInstructionExecutions.TestInstructionExecutionCanBeReExecuted,
			TimeOutTime:                             incomingTimeOutChannelCommand.TimeOutChannelTestInstructionExecutions.TimeOutTime,
		}

		var tempTimeOutChannelCommand common_config.TimeOutChannelCommandStruct
		tempTimeOutChannelCommand = common_config.TimeOutChannelCommandStruct{
			TimeOutChannelCommand:                   common_config.TimeOutChannelCommandRemoveTestInstructionExecutionFromTimeOutTimerDueToTimeOutFromTimer,
			TimeOutChannelTestInstructionExecutions: tempTimeOutChannelTestInstructionExecutions,
			SendID:                                  "6605911d-7e53-4321-b359-9e55b399dd44",
		}

		// Send message on TimeOutEngineChannel to remove TestInstructionExecution from Timer-queue
		TimeOutChannelEngineCommandChannel <- tempTimeOutChannelCommand
	}

	return

}
