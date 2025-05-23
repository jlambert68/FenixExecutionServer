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
	executionTrack int,
	incomingTimeOutChannelCommand *common_config.TimeOutChannelCommandStruct) {

	common_config.Logger.WithFields(logrus.Fields{
		"id":                                      "bb9dcac7-ef0d-4559-b710-9ac04b3b4c6a",
		"incomingTimeOutChannelCommand":           incomingTimeOutChannelCommand,
		"TimeOutChannelCommand":                   common_config.TimeOutChannelCommandsForDebugPrinting[incomingTimeOutChannelCommand.TimeOutChannelCommand],
		"executionTrack":                          executionTrack,
		"len(*timeOutMapSlice[executionTrack])":   len(*timeOutMapSlice[executionTrack]),
		"len(*timedOutMapSlice[executionTrack]),": len(*timedOutMapSlice[executionTrack]),
		"SendID": incomingTimeOutChannelCommand.SendID,
	}).Debug("Incoming 'processAddTestInstructionExecutionToTimeOutTimer'")

	defer common_config.Logger.WithFields(logrus.Fields{
		"id": "6d94b817-4efe-40e1-bd8a-95a201dae3de",
	}).Debug("Outgoing 'processAddTestInstructionExecutionToTimeOutTimer'")

	incomingTimeOutMapCount := len(*timeOutMapSlice[executionTrack])

	common_config.Logger.WithFields(logrus.Fields{
		"id":                      "58f707fe-36f6-4c05-8d75-7b1e00377e8f",
		"key":                     incomingTimeOutChannelCommand.TimeOutChannelTestInstructionExecutions.TestInstructionExecutionUuid,
		"incomingTimeOutMapCount": incomingTimeOutMapCount,
	}).Debug("Number of incoming items '*timeOutMapSlice[executionTrack]'")

	var noOfTimeOutTimerWasFound bool

	// Secure that number of TimeOut-timers has increased by one
	defer func() {
		outgoingTimeOutMapCount := len(*timeOutMapSlice[executionTrack])

		if noOfTimeOutTimerWasFound == false && outgoingTimeOutMapCount-incomingTimeOutMapCount != 1 {
			common_config.Logger.WithFields(logrus.Fields{
				"id":       "07a9f4a2-92d1-4dbf-b366-bff9092def5b",
				"key":      incomingTimeOutChannelCommand.TimeOutChannelTestInstructionExecutions.TestInstructionExecutionUuid,
				"SenderId": incomingTimeOutChannelCommand.SendID,
			}).Error("Missing one item in '*timeOutMapSlice[executionTrack]'")

			//			testInstructionExecutionTimeOutEngineObject.processAddTestInstructionExecutionToTimeOutTimer(
			//				executionTrack,
			//				incomingTimeOutChannelCommand)
		}
	}()

	// Force GC to clear up, should see a memory drop
	//runtime.GC()
	common_config.PrintMemUsage()

	// Variables needed
	var existsInMap bool

	// Create Map-key for '*timeOutMapSlice[executionTrack]'
	var timeOutMapKey string

	var testInstructionExecutionVersionAsString string
	testInstructionExecutionVersionAsString = strconv.Itoa(int(
		incomingTimeOutChannelCommand.TimeOutChannelTestInstructionExecutions.TestInstructionExecutionVersion))

	timeOutMapKey = incomingTimeOutChannelCommand.TimeOutChannelTestInstructionExecutions.TestInstructionExecutionUuid +
		testInstructionExecutionVersionAsString

	// Secure that the TimeOut-timer doesn't already exist
	_, existsInMap = (*timeOutMapSlice[executionTrack])[timeOutMapKey]
	if existsInMap == true {
		common_config.Logger.WithFields(logrus.Fields{
			"id":            "47e01275-2517-4c7a-8e09-71a15eb133c7",
			"timeOutMapKey": timeOutMapKey,
		}).Error("'*timeOutMapSlice[executionTrack]' already exist")

		return

	}

	// If the map is empty then this TestInstructionExecution has the next, and only, upcoming TimeOut
	if len(*timeOutMapSlice[executionTrack]) == 0 {

		// Create object to be stored
		var timeOutMapObject *timeOutMapStruct
		timeOutMapObject = &timeOutMapStruct{
			currentTimeOutMapKey:               timeOutMapKey,
			previousTimeOutMapKey:              timeOutMapKey,
			nextTimeOutMapKey:                  timeOutMapKey,
			currentTimeOutChannelCommandObject: incomingTimeOutChannelCommand,
		}

		// Store object in '*timeOutMapSlice[executionTrack]'
		(*timeOutMapSlice[executionTrack])[timeOutMapKey] = timeOutMapObject

		// Set variable that keeps track of next object with upcoming Timeout
		nextUpcomingObjectMapKeyWithTimeOutSlice[executionTrack] = timeOutMapKey

		// Start new TimeOut-timer for this TestInstructionExecution as go-routine
		go testInstructionExecutionTimeOutEngineObject.startTimeOutTimerTestInstructionExecution(
			executionTrack,
			timeOutMapObject,
			incomingTimeOutChannelCommand,
			timeOutMapKey,
			"cb66ee1d-9500-41fe-90d3-14aa30d233d5")

		return
	}

	// If there are at least one 'TimeOutChannelCommandObject' in '*timeOutMapSlice[executionTrack]' then process in a recursive way.
	// Start with object that has next upcoming TimeOut
	_ = testInstructionExecutionTimeOutEngineObject.recursiveAddTestInstructionExecutionToTimeOutTimer(
		executionTrack,
		incomingTimeOutChannelCommand,
		nextUpcomingObjectMapKeyWithTimeOutSlice[executionTrack],
		incomingTimeOutChannelCommand.MessageInitiatedFromPubSubSend)

}

func (testInstructionExecutionTimeOutEngineObject *TestInstructionTimeOutEngineObjectStruct) recursiveAddTestInstructionExecutionToTimeOutTimer(
	executionTrack int,
	newIncomingTimeOutChannelCommandObject *common_config.TimeOutChannelCommandStruct,
	currentTimeOutMapKey string,
	messageInitiatedFromPubSubSend bool) (err error) {

	var existsInMap bool

	// Extract TimeOut-time from Objects
	var newObjectsTimeOutTime time.Time
	var currentProcessedObjectsTimeOutTime time.Time

	// Extract TimeOut-time from new object
	newObjectsTimeOutTime = newIncomingTimeOutChannelCommandObject.TimeOutChannelTestInstructionExecutions.TimeOutTime

	// Extract TimeOut-time from current processed object
	var currentProcessedObjects *timeOutMapStruct
	currentProcessedObjects, existsInMap = (*timeOutMapSlice[executionTrack])[currentTimeOutMapKey]
	if existsInMap == false {
		common_config.Logger.WithFields(logrus.Fields{
			"id":                   "9d6f51c0-9a68-477c-844c-61ab5f6416bb",
			"currentTimeOutMapKey": currentTimeOutMapKey,
		}).Error("'*timeOutMapSlice[executionTrack]' doesn't contain any object for for the 'currentTimeOutMapKey'")

		errorId := "af257654-4034-451d-9d1e-8385e2497264"
		err = errors.New(fmt.Sprintf(
			"'*timeOutMapSlice[executionTrack]' doesn't contain any object for for the 'currentTimeOutMapKey': '%s' [ErroId: %s]",
			currentTimeOutMapKey,
			errorId))

		return err

	}

	// Extract current object timeout time
	currentProcessedObjectsTimeOutTime = currentProcessedObjects.currentTimeOutChannelCommandObject.TimeOutChannelTestInstructionExecutions.TimeOutTime

	// Create Map-key for '*timeOutMapSlice[executionTrack]' for new object
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
				executionTrack,
				newObjectsTimeOutMapKey,
				newObjectsTimeOutMapKey,
				currentTimeOutMapKey,
				newIncomingTimeOutChannelCommandObject,
				"90b48a0a-8281-4d41-b15b-bc155456948c",
				messageInitiatedFromPubSubSend)

			if err != nil {
				return err
			}

			// Update current object regarding previous and next object MapKey
			err = testInstructionExecutionTimeOutEngineObject.updateTimeOutChannelCommandObject(
				executionTrack,
				newObjectsTimeOutMapKey,
				currentTimeOutMapKey,
				"")

			// Set variable that keeps track of first object with upcoming Timeout
			nextUpcomingObjectMapKeyWithTimeOutSlice[executionTrack] = newObjectsTimeOutMapKey

			// Cancel timer of previous first object
			currentProcessedObjects.cancellableTimer.Cancel()

			// Start new TimeOut-timer for this TestInstructionExecution as go-routine
			go testInstructionExecutionTimeOutEngineObject.startTimeOutTimerTestInstructionExecution(
				executionTrack,
				newTimeOutMapObject,
				newIncomingTimeOutChannelCommandObject,
				newObjectsTimeOutMapKey,
				"41a72efa-bdd3-4933-a913-a96e69a7c9c6")

			return err

		}

		// Current object is not the first object or not the last object
		if currentProcessedObjects.previousTimeOutMapKey != currentProcessedObjects.currentTimeOutMapKey {

			// Insert new object before current object
			_, err = testInstructionExecutionTimeOutEngineObject.storeNewTimeOutChannelCommandObject(
				executionTrack,
				currentProcessedObjects.previousTimeOutMapKey,
				newObjectsTimeOutMapKey,
				currentTimeOutMapKey,
				newIncomingTimeOutChannelCommandObject,
				"afd9bec8-797a-4d03-bfb4-bdb05627fd8c",
				messageInitiatedFromPubSubSend)

			if err != nil {
				return err
			}

			// Update previous object regarding next object MapKey
			err = testInstructionExecutionTimeOutEngineObject.updateTimeOutChannelCommandObject(
				executionTrack,
				"",
				currentProcessedObjects.previousTimeOutMapKey,
				newObjectsTimeOutMapKey)

			if err != nil {
				return err
			}

			// Update current object regarding previous and next object MapKey
			err = testInstructionExecutionTimeOutEngineObject.updateTimeOutChannelCommandObject(
				executionTrack,
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
				executionTrack,
				currentProcessedObjects.currentTimeOutMapKey,
				newObjectsTimeOutMapKey,
				newObjectsTimeOutMapKey,
				newIncomingTimeOutChannelCommandObject,
				"7f70c268-6164-4d69-995f-a0d9c0fe465c",
				messageInitiatedFromPubSubSend)

			if err != nil {
				return err
			}

			// Update current object regarding previous and next object MapKey
			err = testInstructionExecutionTimeOutEngineObject.updateTimeOutChannelCommandObject(
				executionTrack,
				"",
				currentTimeOutMapKey,
				newObjectsTimeOutMapKey)

			return err

		} else {

			// Do recursive call to "next object"
			err = testInstructionExecutionTimeOutEngineObject.recursiveAddTestInstructionExecutionToTimeOutTimer(
				executionTrack,
				newIncomingTimeOutChannelCommandObject,
				currentProcessedObjects.nextTimeOutMapKey,
				messageInitiatedFromPubSubSend)

			return err
		}

	}

	return err
}

// Insert New object into map
func (testInstructionExecutionTimeOutEngineObject *TestInstructionTimeOutEngineObjectStruct) storeNewTimeOutChannelCommandObject(
	executionTrack int,
	previousTimeOutMapKey string,
	currentTimeOutMapKey string,
	nextTimeOutMapKey string,
	newIncomingTimeOutChannelCommandObject *common_config.TimeOutChannelCommandStruct,
	senderId string,
	messageInitiatedFromPubSubSend bool) (timeOutMapObject *timeOutMapStruct, err error) {

	common_config.Logger.WithFields(logrus.Fields{
		"id":       "90d20c2c-db78-41a8-919f-4ccd1b9d6db8",
		"senderId": senderId,
	}).Debug("Incoming 'storeNewTimeOutChannelCommandObject'")

	var existsInMap bool

	// Create Map-key for '*timeOutMapSlice[executionTrack]'
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
	// Only do this when message is initiated from PubSubSend
	if messageInitiatedFromPubSubSend == true {

		_, existsInMap = (*timeOutMapSlice[executionTrack])[currentTimeOutMapKey]
		if existsInMap == true {
			common_config.Logger.WithFields(logrus.Fields{
				"id":                   "ac69e72d-b124-43c8-a7e9-8760da369bec",
				"currentTimeOutMapKey": currentTimeOutMapKey,
			}).Error("'*timeOutMapSlice[executionTrack]' does already contain an object for for the 'currentTimeOutMapKey'")

			errorId := "af257654-4034-451d-9d1e-8385e2497264"
			err = errors.New(fmt.Sprintf("'*timeOutMapSlice[executionTrack]' does already contain an object for for the 'currentTimeOutMapKey': '%s' [ErroId: %s]",
				currentTimeOutMapKey, errorId))

			return nil, err

		}
	}

	// Store object in '*timeOutMapSlice[executionTrack]'
	(*timeOutMapSlice[executionTrack])[timeOutMapKey] = timeOutMapObject

	// Is this the first object, to Time Out
	if currentTimeOutMapKey == previousTimeOutMapKey {

		// Set variable that keeps track of next object with upcoming Timeout
		nextUpcomingObjectMapKeyWithTimeOutSlice[executionTrack] = timeOutMapKey
	}

	// Remove Allocation for TimeOut-timer
	delete(AllocatedTimeOutTimerMapSlice[executionTrack], timeOutMapKey)

	return timeOutMapObject, err
}

// Update Previous and Next MapKey for Object
func (testInstructionExecutionTimeOutEngineObject *TestInstructionTimeOutEngineObjectStruct) updateTimeOutChannelCommandObject(
	executionTrack int,
	previousTimeOutMapKey string,
	currentTimeOutMapKey string,
	nextTimeOutMapKey string) (err error) {

	common_config.Logger.WithFields(logrus.Fields{
		"id":                    "ad37804a-286b-453f-8709-ccae90cbd166",
		"previousTimeOutMapKey": previousTimeOutMapKey,
		"currentTimeOutMapKey":  currentTimeOutMapKey,
		"nextTimeOutMapKey":     nextTimeOutMapKey,
	}).Debug("Incoming 'updateTimeOutChannelCommandObject'")

	// Create object to be stored
	var timeOutMapObject *timeOutMapStruct
	var existsInMap bool

	// Verify that Object does exist in Map
	timeOutMapObject, existsInMap = (*timeOutMapSlice[executionTrack])[currentTimeOutMapKey]
	if existsInMap == false {
		common_config.Logger.WithFields(logrus.Fields{
			"id":                   "015c9dca-211d-47d9-bd4e-0a30ed144032",
			"currentTimeOutMapKey": currentTimeOutMapKey,
		}).Error("'*timeOutMapSlice[executionTrack]' doesn't have any object for 'currentTimeOutMapKey'")

		errorId := "cc40f9db-fd3f-4963-af85-c6d3a63da58a"
		err = errors.New(fmt.Sprintf("'*timeOutMapSlice[executionTrack]'doesn't have any object for 'currentTimeOutMapKey': '%s' [ErroId: %s]",
			currentTimeOutMapKey, errorId))

		return err

	}

	// Update object in '*timeOutMapSlice[executionTrack]'
	if previousTimeOutMapKey != "" {
		timeOutMapObject.previousTimeOutMapKey = previousTimeOutMapKey
	}
	if nextTimeOutMapKey != "" {
		timeOutMapObject.nextTimeOutMapKey = nextTimeOutMapKey
	}

	common_config.Logger.WithFields(logrus.Fields{
		"id":                                     "ee97ecdf-21c4-468f-8c65-78d8e0d4b10f",
		"timeOutMapObject.previousTimeOutMapKey": timeOutMapObject.previousTimeOutMapKey,
		"timeOutMapObject.currentTimeOutMapKey":  timeOutMapObject.currentTimeOutMapKey,
		"timeOutMapObject.nextTimeOutMapKey":     timeOutMapObject.nextTimeOutMapKey,
	}).Debug("Outgoing 'updateTimeOutChannelCommandObject'")

	return err
}

// Start new TimeOut-timer for this TestInstructionExecution as go-routine
func (testInstructionExecutionTimeOutEngineObject *TestInstructionTimeOutEngineObjectStruct) startTimeOutTimerTestInstructionExecution(
	executionTrack int,
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

	// Add reference to CancellableTimer in '*timeOutMapSlice[executionTrack]-object'
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
		go func() {
			testInstructionExecutionEngine.ExecutionEngineCommandChannelSlice[executionTrack] <- executionEngineChannelCommand
		}()

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
			MessageInitiatedFromPubSubSend:          false,
		}

		// Send message on TimeOutEngineChannel to remove TestInstructionExecution from Timer-queue
		TimeOutChannelEngineCommandChannelSlice[executionTrack] <- tempTimeOutChannelCommand
	}

	return

}
