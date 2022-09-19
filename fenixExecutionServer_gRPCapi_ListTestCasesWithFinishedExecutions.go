package main

import (
	"context"
	fenixExecutionServerGuiGrpcApi "github.com/jlambert68/FenixGrpcApi/FenixExecutionServer/fenixExecutionServerGuiGrpcApi/go_grpc_api"
	"github.com/sirupsen/logrus"
)

// ListAllImmatureTestInstructionAttributes - *********************************************************************
// The TestCase Builder asks for all TestInstructions Attributes that the user must add values to in TestCase
func (s *fenixExecutionServerGuiGrpcServicesServer) ListTestCasesWithFinishedExecutions(ctx context.Context, listTestCasesWithFinishedExecutionsRequest *fenixExecutionServerGuiGrpcApi.ListTestCasesWithFinishedExecutionsRequest) (*fenixExecutionServerGuiGrpcApi.ListTestCasesWithFinishedExecutionsResponse, error) {

	// Define the response message
	var responseMessage *fenixExecutionServerGuiGrpcApi.ListTestCasesWithFinishedExecutionsResponse

	fenixExecutionServerObject.logger.WithFields(logrus.Fields{
		"id": "33451c5f-1230-4ea1-817e-08afa1c1192b",
	}).Debug("Incoming 'gRPC - ListTestCasesWithFinishedExecutions'")

	defer fenixExecutionServerObject.logger.WithFields(logrus.Fields{
		"id": "1cabcbc9-86ab-4ffb-b41f-8562c3bc1d75",
	}).Debug("Outgoing 'gRPC - ListTestCasesWithFinishedExecutions'")

	// Current user
	userID := listTestCasesWithFinishedExecutionsRequest.UserIdentification.UserId

	// Check if Client is using correct proto files version
	returnMessage := fenixExecutionServerObject.isClientUsingCorrectTestDataProtoFileVersion(userID, fenixExecutionServerGuiGrpcApi.CurrentFenixExecutionGuiProtoFileVersionEnum(listTestCasesWithFinishedExecutionsRequest.UserIdentification.ProtoFileVersionUsedByClient))
	if returnMessage != nil {

		responseMessage = &fenixExecutionServerGuiGrpcApi.ListTestCasesWithFinishedExecutionsResponse{
			AckNackResponse:               returnMessage,
			TestCaseWithFinishedExecution: nil,
		}

		// Exiting
		return responseMessage, nil
	}

	// Define variables to store data from DB in
	var listTestCasesWithFinishedExecutionsResponse []*fenixExecutionServerGuiGrpcApi.TestCaseWithFinishedExecutionMessage

	// Get users ImmatureTestInstruction-data from CloudDB
	listTestCasesWithFinishedExecutionsResponse, err := fenixExecutionServerObject.listTestCasesWithFinishedExecutionsLoadFromCloudDB(userID)
	if err != nil {
		// Something went wrong so return an error to caller
		responseMessage = &fenixExecutionServerGuiGrpcApi.ListTestCasesWithFinishedExecutionsResponse{
			AckNackResponse: &fenixExecutionServerGuiGrpcApi.AckNackResponse{
				AckNack:                      false,
				Comments:                     "Got some Error when retrieving ListTestCasesWithFinishedExecutions from database",
				ErrorCodes:                   []fenixExecutionServerGuiGrpcApi.ErrorCodesEnum{fenixExecutionServerGuiGrpcApi.ErrorCodesEnum_ERROR_DATABASE_PROBLEM},
				ProtoFileVersionUsedByClient: fenixExecutionServerGuiGrpcApi.CurrentFenixExecutionGuiProtoFileVersionEnum(fenixExecutionServerObject.getHighestFenixTestDataProtoFileVersion()),
			},
			TestCaseWithFinishedExecution: nil,
		}

		// Exiting
		return responseMessage, nil
	}

	// Create the response to caller
	responseMessage = &fenixExecutionServerGuiGrpcApi.ListTestCasesWithFinishedExecutionsResponse{
		AckNackResponse: &fenixExecutionServerGuiGrpcApi.AckNackResponse{
			AckNack:                      true,
			Comments:                     "",
			ErrorCodes:                   nil,
			ProtoFileVersionUsedByClient: fenixExecutionServerGuiGrpcApi.CurrentFenixExecutionGuiProtoFileVersionEnum(fenixExecutionServerObject.getHighestFenixTestDataProtoFileVersion()),
		},
		TestCaseWithFinishedExecution: listTestCasesWithFinishedExecutionsResponse,
	}

	return responseMessage, nil
}
