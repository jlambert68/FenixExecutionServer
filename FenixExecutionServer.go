package main

import (
	"FenixExecutionServer/broadcastingEngine"
	"FenixExecutionServer/common_config"
	"FenixExecutionServer/testInstructionExecutionEngine"
	"FenixExecutionServer/testInstructionTimeOutEngine"
	fenixSyncShared "github.com/jlambert68/FenixSyncShared"
	"github.com/sirupsen/logrus"
	"os"
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
		logger:                    nil,
		gcpAccessToken:            nil,
		executionEngineChannelRef: nil,
		executionEngine:           &testInstructionExecutionEngine.TestInstructionExecutionEngineStruct{},
	}

	// Init logger
	fenixExecutionServerObject.InitLogger("log20.log")

	// Clean up when leaving. Is placed after logger because shutdown logs information
	defer cleanup()

	// Create Channel used for sending Commands to TestInstructionExecutionCommandsEngine
	testInstructionExecutionEngine.ExecutionEngineCommandChannel = make(chan testInstructionExecutionEngine.ChannelCommandStruct, testInstructionExecutionEngine.ExecutionEngineChannelSize)
	myCommandChannelRef := &testInstructionExecutionEngine.ExecutionEngineCommandChannel
	fenixExecutionServerObject.executionEngineChannelRef = myCommandChannelRef

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
	fenixExecutionServerObject.executionEngine.InitiateTestInstructionExecutionEngineCommandChannelReader(*myCommandChannelRef)

	// Start Receiver channel for TimeOutEngine
	testInstructionTimeOutEngine.TestInstructionExecutionTimeOutEngineObject.InitiateTestInstructionExecutionTimeOutEngineChannelReader()

	// Start Backend gRPC-server
	fenixExecutionServerObject.InitGrpcServer()

}
