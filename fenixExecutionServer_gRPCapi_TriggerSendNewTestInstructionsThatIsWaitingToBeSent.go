package main

import (
	"FenixExecutionServer/common_config"
	"FenixExecutionServer/testInstructionExecutionEngine"
	"context"
	fenixExecutionServerGrpcApi "github.com/jlambert68/FenixGrpcApi/FenixExecutionServer/fenixExecutionServerGrpcApi/go_grpc_api"
	"github.com/sirupsen/logrus"
	"time"
)

// TriggerSendNewTestInstructionsThatIsWaitingToBeSent - *********************************************************************
// Used to trigger/re-trigger sending new TestInstructionExecutions to workers
func (s *fenixExecutionServerGrpcServicesServer) TriggerSendNewTestInstructionsThatIsWaitingToBeSent(ctx context.Context, testCaseExecutionsToProcessMessage *fenixExecutionServerGrpcApi.TestCaseExecutionsToProcessMessage) (*fenixExecutionServerGrpcApi.AckNackResponse, error) {

	fenixExecutionServerObject.logger.WithFields(logrus.Fields{
		"id": "3aede4ca-356b-43cd-a21a-5fe0a606bc3d",
	}).Debug("Incoming 'gRPC - TriggerSendNewTestInstructionsThatIsWaitingToBeSent'")

	defer fenixExecutionServerObject.logger.WithFields(logrus.Fields{
		"id": "22d31274-ee65-4255-8de7-7188eab13bdb",
	}).Debug("Outgoing 'gRPC - TriggerSendNewTestInstructionsThatIsWaitingToBeSent'")

	// Reset Application shut time timer when there is an incoming gRPC-call
	var endApplicationWhenNoIncomingGrpcCallsSenderData endApplicationWhenNoIncomingGrpcCallsStruct
	endApplicationWhenNoIncomingGrpcCallsSenderData = endApplicationWhenNoIncomingGrpcCallsStruct{
		gRPCTimeStamp: time.Now(),
		senderName:    "gRPC - TriggerSendNewTestInstructionsThatIsWaitingToBeSent",
	}
	endApplicationWhenNoIncomingGrpcCalls <- endApplicationWhenNoIncomingGrpcCallsSenderData

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
			TestCaseExecutionUuid:          testCaseExecutionToProcessMessage.TestCaseExecutionsUuid,
			TestCaseExecutionVersion:       testCaseExecutionToProcessMessage.TestCaseExecutionVersion,
			ExecutionStatusReportLevelEnum: fenixExecutionServerGrpcApi.TestInstructionExecutionStatusEnum(testCaseExecutionToProcessMessage.ExecutionStatusReportLevel),
		}
		channelCommandTestCasesExecution = append(channelCommandTestCasesExecution, channelCommandTestCaseExecution)
	}
	/*
		// Trigger TestInstructionEngine to check if there are any TestInstructions on the ExecutionQueue
		channelCommandMessage := testInstructionExecutionEngine.ChannelCommandStruct{
			ChannelCommand:                   testInstructionExecutionEngine.ChannelCommandCheckForTestInstructionExecutionsWaitingToBeSentToWorker,
			ChannelCommandTestCaseExecutions: channelCommandTestCasesExecution,
		}

		// Send Message on Channel
		*fenixExecutionServerObject.executionEngineChannelRef <- channelCommandMessage


	*/
	return &fenixExecutionServerGrpcApi.AckNackResponse{AckNack: true, Comments: ""}, nil
}
