package main

import (
	"FenixExecutionServer/broadcastingEngine"
	"FenixExecutionServer/common_config"
	"FenixExecutionServer/testInstructionExecutionEngine"
	"FenixExecutionServer/testInstructionTimeOutEngine"
	fenixSyncShared "github.com/jlambert68/FenixSyncShared"
	"github.com/sirupsen/logrus"
	"os"
	"sync"
	"time"
)

// Used for only process cleanup once
var cleanupProcessed = false

func cleanup() {

	if cleanupProcessed == false {

		cleanupProcessed = true

		// Cleanup before close down application
		fenixExecutionServerObject.logger.WithFields(logrus.Fields{}).Info("Clean up and shut down servers")

		// Stop Backend gRPC Server
		fenixExecutionServerObject.StopGrpcServer()

		//log.Println("Close DB_session: %v", DB_session)
		//DB_session.Close()
	}
}

func fenixExecutionServerMain() {

	// Connect to CloudDB
	fenixSyncShared.ConnectToDB()

	// Set up BackendObject
	fenixExecutionServerObject = &fenixExecutionServerObjectStruct{
		logger:                         nil,
		gcpAccessToken:                 nil,
		executionEngineChannelRefSlice: nil,
		executionEngine:                &testInstructionExecutionEngine.TestInstructionExecutionEngineStruct{},
	}

	// Init logger
	fenixExecutionServerObject.InitLogger("log83.log")

	// Clean up when leaving. Is placed after logger because shutdown logs information
	defer cleanup()

	// Create one instance per execution track for channel
	for executionTrackNumber := 0; executionTrackNumber < common_config.NumberOfParallellExecutionEngineCommandChannels; executionTrackNumber++ {

		// Create Channel used for sending Commands to TestInstructionExecutionCommandsEngine
		var executionEngineCommandChannel testInstructionExecutionEngine.ExecutionEngineChannelType
		executionEngineCommandChannel = make(chan testInstructionExecutionEngine.ChannelCommandStruct, testInstructionExecutionEngine.ExecutionEngineChannelSize)

		// Append to 'ExecutionEngineCommandChannelSlice'
		testInstructionExecutionEngine.ExecutionEngineCommandChannelSlice = append(
			testInstructionExecutionEngine.ExecutionEngineCommandChannelSlice,
			executionEngineCommandChannel)

	}

	// Create reference to channel and append to fenixExecutionServerObject-slice
	myCommandChannelRefSlice := &testInstructionExecutionEngine.ExecutionEngineCommandChannelSlice

	//
	fenixExecutionServerObject.executionEngineChannelRefSlice = myCommandChannelRefSlice

	// Initiate logger in TestInstructionEngine
	fenixExecutionServerObject.executionEngine.SetLogger(fenixExecutionServerObject.logger)

	// Store Logger in common_config variable
	common_config.Logger = fenixExecutionServerObject.logger

	// Load Domain Worker Addresses
	err := fenixExecutionServerObject.prepareGetDomainWorkerAddresses()
	if err != nil {
		fenixExecutionServerObject.logger.WithFields(logrus.Fields{
			"Id":    "85824093-ae12-4d22-8f9e-1b66a1bf2bbd",
			"Error": err,
		}).Error("Couldn't load Domain Worker Data from Cloud-DB, exiting")

		os.Exit(0)
	}

	// Start BroadcastingEngine, for sending info about when a TestCaseExecutionUuid or TestInstructionExecution has updated
	//its status. Messages are sent to BroadcastEngine using channels
	go broadcastingEngine.InitiateAndStartBroadcastNotifyEngine()

	// Start Receiver channel for Commands
	fenixExecutionServerObject.executionEngine.InitiateTestInstructionExecutionEngineCommandChannelReader(*myCommandChannelRefSlice)

	// Start Receiver channel for TimeOutEngine
	testInstructionTimeOutEngine.TestInstructionExecutionTimeOutEngineObject.InitiateTestInstructionExecutionTimeOutEngineChannelReader()

	// Initiate Channel used to decide when application should end, due to no gRPC-activity
	endApplicationWhenNoIncomingGrpcCalls = make(chan time.Time, 100)

	// Start Backend gRPC-server
	go fenixExecutionServerObject.InitGrpcServer()

	// When did the last gRPC-message arrived
	var latestGrpcMessageTimeStamp time.Time
	latestGrpcMessageTimeStamp = time.Now()

	// Minutes To Wait before ending application
	var minutesToWait time.Duration
	minutesToWait = common_config.MinutesToShutDownWithOutAnyGrpcTraffic

	// Wait group that keeps track that there was a respons from all response channels
	var waitGroup sync.WaitGroup

	// Array that save responses from TimeOutEngine
	var timeOutResponsDurations []time.Duration

	for {

		// Either full time and then shut down or adjust time when incoming due to gRPC-traffic
		select {

		// Each incoming gRPC-call will set new baseline for waiting
		case incomingGrpcMessageTimeStamp := <-endApplicationWhenNoIncomingGrpcCalls:

			// If incoming TimeStamp is after current then set that as new latest timestamp
			if incomingGrpcMessageTimeStamp.After(latestGrpcMessageTimeStamp) {
				latestGrpcMessageTimeStamp = incomingGrpcMessageTimeStamp
			}

			// If no incoming gRPC-traffic then this will be triggered
		case <-time.After(minutesToWait):

			// If It has gone 'common_config.MinutesToShutDownWithOutAnyGrpcTraffic' minutes
			// since last gPRC-message then end application
			// Or just restart gRPC-server if application runs locally
			var timeDurationSinceLatestGrpcCall time.Duration
			timeDurationSinceLatestGrpcCall = time.Now().Sub(latestGrpcMessageTimeStamp)
			if timeDurationSinceLatestGrpcCall < common_config.MinutesToShutDownWithOutAnyGrpcTraffic {
				// Nothing has happened for 'MinutesToShutDownWithOutAnyGrpcTraffic' so check if there are any timer
				// that should soon fire

				// Initiate array for TimeOutEngine-responses
				timeOutResponsDurations = make([]time.Duration, common_config.NumberOfParallellTimeOutChannels)

				// Wait group should wait for all Channels to respond
				waitGroup.Add(common_config.NumberOfParallellTimeOutChannels)

				// Loop all TimeOut-tracks and send request for when next timeout occurs
				for timeOutChannelCounter := 0; timeOutChannelCounter < common_config.NumberOfParallellTimeOutChannels; timeOutChannelCounter++ {

					// Start question to each TimeOut-time as goroutine
					go func(timeOutChannelCounter int) {

						// Create Response channel from TimeOutEngine to get answer of durantion until next timeout
						var timeOutResponseChannelForDurationUntilTimeOutOccurs common_config.TimeOutResponseChannelForDurationUntilTimeOutOccursType
						timeOutResponseChannelForDurationUntilTimeOutOccurs = make(chan common_config.TimeOutResponseChannelForDurationUntilTimeOutOccursStruct)

						var tempTimeOutChannelCommand common_config.TimeOutChannelCommandStruct
						tempTimeOutChannelCommand = common_config.TimeOutChannelCommandStruct{
							TimeOutChannelCommand:                               common_config.TimeOutChannelCommandTimeUntilNextTimeOutTimerToFires,
							TimeOutResponseChannelForDurationUntilTimeOutOccurs: &timeOutResponseChannelForDurationUntilTimeOutOccurs,

							SendID: "c208cca2-036c-449d-a997-00788c944441",
						}

						// Send message on TimeOutEngineChannel to get information about if TestInstructionExecution already has TimedOut
						*common_config.TimeOutChannelEngineCommandChannelReferenceSlice[timeOutChannelCounter] <- tempTimeOutChannelCommand

						// Response from TimeOutEngine
						var timeOutResponseChannelForDurationUntilTimeOutOccursResponse common_config.TimeOutResponseChannelForDurationUntilTimeOutOccursStruct

						// Wait for response from TimeOutEngine
						timeOutResponseChannelForDurationUntilTimeOutOccursResponse = <-timeOutResponseChannelForDurationUntilTimeOutOccurs

						// Save response duration in array for all TimeOut-durations
						timeOutResponsDurations[timeOutChannelCounter] = timeOutResponseChannelForDurationUntilTimeOutOccursResponse.DurationUntilTimeOutOccurs

						// This goroutine is finished
						waitGroup.Done()

					}(timeOutChannelCounter)

				}

				// Wait for all goroutines to get answers over channels
				waitGroup.Wait()

				// Extract the lowest duration from TimeOutEngine
				var lowestDuration time.Duration
				var lowestDurationFirstValueFound bool = false

				// Loop the all timeOutDuration-responses
				for _, timeOutResponsDuration := range timeOutResponsDurations {

					// '-1' as duration is used when there was an error or if there is no timeout-timer for this track
					// Then just ignore that one, otherwise process
					if timeOutResponsDuration != time.Duration(-1) {

						// When first duration then just copy that value
						if lowestDurationFirstValueFound == false {

							lowestDurationFirstValueFound = true
							lowestDuration = timeOutResponsDuration

						} else {

							// Is this duration lower than the previous lowest found, then this one is the new lowest duration
							if timeOutResponsDuration < lowestDuration {
								lowestDuration = timeOutResponsDuration
							}
						}
					}
				}

				// Only act when all TimeOut-timers has more than 'MaxMinutesLeftUntilNextTimeOutTimer' left before timeout
				if lowestDuration > common_config.MaxMinutesLeftUntilNextTimeOutTimer {

					// If ExecutionServer runs in GCP the end application
					if common_config.ExecutionLocationForFenixExecutionServer == common_config.GCP {
						break
					}

					// Application runs locally then restart gRPC-server
					if common_config.ExecutionLocationForFenixExecutionServer == common_config.LocalhostNoDocker {
						fenixExecutionServerObject.StopGrpcServer()
						go fenixExecutionServerObject.InitGrpcServer()

						// Reset max waiting time because application is running locally
						minutesToWait = common_config.MinutesToShutDownWithOutAnyGrpcTraffic
						latestGrpcMessageTimeStamp = time.Now()
					}
				} else {

					// Reset max waiting time because there is at least one upcoming TimeOut
					minutesToWait = common_config.MinutesToShutDownWithOutAnyGrpcTraffic
					latestGrpcMessageTimeStamp = time.Now()
				}
			}
		}
	}
}
