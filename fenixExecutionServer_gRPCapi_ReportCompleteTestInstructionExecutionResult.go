package main

import (
	"FenixExecutionServer/common_config"
	"FenixExecutionServer/testInstructionExecutionEngine"
	"context"
	fenixExecutionServerGrpcApi "github.com/jlambert68/FenixGrpcApi/FenixExecutionServer/fenixExecutionServerGrpcApi/go_grpc_api"
	"github.com/sirupsen/logrus"
	"time"
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

	// Reset Application shut time timer when there is an incoming gRPC-call
	var endApplicationWhenNoIncomingGrpcCallsSenderData endApplicationWhenNoIncomingGrpcCallsStruct
	endApplicationWhenNoIncomingGrpcCallsSenderData = endApplicationWhenNoIncomingGrpcCallsStruct{
		gRPCTimeStamp: time.Now(),
		senderName:    "gRPC - ReportCompleteTestInstructionExecutionResult",
	}
	endApplicationWhenNoIncomingGrpcCalls <- endApplicationWhenNoIncomingGrpcCallsSenderData

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
	executionTrackNumber = common_config.CalculateExecutionTrackNumber(
		finalTestInstructionExecutionResultMessage.TestInstructionExecutionUuid)

	// Check if the TestInstruction is kept in this ExecutionServer-instance

	// Create Response channel from TimeOutEngine to get answer if TestInstructionExecution is handled by this instance
	var timeOutResponseChannelForTimeOutHasOccurred common_config.TimeOutResponseChannelForTimeOutHasOccurredType
	timeOutResponseChannelForTimeOutHasOccurred = make(chan common_config.TimeOutResponseChannelForTimeOutHasOccurredStruct)

	// Create a message with TestInstructionExecution to be sent to TimeOutEngine for check if it is handled by this instance
	var tempTimeOutChannelTestInstructionExecutions common_config.TimeOutResponseChannelForVerifyIfTestInstructionIsHandledByThisInstanceStruct
	tempTimeOutChannelTestInstructionExecutions = common_config.TimeOutResponseChannelForVerifyIfTestInstructionIsHandledByThisInstanceStruct{
		TestInstructionExecutionUuid:    finalTestInstructionExecutionResultMessage.TestInstructionExecutionUuid,
		TestInstructionExecutionVersion: 1,
	}

	var tempTimeOutChannelCommand common_config.TimeOutChannelCommandStruct
	tempTimeOutChannelCommand = common_config.TimeOutChannelCommandStruct{
		TimeOutChannelCommand:                                                   0,
		TimeOutChannelTestInstructionExecutions:                                 common_config.TimeOutChannelCommandTestInstructionExecutionStruct{},
		TimeOutReturnChannelForTimeOutHasOccurred:                               nil,
		TimeOutResponseChannelForDurationUntilTimeOutOccurs:                     nil,
		TimeOutResponseChannelForVerifyIfTestInstructionIsHandledByThisInstance: common_config.TimeOutChannelCommandHasTestInstructionExecutionAlreadyTimedOut{},
		SendID: "",
	}
	tempTimeOutChannelCommands = common_config.TimeOutChannelCommandStruct{
		TimeOutChannelCommand:                     common_config.X,
		TimeOutChannelTestInstructionExecutions:   common_config.TimeOutChannelCommandTestInstructionExecutionStruct{},
		TimeOutReturnChannelForTimeOutHasOccurred: &timeOutResponseChannelForTimeOutHasOccurred,
		//TimeOutReturnChannelForExistsTestInstructionExecutionInTimeOutTimer: nil,
		SendID: "d5fe76fa-3d85-4f20-b7c8-79e24be5fac0",
	}

	// Send message on TimeOutEngineChannel to get information about if TestInstructionExecution already has TimedOut
	*common_config.TimeOutChannelEngineCommandChannelReferenceSlice[executionTrackNumber] <- tempTimeOutChannelCommand

	// Response from TimeOutEngine
	var timeOutReturnChannelForTimeOutHasOccurredValue common_config.TimeOutResponseChannelForTimeOutHasOccurredStruct

	// Wait for response from TimeOutEngine
	timeOutReturnChannelForTimeOutHasOccurredValue = <-timeOutResponseChannelForTimeOutHasOccurred

	// Verify that TestInstructionExecution hasn't TimedOut yet
	if timeOutReturnChannelForTimeOutHasOccurredValue.TimeOutWasTriggered == true {
		// TestInstructionExecution had already TimedOut
	}

	// Send Message to TestInstructionExecutionEngine via channel
	*fenixExecutionServerObject.executionEngine.CommandChannelReferenceSlice[executionTrackNumber] <- channelCommandMessage
	//*fenixExecutionServerObject.executionEngineChannelRefSlice[executionTrackNumber] <- channelCommandMessage

	// Create Return message
	returnMessage = &fenixExecutionServerGrpcApi.AckNackResponse{
		AckNack:    true,
		Comments:   "",
		ErrorCodes: nil,
		ProtoFileVersionUsedByClient: fenixExecutionServerGrpcApi.CurrentFenixExecutionServerProtoFileVersionEnum(
			common_config.GetHighestFenixExecutionServerProtoFileVersion()),
	}

	return returnMessage, nil
}
