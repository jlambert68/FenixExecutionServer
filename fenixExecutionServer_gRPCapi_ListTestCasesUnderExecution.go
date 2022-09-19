package main

import (
	"context"
	fenixExecutionServerGuiGrpcApi "github.com/jlambert68/FenixGrpcApi/FenixExecutionServer/fenixExecutionServerGuiGrpcApi/go_grpc_api"
	"github.com/sirupsen/logrus"
)

// ListAllImmatureTestInstructionAttributes - *********************************************************************
// The TestCase Builder asks for all TestInstructions Attributes that the user must add values to in TestCase
func (s *fenixExecutionServerGuiGrpcServicesServer) ListTestCasesUnderExecution(ctx context.Context, listTestCasesUnderExecutionRequest *fenixExecutionServerGuiGrpcApi.ListTestCasesUnderExecutionRequest) (*fenixExecutionServerGuiGrpcApi.ListTestCasesUnderExecutionResponse, error) {

	// Define the response message
	var responseMessage *fenixExecutionServerGuiGrpcApi.ListTestCasesUnderExecutionResponse

	fenixExecutionServerObject.logger.WithFields(logrus.Fields{
		"id": "30e50aed-e860-467f-abdf-673d37788616",
	}).Debug("Incoming 'gRPC - ListTestCasesUnderExecution'")

	defer fenixExecutionServerObject.logger.WithFields(logrus.Fields{
		"id": "e316b0ac-edd9-48f7-a550-dcf1a30aeab8",
	}).Debug("Outgoing 'gRPC - ListTestCasesUnderExecution'")

	// Current user
	userID := listTestCasesUnderExecutionRequest.UserIdentification.UserId

	// Check if Client is using correct proto files version
	returnMessage := fenixExecutionServerObject.isClientUsingCorrectTestDataProtoFileVersion(userID, fenixExecutionServerGuiGrpcApi.CurrentFenixExecutionGuiProtoFileVersionEnum(listTestCasesUnderExecutionRequest.UserIdentification.ProtoFileVersionUsedByClient))
	if returnMessage != nil {

		responseMessage = &fenixExecutionServerGuiGrpcApi.ListTestCasesUnderExecutionResponse{
			AckNackResponse: returnMessage,
		}

		// Exiting
		return responseMessage, nil
	}

	// Define variables to store data from DB in
	var testCaseUnderExecutionMessage []*fenixExecutionServerGuiGrpcApi.TestCaseUnderExecutionMessage

	// Get users ImmatureTestInstruction-data from CloudDB
	testCaseUnderExecutionMessage, err := fenixExecutionServerObject.listTestCasesUnderExecutionLoadFromCloudDB(userID)
	if err != nil {
		// Something went wrong so return an error to caller
		responseMessage = &fenixExecutionServerGuiGrpcApi.ListTestCasesUnderExecutionResponse{
			AckNackResponse: &fenixExecutionServerGuiGrpcApi.AckNackResponse{
				AckNack:                      false,
				Comments:                     "Got some Error when retrieving TestCaseUnderExecutionMessage from database",
				ErrorCodes:                   []fenixExecutionServerGuiGrpcApi.ErrorCodesEnum{fenixExecutionServerGuiGrpcApi.ErrorCodesEnum_ERROR_DATABASE_PROBLEM},
				ProtoFileVersionUsedByClient: fenixExecutionServerGuiGrpcApi.CurrentFenixExecutionGuiProtoFileVersionEnum(fenixExecutionServerObject.getHighestFenixTestDataProtoFileVersion()),
			},
			TestCasesUnderExecution: nil,
		}

		// Exiting
		return responseMessage, nil
	}

	// Create the response to caller
	responseMessage = &fenixExecutionServerGuiGrpcApi.ListTestCasesUnderExecutionResponse{
		AckNackResponse: &fenixExecutionServerGuiGrpcApi.AckNackResponse{
			AckNack:                      true,
			Comments:                     "",
			ErrorCodes:                   nil,
			ProtoFileVersionUsedByClient: fenixExecutionServerGuiGrpcApi.CurrentFenixExecutionGuiProtoFileVersionEnum(fenixExecutionServerObject.getHighestFenixTestDataProtoFileVersion()),
		},
		TestCasesUnderExecution: testCaseUnderExecutionMessage,
	}

	return responseMessage, nil
}
