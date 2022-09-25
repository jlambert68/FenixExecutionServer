package main

import (
	"context"
	fenixExecutionServerGrpcApi "github.com/jlambert68/FenixGrpcApi/FenixExecutionServer/fenixExecutionServerGrpcApi/go_grpc_api"
	"github.com/sirupsen/logrus"
)

// InformThatThereAreNewTestCasesOnExecutionQueue - *********************************************************************
// ExecutionServerGui-server inform ExecutionServer that there is a new TestCase that is ready on the Execution-queue
func (s *fenixExecutionServerGrpcServicesServer) InformThatThereAreNewTestInstructionsOnExecutionQueue(ctx context.Context, emptyParameter *fenixExecutionServerGrpcApi.EmptyParameter) (*fenixExecutionServerGrpcApi.AckNackResponse, error) {

	fenixExecutionServerObject.logger.WithFields(logrus.Fields{
		"id": "7ceb7c60-e90e-40ea-92c7-7cc5becb0d98",
	}).Debug("Incoming 'gRPC - InformThatThereAreNewTestInstructionsOnExecutionQueue'")

	defer fenixExecutionServerObject.logger.WithFields(logrus.Fields{
		"id": "38224ef0-060d-4b64-b4ff-f1f68939b53b",
	}).Debug("Outgoing 'gRPC - InformThatThereAreNewTestInstructionsOnExecutionQueue'")

	// Current user
	userID := "gRPC-api doesn't support UserId"

	// Check if Client is using correct proto files version
	returnMessage := fenixExecutionServerObject.isClientUsingCorrectTestDataProtoFileVersion(userID, fenixExecutionServerGrpcApi.CurrentFenixExecutionServerProtoFileVersionEnum(emptyParameter.ProtoFileVersionUsedByClient))
	if returnMessage != nil {

		// Exiting
		return returnMessage, nil
	}

	/*
		// Create TestInstructions to be saved on 'TestInstructionExecutionQueue'
		returnMessage = fenixExecutionServerObject.prepareInformThatThereAreNewTestCasesOnExecutionQueueSaveToCloudDB(emptyParameter)
		if returnMessage != nil {
			return returnMessage, nil
		}
	*/
	return &fenixExecutionServerGrpcApi.AckNackResponse{AckNack: true, Comments: ""}, nil
}
