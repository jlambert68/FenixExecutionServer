package main

import (
	"FenixExecutionServer/common_config"
	"FenixExecutionServer/testInstructionExecutionEngine"
	"context"
	fenixExecutionServerGrpcApi "github.com/jlambert68/FenixGrpcApi/FenixExecutionServer/fenixExecutionServerGrpcApi/go_grpc_api"
	"github.com/sirupsen/logrus"
	"sort"
	"time"
)

// InformThatThereAreNewTestCasesOnExecutionQueue - *********************************************************************
// ExecutionServerGui-server inform ExecutionServer that there is a new TestCase that is ready on the Execution-queue
func (s *fenixExecutionServerGrpcServicesServer) InformThatThereAreNewTestCasesOnExecutionQueue(ctx context.Context, testCaseExecutionsToProcessMessage *fenixExecutionServerGrpcApi.TestCaseExecutionsToProcessMessage) (*fenixExecutionServerGrpcApi.AckNackResponse, error) {

	fenixExecutionServerObject.logger.WithFields(logrus.Fields{
		"id":                                 "862cb663-daea-4f33-9f6e-03594d3005df",
		"testCaseExecutionsToProcessMessage": testCaseExecutionsToProcessMessage,
	}).Debug("Incoming 'gRPC - InformThatThereAreNewTestCasesOnExecutionQueue'")

	defer fenixExecutionServerObject.logger.WithFields(logrus.Fields{
		"id": "6507f7a9-4def-4a38-90c7-7bb19311f10f",
	}).Debug("Outgoing 'gRPC - InformThatThereAreNewTestCasesOnExecutionQueue'")

	// Reset Application shut time timer when there is an incoming gRPC-call
	endApplicationWhenNoIncomingGrpcCalls <- time.Now()

	// Current user
	userID := "gRPC-api doesn't support UserId"

	// Check if Client is using correct proto files version
	returnMessage := common_config.IsClientUsingCorrectTestDataProtoFileVersion(userID, fenixExecutionServerGrpcApi.CurrentFenixExecutionServerProtoFileVersionEnum(testCaseExecutionsToProcessMessage.ProtoFileVersionUsedByClient))
	if returnMessage != nil {

		// Exiting
		return returnMessage, nil
	}

	// If no TestCaseExecutions then return error
	if len(testCaseExecutionsToProcessMessage.TestCaseExecutionsToProcess) == 0 {
		return &fenixExecutionServerGrpcApi.AckNackResponse{AckNack: false, Comments: "Can't accept zero TestCaseExecutions in request"}, nil
	}

	// Convert TestCaseExecutions to process from gRPC-format into message format used within channel
	var channelCommandTestCasesExecution []testInstructionExecutionEngine.ChannelCommandTestCaseExecutionStruct
	for _, testCaseExecutionToProcessMessage := range testCaseExecutionsToProcessMessage.TestCaseExecutionsToProcess {
		var channelCommandTestCaseExecution testInstructionExecutionEngine.ChannelCommandTestCaseExecutionStruct
		channelCommandTestCaseExecution = testInstructionExecutionEngine.ChannelCommandTestCaseExecutionStruct{
			TestCaseExecutionUuid:    testCaseExecutionToProcessMessage.TestCaseExecutionsUuid,
			TestCaseExecutionVersion: testCaseExecutionToProcessMessage.TestCaseExecutionVersion,
		}
		channelCommandTestCasesExecution = append(channelCommandTestCasesExecution, channelCommandTestCaseExecution)
	}

	// Create TestInstructions to be saved on 'TestInstructionExecutionQueue'
	//	returnMessage = fenixExecutionServerObject.prepareInformThatThereAreNewTestCasesOnExecutionQueueSaveToCloudDB(channelCommandTestCasesExecution)

	// Send TestCaseExecutions to ExecutionEngineChannel
	channelCommandMessage := testInstructionExecutionEngine.ChannelCommandStruct{
		ChannelCommand:                   testInstructionExecutionEngine.ChannelCommandProcessTestCaseExecutionsOnExecutionQueue,
		ChannelCommandTestCaseExecutions: channelCommandTestCasesExecution,
	}

	// Extract "lowest" TestCaseExecutionUuid
	var executionTrackNumber int

	if len(channelCommandTestCasesExecution) > 0 {
		var uuidSlice []string
		for _, uuid := range channelCommandTestCasesExecution {
			uuidSlice = append(uuidSlice, uuid.TestCaseExecutionUuid)
		}
		sort.Strings(uuidSlice)

		// Define Execution Track based on "lowest "TestCaseExecutionUuid
		executionTrackNumber = common_config.CalculateExecutionTrackNumber(uuidSlice[0])
	}

	// Send Message on Channel
	var executionEngineChannelEngineChannelSlice []testInstructionExecutionEngine.ExecutionEngineChannelType
	executionEngineChannelEngineChannelSlice = *fenixExecutionServerObject.executionEngineChannelRefSlice
	executionEngineChannelEngineChannelSlice[executionTrackNumber] <- channelCommandMessage

	/*
		returnMessage = testInstructionExecutionEngine.TestInstructionExecutionEngineObject.PrepareInformThatThereAreNewTestCasesOnExecutionQueueSaveToCloudDB(
			channelCommandTestCasesExecution)

		if returnMessage != nil {
			// No errors when moving TestCases from queue into executing TestCases, so then start process TestInstruction

			return returnMessage, nil
		}
	*/

	return &fenixExecutionServerGrpcApi.AckNackResponse{AckNack: true, Comments: ""}, nil
}
