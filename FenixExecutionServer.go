package main

import (
	"FenixExecutionServer/broadcastingEngine"
	"FenixExecutionServer/common_config"
	"FenixExecutionServer/testInstructionExecutionEngine"
	"FenixExecutionServer/testInstructionTimeOutEngine"
	"fmt"
	fenixSyncShared "github.com/jlambert68/FenixSyncShared"
	"github.com/sirupsen/logrus"
	"os"
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
	minutesToWait = 15 * time.Minute

	for {

		select {
		case incomingGrpcMessageTimeStamp := <-endApplicationWhenNoIncomingGrpcCalls:
			fmt.Println(incomingGrpcMessageTimeStamp)

			// If incoming TimeStamp is after current then set that as new latest timestamp
			if incomingGrpcMessageTimeStamp.After(latestGrpcMessageTimeStamp) {
				latestGrpcMessageTimeStamp = incomingGrpcMessageTimeStamp
			}

		case <-time.After(minutesToWait):
			fmt.Println("timeout: " + minutesToWait.String())

			// If It has gone 15 minutes since last gPRC-message then end application
			// Or restart gRPC-server if application runs locally
			var timeDurationSinceLatestGrpcCall time.Duration
			timeDurationSinceLatestGrpcCall = time.Now().Sub(latestGrpcMessageTimeStamp)
			if timeDurationSinceLatestGrpcCall < 0 {
				// Nothing has happened for 15 minutes

				/*
					var executionLocation common_config.ExecutionLocationTypeType
					executionLocation = common_config.ExecutionLocationForFenixExecutionServer
					select  executionLocation {
					case comm
					}


				*/
				fenixExecutionServerObject.StopGrpcServer()

			} else {
				// Less than 15 minutes so set new shorter timer
				minutesToWait = time.Duration(time.Minute*15) - timeDurationSinceLatestGrpcCall
			}
		}

	}

}
