package main

import (
	"FenixExecutionServer/common_config"
	"FenixExecutionServer/testInstructionExecutionEngine"
	"context"
	fenixExecutionServerGrpcApi "github.com/jlambert68/FenixGrpcApi/FenixExecutionServer/fenixExecutionServerGrpcApi/go_grpc_api"
	"github.com/sirupsen/logrus"
)

// TriggerSendNewTestInstructionsThatIsWaitingToBeSent - *********************************************************************
// Used to trigger/re-trigger sending new TestInstructionExecutions to workers
func (s *fenixExecutionServerGrpcServicesServer) TriggerSendNewTestInstructionsThatIsWaitingToBeSent(ctx context.Context, emptyParameter *fenixExecutionServerGrpcApi.EmptyParameter) (*fenixExecutionServerGrpcApi.AckNackResponse, error) {

	fenixExecutionServerObject.logger.WithFields(logrus.Fields{
		"id": "3aede4ca-356b-43cd-a21a-5fe0a606bc3d",
	}).Debug("Incoming 'gRPC - TriggerSendNewTestInstructionsThatIsWaitingToBeSent'")

	defer fenixExecutionServerObject.logger.WithFields(logrus.Fields{
		"id": "22d31274-ee65-4255-8de7-7188eab13bdb",
	}).Debug("Outgoing 'gRPC - TriggerSendNewTestInstructionsThatIsWaitingToBeSent'")

	// Current user
	userID := "gRPC-api doesn't support UserId"

	// Check if Client is using correct proto files version
	returnMessage := common_config.IsClientUsingCorrectTestDataProtoFileVersion(userID, fenixExecutionServerGrpcApi.CurrentFenixExecutionServerProtoFileVersionEnum(emptyParameter.ProtoFileVersionUsedByClient))
	if returnMessage != nil {

		// Exiting
		return returnMessage, nil
	}

	// Trigger TestInstructionEngine to check if there are any TestInstructions on the ExecutionQueue
	channelCommandMessage := testInstructionExecutionEngine.ChannelCommandStruct{
		ChannelCommand: testInstructionExecutionEngine.ChannelCommandCheckNewTestInstructionExecutions,
	}

	// Send Message on Channel
	*fenixExecutionServerObject.executionEngineChannelRef <- channelCommandMessage

	return &fenixExecutionServerGrpcApi.AckNackResponse{AckNack: true, Comments: ""}, nil
}
