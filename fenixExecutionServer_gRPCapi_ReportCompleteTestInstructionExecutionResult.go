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
func (s *fenixExecutionServerGrpcServicesServer) ReportCompleteTestInstructionExecutionResult(
	ctx context.Context,
	finalTestInstructionExecutionResultMessage *fenixExecutionServerGrpcApi.FinalTestInstructionExecutionResultMessage) (
	*fenixExecutionServerGrpcApi.AckNackResponse, error) {

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

	// Define Execution Track based on "lowest "TestCaseExecutionUuid
	var executionTrackNumber int
	executionTrackNumber = common_config.CalculateExecutionTrackNumber(
		finalTestInstructionExecutionResultMessage.TestInstructionExecutionUuid)

	// *** Check if the TestInstruction is kept in this ExecutionServer-instance ***

	// Create Response channel from TimeOutEngine to get answer if TestInstructionExecution is handled by this instance
	var timeOutResponseChannelForIsThisHandledByThisExecutionInstance common_config.TimeOutResponseChannelForVerifyIfTestInstructionIsHandledByThisInstanceType
	timeOutResponseChannelForIsThisHandledByThisExecutionInstance = make(chan common_config.TimeOutResponseChannelForVerifyIfTestInstructionIsHandledByThisInstanceStruct)

	var tempTimeOutChannelTestInstructionExecutions common_config.TimeOutChannelCommandTestInstructionExecutionStruct
	tempTimeOutChannelTestInstructionExecutions = common_config.TimeOutChannelCommandTestInstructionExecutionStruct{
		TestCaseExecutionUuid:                   "",
		TestCaseExecutionVersion:                0,
		TestInstructionExecutionUuid:            finalTestInstructionExecutionResultMessage.TestInstructionExecutionUuid,
		TestInstructionExecutionVersion:         1,
		TestInstructionExecutionCanBeReExecuted: false,
		TimeOutTime:                             time.Time{},
	}

	var tempTimeOutChannelCommand common_config.TimeOutChannelCommandStruct
	tempTimeOutChannelCommand = common_config.TimeOutChannelCommandStruct{
		TimeOutChannelCommand:                                                   common_config.TimeOutChannelCommandVerifyIfTestInstructionIsHandledByThisExecutionInstance,
		TimeOutChannelTestInstructionExecutions:                                 tempTimeOutChannelTestInstructionExecutions,
		TimeOutReturnChannelForTimeOutHasOccurred:                               nil,
		TimeOutResponseChannelForDurationUntilTimeOutOccurs:                     nil,
		TimeOutResponseChannelForVerifyIfTestInstructionIsHandledByThisInstance: &timeOutResponseChannelForIsThisHandledByThisExecutionInstance,
		SendID:                         "4a49cb94-3afc-42aa-9b72-b57010559c2c",
		MessageInitiatedFromPubSubSend: false,
	}

	// Send message on TimeOutEngineChannel to get information about if TestInstructionExecution already has TimedOut
	*common_config.TimeOutChannelEngineCommandChannelReferenceSlice[executionTrackNumber] <- tempTimeOutChannelCommand

	// Response from TimeOutEngine
	var timeOutResponseChannelForVerifyIfTestInstructionIsHandledByThisInstanceValue common_config.TimeOutResponseChannelForVerifyIfTestInstructionIsHandledByThisInstanceStruct

	// Wait for response from TimeOutEngine
	timeOutResponseChannelForVerifyIfTestInstructionIsHandledByThisInstanceValue = <-timeOutResponseChannelForIsThisHandledByThisExecutionInstance

	// Verify that TestInstructionExecution is handled by this Execution-instance
	if timeOutResponseChannelForVerifyIfTestInstructionIsHandledByThisInstanceValue.TestInstructionIsHandledByThisExecutionInstance == true {
		// *** TestInstructionExecution is handled by this Execution-instance ***

		// Create Message to be sent to TestInstructionExecutionEngine
		channelCommandMessage := testInstructionExecutionEngine.ChannelCommandStruct{
			ChannelCommand: testInstructionExecutionEngine.ChannelCommandProcessFinalTestInstructionExecutionResultMessage,
			FinalTestInstructionExecutionResultMessage: finalTestInstructionExecutionResultMessage,
		}

		// Send Message to TestInstructionExecutionEngine via channel
		*fenixExecutionServerObject.executionEngine.CommandChannelReferenceSlice[executionTrackNumber] <- channelCommandMessage

	} else {
		// TestInstructionExecution is NOT handled by this Execution-instance
		common_config.Logger.WithFields(logrus.Fields{
			"id": "dfe9b1f8-05c3-4553-b6d3-134d782c8a96",
			"finalTestInstructionExecutionResultMessage.TestInstructionExecutionUuid": finalTestInstructionExecutionResultMessage.TestInstructionExecutionUuid,
		}).Info("TestInstructionExecutionUuid is not handled by this Execution-instance")

		// Create Message to be sent to TestInstructionExecutionEngine
		channelCommandMessage := testInstructionExecutionEngine.ChannelCommandStruct{
			ChannelCommand: testInstructionExecutionEngine.ChannelCommandFinalTestInstructionExecutionResultIsNotHandledByThisExecutionInstance,
			FinalTestInstructionExecutionResultMessage: finalTestInstructionExecutionResultMessage,
		}

		// Send Message to TestInstructionExecutionEngine via channel
		*fenixExecutionServerObject.executionEngine.CommandChannelReferenceSlice[executionTrackNumber] <- channelCommandMessage

	}

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
