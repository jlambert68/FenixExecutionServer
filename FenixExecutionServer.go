package main

import (
	"FenixExecutionServer/testInstructionExecutionEngine"
	fenixSyncShared "github.com/jlambert68/FenixSyncShared"
	"github.com/sirupsen/logrus"
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
	fenixExecutionServerObject.InitLogger("")

	// Clean up when leaving. Is placed after logger because shutdown logs information
	defer cleanup()

	// Create Channel used for sending Commands to TestInstructionExecutionCommandsEngine
	testInstructionExecutionEngine.ExecutionEngineCommandChannel = make(chan testInstructionExecutionEngine.ChannelCommandStruct)
	myCommandChannelRef := &testInstructionExecutionEngine.ExecutionEngineCommandChannel
	fenixExecutionServerObject.executionEngineChannelRef = myCommandChannelRef

	// Initiate logger in TestInstructionEngine
	fenixExecutionServerObject.executionEngine.SetLogger(fenixExecutionServerObject.logger)

	// Start Receiver channel for Commands
	fenixExecutionServerObject.executionEngine.InitiateTestInstructionExecutionEngineCommandChannelReader(*myCommandChannelRef)

	// Start Backend gRPC-server
	fenixExecutionServerObject.InitGrpcServer()

}
