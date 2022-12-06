package main

import (
	"FenixExecutionServer/common_config"
	"FenixExecutionServer/testInstructionExecutionEngine"
	"context"
	fenixExecutionServerGrpcApi "github.com/jlambert68/FenixGrpcApi/FenixExecutionServer/fenixExecutionServerGrpcApi/go_grpc_api"
	"github.com/sirupsen/logrus"
)

// InformThatThereAreNewTestInstructionsOnExecutionQueue - *********************************************************************
// ExecutionServerGui-server inform ExecutionServer that there is a new TestCase that is ready on the Execution-queue
func (s *fenixExecutionServerGrpcServicesServer) InformThatThereAreNewTestInstructionsOnExecutionQueue(ctx context.Context, testCaseExecutionsToProcessMessage *fenixExecutionServerGrpcApi.TestCaseExecutionsToProcessMessage) (*fenixExecutionServerGrpcApi.AckNackResponse, error) {

	fenixExecutionServerObject.logger.WithFields(logrus.Fields{
		"id": "7ceb7c60-e90e-40ea-92c7-7cc5becb0d98",
	}).Debug("Incoming 'gRPC - InformThatThereAreNewTestInstructionsOnExecutionQueue'")

	defer fenixExecutionServerObject.logger.WithFields(logrus.Fields{
		"id": "38224ef0-060d-4b64-b4ff-f1f68939b53b",
	}).Debug("Outgoing 'gRPC - InformThatThereAreNewTestInstructionsOnExecutionQueue'")

	// Current user
	userID := "gRPC-api doesn't support UserId"

	// Check if Client is using correct proto files version
	returnMessage := common_config.IsClientUsingCorrectTestDataProtoFileVersion(userID, fenixExecutionServerGrpcApi.CurrentFenixExecutionServerProtoFileVersionEnum(testCaseExecutionsToProcessMessage.ProtoFileVersionUsedByClient))
	if returnMessage != nil {

		// Exiting
		return returnMessage, nil
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
	/*
		// Trigger TestInstructionEngine to check if there are any TestInstructions on the ExecutionQueue
		channelCommandMessage := testInstructionExecutionEngine.ChannelCommandStruct{
			ChannelCommand:                   testInstructionExecutionEngine.ChannelCommandCheckForTestInstructionExecutionWaitingOnQueue,
			ChannelCommandTestCaseExecutions: channelCommandTestCasesExecution,
		}



		// Send Message on Channel
		*fenixExecutionServerObject.executionEngineChannelRef <- channelCommandMessage
	*/
	return &fenixExecutionServerGrpcApi.AckNackResponse{AckNack: true, Comments: ""}, nil
}
