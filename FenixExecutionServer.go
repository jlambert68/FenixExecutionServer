package main

import (
	"FenixExecutionServer/broadcastEngine_TestInstructionNotHandledByThisInstance"
	"FenixExecutionServer/broadcastingEngine_ExecutionStatusUpdate"
	"FenixExecutionServer/common_config"
	"FenixExecutionServer/testInstructionExecutionEngine"
	"FenixExecutionServer/testInstructionTimeOutEngine"
	"cloud.google.com/go/firestore"
	"fmt"
	"github.com/jlambert68/FenixScriptEngine/luaEngine"
	fenixSyncShared "github.com/jlambert68/FenixSyncShared"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"
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

		// Close Database Connection
		fenixExecutionServerObject.logger.WithFields(logrus.Fields{
			"Id": "01fab57b-4961-4209-b38b-068471b7ec2b",
		}).Info("Closing Database connection")

		fenixSyncShared.DbPool.Close()
	}
}

func fenixExecutionServerMain() {

	// Connect to CloudDB
	fenixSyncShared.ConnectToDB()

	// Initiate Lua-script-Engine. TODO For now only Fenix-Placeholders are supported
	luaEngine.InitiateLuaScriptEngine([][]byte{})

	// Start cleaner for ExecutionStatus-message on 'DeadLettering', when it should be used
	if common_config.UsePubSubWhenSendingExecutionStatusToGuiExecutionServer == true {
		go broadcastingEngine_ExecutionStatusUpdate.PullPubSubExecutionStatusMessagesFromDeadLettering()
	}

	// Set up BackendObject
	fenixExecutionServerObject = &fenixExecutionServerObjectStruct{
		logger:                         nil,
		gcpAccessToken:                 nil,
		executionEngineChannelRefSlice: nil,
		executionEngine:                &testInstructionExecutionEngine.TestInstructionExecutionEngineStruct{},
	}

	// Init logger
	if common_config.ExecutionLocationForFenixExecutionServer == common_config.GCP {
		// Always use standard output in GCP to be able for GCP to pick up logs
		fenixExecutionServerObject.InitLogger("")
	} else {
		fenixExecutionServerObject.InitLogger("")
	}

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
	go broadcastingEngine_ExecutionStatusUpdate.InitiateAndStartBroadcastNotifyEngine_ExecutionStatusUpdate()

	// Start listen for Broadcasts regarding change when a TestInstructionExecution is handled by other ExecutionInstance
	broadcastEngine_TestInstructionNotHandledByThisInstance.InitiateAndStartBroadcastNotifyEngine()

	// Start BroadcastingEngine, for sending info about when a TestInstructionExecution is handled by other ExecutionInstance
	//its status. Messages are sent to BroadcastEngine using channels
	go broadcastEngine_TestInstructionNotHandledByThisInstance.InitiateAndStartBroadcastNotifyEngine_TestInstructionNotHandledByThisInstance()

	// Start Receiver channel for TimeOutEngine
	testInstructionTimeOutEngine.TestInstructionExecutionTimeOutEngineObject.InitiateTestInstructionExecutionTimeOutEngineChannelReader()

	// Start Receiver channel for Commands
	fenixExecutionServerObject.executionEngine.InitiateTestInstructionExecutionEngineCommandChannelReader(*myCommandChannelRefSlice)

	// Initiate Channel used to decide when application should end, due to no gRPC-activity
	endApplicationWhenNoIncomingGrpcCalls = make(chan endApplicationWhenNoIncomingGrpcCallsStruct, 100)

	// Start Backend gRPC-server
	go fenixExecutionServerObject.InitGrpcServer()

	// Minutes To Wait before ending application
	var minutesToWait time.Duration
	minutesToWait = common_config.MinutesToShutDownWithOutAnyGrpcTraffic

	// Wait group that keeps track that there was a respons from all response channels
	var waitGroup sync.WaitGroup

	// Array that save responses from TimeOutEngine
	var timeOutResponsDurations []time.Duration

	//
	for {

		// Either full time and then shut down or adjust time when incoming due to gRPC-traffic
		select {

		// Each incoming gRPC-call will set new baseline for waiting
		case incomingGrpcMessageTimeStamp := <-endApplicationWhenNoIncomingGrpcCalls:
			fmt.Println("incoming gRPC-call: " + incomingGrpcMessageTimeStamp.gRPCTimeStamp.String() + " from " + incomingGrpcMessageTimeStamp.senderName)

			fenixExecutionServerObject.logger.WithFields(logrus.Fields{
				"Id":            "86a96eb3-472f-4b19-b01c-ad925cba1e59",
				"gRPCTimeStamp": incomingGrpcMessageTimeStamp.gRPCTimeStamp.String(),
				"senderName":    incomingGrpcMessageTimeStamp.senderName,
			}).Debug("incoming gRPC-call")

			// If no incoming gRPC-traffic then this will be triggered
		case <-time.After(minutesToWait):
			fmt.Println("Waited : " + minutesToWait.String())

			fenixExecutionServerObject.logger.WithFields(logrus.Fields{
				"Id":            "2bde0570-6312-4476-b21d-9299eb535147",
				"minutesToWait": minutesToWait.String(),
			}).Debug("If no incoming gRPC-traffic then this will be triggered")

			// Initiate array for TimeOutEngine-responses
			timeOutResponsDurations = make([]time.Duration, common_config.NumberOfParallellTimeOutChannels)

			// Wait group should wait for all Channels to respond
			waitGroup.Add(common_config.NumberOfParallellTimeOutChannels)

			// Loop all TimeOut-tracks and send request for when next timeout occurs
			for timeOutChannelCounter := 0; timeOutChannelCounter < common_config.NumberOfParallellTimeOutChannels; timeOutChannelCounter++ {

				// Start question to each TimeOut-time as goroutine
				go func(timeOutChannelCounter int) {

					// Create Response channel from TimeOutEngine to get answer of duration until next timeout
					var timeOutResponseChannelForDurationUntilTimeOutOccurs common_config.TimeOutResponseChannelForDurationUntilTimeOutOccursType
					timeOutResponseChannelForDurationUntilTimeOutOccurs = make(chan common_config.TimeOutResponseChannelForDurationUntilTimeOutOccursStruct)

					var tempTimeOutChannelCommand common_config.TimeOutChannelCommandStruct
					tempTimeOutChannelCommand = common_config.TimeOutChannelCommandStruct{
						TimeOutChannelCommand:                               common_config.TimeOutChannelCommandTimeUntilNextTimeOutTimerToFires,
						TimeOutResponseChannelForDurationUntilTimeOutOccurs: &timeOutResponseChannelForDurationUntilTimeOutOccurs,

						SendID:                         "c208cca2-036c-449d-a997-00788c944441",
						MessageInitiatedFromPubSubSend: false,
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
			// Or there were no TimeOut-timers left at all
			if lowestDuration > common_config.MaxMinutesLeftUntilNextTimeOutTimer || lowestDurationFirstValueFound == false {

				// If ExecutionServer runs in GCP then first Store timestamp for next wakeup time, and then end application
				//if common_config.ExecutionLocationForFenixExecutionServer != common_config.GCP { // Local
				if common_config.ExecutionLocationForFenixExecutionServer == common_config.GCP { // GCP

					// Only store data in FireStore-DB when there are ongoing TimeOut-timers
					if lowestDurationFirstValueFound == true {

						// Store TimeStamp in Firestore-DB
						ctx := context.Background()

						//opt := option. WithCredentialsFile("../../service_account.json")
						fireStoreClient, err := firestore.NewClient(ctx, common_config.GCPProjectId) //, opt)
						if err != nil {
							fenixExecutionServerObject.logger.WithFields(logrus.Fields{
								"Id":  "348c45b2-cdb3-434d-accf-a0837e88552f",
								"err": err,
							}).Error("Problem when creating new FireStore client")
						}
						defer fireStoreClient.Close()

						// If no TimeOut-timer exist then don't store any data in FireStore-DB

						// Convert Wakeup timestamp into string
						var wakeupTimeStamp time.Time
						var wakeupTimeStampAsString string

						// Wakeup time = Now() + "Duration until next Timer" - "Time Before timer should start"
						wakeupTimeStamp = time.Now().Add(lowestDuration).Add(common_config.NumberOfMinutesBeforeNextTimeOutTimerToStart)
						wakeupTimeStampAsString = wakeupTimeStamp.Format("2006-01-02 15:04:05 -0700 MST")

						// Store TimeStamp in FireStore-Db
						_, err = fireStoreClient.Collection("nextexecutionservertimeouttimer").Doc("timeouttimestamp").Set(ctx, map[string]interface{}{
							"mytimestamp": wakeupTimeStampAsString,
						})
						if err != nil {
							fenixExecutionServerObject.logger.WithFields(logrus.Fields{
								"Id":                      "6921816c-3fea-46ee-8f32-04526e1636ec",
								"err":                     err,
								"wakeupTimeStampAsString": wakeupTimeStampAsString,
							}).Error("Problem when storing timestamp into FireStore-DB")
						} else {

							// Log What time to be woken up by GCP cron job
							fenixExecutionServerObject.logger.WithFields(logrus.Fields{
								"Id":                      "83363cbe-684a-48ab-b142-a12ba7e96b99",
								"wakeupTimeStampAsString": wakeupTimeStampAsString,
							}).Info("Expected to be woken up at this time by the GCP Cron Job")
						}

					}

					// Log that ExecutionServer is shutting down
					fenixExecutionServerObject.logger.WithFields(logrus.Fields{
						"Id":                     "b77ddbb8-3834-4591-ba59-eea2ff7affd5",
						"ApplicationRuntimeUuid": common_config.ApplicationRuntimeUuid,
					}).Info("Fenix ExecutionServer is shutting down")

					//TODO fix so when application i shut down by GCP then run code for saving Timers to FireStore-DB
					// End Application
					os.Exit(0)
				}

				// Application runs locally then restart gRPC-server
				if common_config.ExecutionLocationForFenixExecutionServer == common_config.LocalhostNoDocker {
					fenixExecutionServerObject.StopGrpcServer()
					go fenixExecutionServerObject.InitGrpcServer()

					// Reset max waiting time because application is running locally
					minutesToWait = common_config.MinutesToShutDownWithOutAnyGrpcTraffic

				}
			} else {

				// Reset max waiting time because there is at least one upcoming TimeOut
				minutesToWait = common_config.MinutesToShutDownWithOutAnyGrpcTraffic

			}

		}
	}
}
