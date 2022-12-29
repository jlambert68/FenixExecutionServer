package main

import (
	"FenixExecutionServer/common_config"
	"FenixExecutionServer/testInstructionExecutionEngine"
	"context"
	fenixExecutionServerGrpcApi "github.com/jlambert68/FenixGrpcApi/FenixExecutionServer/fenixExecutionServerGrpcApi/go_grpc_api"
	"github.com/sirupsen/logrus"
)

// ReportCompleteTestInstructionExecutionResult - *********************************************************************
// When a TestInstruction has been fully executed the Client use this to inform the results of the execution result to the Server
func (s *fenixExecutionServerGrpcServicesServer) ReportCompleteTestInstructionExecutionResult(ctx context.Context, finalTestInstructionExecutionResultMessage *fenixExecutionServerGrpcApi.FinalTestInstructionExecutionResultMessage) (*fenixExecutionServerGrpcApi.AckNackResponse, error) {

	fenixExecutionServerObject.logger.WithFields(logrus.Fields{
		"id": "299bac9a-bb4c-4dcd-9ca6-e486efc9e112",
		"finalTestInstructionExecutionResultMessage": finalTestInstructionExecutionResultMessage,
	}).Debug("Incoming 'gRPC - ReportCompleteTestInstructionExecutionResult'")

	defer fenixExecutionServerObject.logger.WithFields(logrus.Fields{
		"id": "61d0939d-bc96-46ea-9623-190cd2942d3e",
	}).Debug("Outgoing 'gRPC - ReportCompleteTestInstructionExecutionResult'")

	// Current user
	userID := finalTestInstructionExecutionResultMessage.ClientSystemIdentification.DomainUuid

	// Check if Client is using correct proto files version
	returnMessage := common_config.IsClientUsingCorrectTestDataProtoFileVersion(userID,
		fenixExecutionServerGrpcApi.CurrentFenixExecutionServerProtoFileVersionEnum(
			finalTestInstructionExecutionResultMessage.ClientSystemIdentification.ProtoFileVersionUsedByClient))
	if returnMessage != nil {

		// Exiting
		return returnMessage, nil
	}

	// returnMessage = fenixExecutionServerObject.prepareReportCompleteTestInstructionExecutionResultSaveToCloudDB(finalTestInstructionExecutionResultMessage)

	// Create Message to be sent to TestInstructionExecutionEngine
	channelCommandMessage := testInstructionExecutionEngine.ChannelCommandStruct{
		ChannelCommand: testInstructionExecutionEngine.ChannelCommandProcessFinalTestInstructionExecutionResultMessage,
		FinalTestInstructionExecutionResultMessage: finalTestInstructionExecutionResultMessage,
	}

	// Define Execution Track based on "lowest "TestCaseExecutionUuid
	var executionTrackNumber int
	executionTrackNumber = common_config.CalculateExecutionTrackNumber(finalTestInstructionExecutionResultMessage.TestInstructionExecutionUuid)

	// Send Message to TestInstructionExecutionEngine via channel
	*fenixExecutionServerObject.executionEngine.CommandChannelReferenceSlice[executionTrackNumber] <- channelCommandMessage
	//*fenixExecutionServerObject.executionEngineChannelRefSlice[executionTrackNumber] <- channelCommandMessage

	// Create Return message
	returnMessage = &fenixExecutionServerGrpcApi.AckNackResponse{
		AckNack:                      true,
		Comments:                     "",
		ErrorCodes:                   nil,
		ProtoFileVersionUsedByClient: fenixExecutionServerGrpcApi.CurrentFenixExecutionServerProtoFileVersionEnum(common_config.GetHighestFenixExecutionServerProtoFileVersion()),
	}

	return returnMessage, nil
}
