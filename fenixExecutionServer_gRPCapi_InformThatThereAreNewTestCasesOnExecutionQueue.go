package main

import (
	"FenixExecutionServer/common_config"
	"context"
	fenixExecutionServerGrpcApi "github.com/jlambert68/FenixGrpcApi/FenixExecutionServer/fenixExecutionServerGrpcApi/go_grpc_api"
	"github.com/sirupsen/logrus"
)

// InformThatThereAreNewTestCasesOnExecutionQueue - *********************************************************************
// ExecutionServerGui-server inform ExecutionServer that there is a new TestCase that is ready on the Execution-queue
func (s *fenixExecutionServerGrpcServicesServer) InformThatThereAreNewTestCasesOnExecutionQueue(ctx context.Context, emptyParameter *fenixExecutionServerGrpcApi.EmptyParameter) (*fenixExecutionServerGrpcApi.AckNackResponse, error) {

	fenixExecutionServerObject.logger.WithFields(logrus.Fields{
		"id": "862cb663-daea-4f33-9f6e-03594d3005df",
	}).Debug("Incoming 'gRPC - InformThatThereAreNewTestCasesOnExecutionQueue'")

	defer fenixExecutionServerObject.logger.WithFields(logrus.Fields{
		"id": "6507f7a9-4def-4a38-90c7-7bb19311f10f",
	}).Debug("Outgoing 'gRPC - InformThatThereAreNewTestCasesOnExecutionQueue'")

	// Current user
	userID := "gRPC-api doesn't support UserId"

	// Check if Client is using correct proto files version
	returnMessage := common_config.IsClientUsingCorrectTestDataProtoFileVersion(userID, fenixExecutionServerGrpcApi.CurrentFenixExecutionServerProtoFileVersionEnum(emptyParameter.ProtoFileVersionUsedByClient))
	if returnMessage != nil {

		// Exiting
		return returnMessage, nil
	}

	// Create TestInstructions to be saved on 'TestInstructionExecutionQueue'
	returnMessage = fenixExecutionServerObject.prepareInformThatThereAreNewTestCasesOnExecutionQueueSaveToCloudDB(emptyParameter)
	if returnMessage != nil {
		return returnMessage, nil
	}

	return &fenixExecutionServerGrpcApi.AckNackResponse{AckNack: true, Comments: ""}, nil
}
