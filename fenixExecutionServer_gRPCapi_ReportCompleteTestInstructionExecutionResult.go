package main

import (
	"FenixExecutionServer/common_config"
	"context"
	fenixExecutionServerGrpcApi "github.com/jlambert68/FenixGrpcApi/FenixExecutionServer/fenixExecutionServerGrpcApi/go_grpc_api"
	"github.com/sirupsen/logrus"
)

// ReportCompleteTestInstructionExecutionResult - *********************************************************************
// When a TestInstruction has been fully executed the Client use this to inform the results of the execution result to the Server
func (s *fenixExecutionServerGrpcServicesServer) ReportCompleteTestInstructionExecutionResult(ctx context.Context, finalTestInstructionExecutionResultMessage *fenixExecutionServerGrpcApi.FinalTestInstructionExecutionResultMessage) (*fenixExecutionServerGrpcApi.AckNackResponse, error) {

	fenixExecutionServerObject.logger.WithFields(logrus.Fields{
		"id": "299bac9a-bb4c-4dcd-9ca6-e486efc9e112",
	}).Debug("Incoming 'gRPC - ReportCompleteTestInstructionExecutionResult'")

	defer fenixExecutionServerObject.logger.WithFields(logrus.Fields{
		"id": "61d0939d-bc96-46ea-9623-190cd2942d3e",
	}).Debug("Outgoing 'gRPC - ReportCompleteTestInstructionExecutionResult'")

	// Current user
	userID := finalTestInstructionExecutionResultMessage.ClientSystemIdentification.DomainUuid

	// Check if Client is using correct proto files version
	returnMessage := common_config.IsClientUsingCorrectTestDataProtoFileVersion(userID, fenixExecutionServerGrpcApi.CurrentFenixExecutionServerProtoFileVersionEnum(finalTestInstructionExecutionResultMessage.ClientSystemIdentification.ProtoFileVersionUsedByClient))
	if returnMessage != nil {

		// Exiting
		return returnMessage, nil
	}

	return &fenixExecutionServerGrpcApi.AckNackResponse{AckNack: true, Comments: ""}, nil
}
