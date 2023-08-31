package main

import (
	"FenixExecutionServer/common_config"
	"FenixExecutionServer/testInstructionExecutionEngine"
	"context"
	fenixExecutionServerGrpcApi "github.com/jlambert68/FenixGrpcApi/FenixExecutionServer/fenixExecutionServerGrpcApi/go_grpc_api"
	"github.com/sirupsen/logrus"
	"time"
)

// ProcessResponseTestInstructionExecution - *********************************************************************
// TestInstructionExecution was received by connector and this response tells if the Connector can execution the TestInstruction or not
func (s *fenixExecutionServerGrpcServicesServer) ProcessResponseTestInstructionExecution(
	ctx context.Context,
	processTestInstructionExecutionResponseStatus *fenixExecutionServerGrpcApi.ProcessTestInstructionExecutionResponseStatus) (
	*fenixExecutionServerGrpcApi.AckNackResponse,
	error) {

	fenixExecutionServerObject.logger.WithFields(logrus.Fields{
		"id": "360b280b-ff75-410d-8a46-5a6311e4d047",
		"processTestInstructionExecutionResponseStatus": processTestInstructionExecutionResponseStatus,
	}).Debug("Incoming 'gRPC - ProcessResponseTestInstructionExecution'")

	defer fenixExecutionServerObject.logger.WithFields(logrus.Fields{
		"id": "1bf45bea-0642-4ca3-978f-4cf9c0ca26c3",
	}).Debug("Outgoing 'gRPC - ProcessResponseTestInstructionExecution'")

	// Reset Application shut time timer when there is an incoming gRPC-call
	var endApplicationWhenNoIncomingGrpcCallsSenderData endApplicationWhenNoIncomingGrpcCallsStruct
	endApplicationWhenNoIncomingGrpcCallsSenderData = endApplicationWhenNoIncomingGrpcCallsStruct{
		gRPCTimeStamp: time.Now(),
		senderName:    "gRPC - ReportCompleteTestInstructionExecutionResult",
	}
	endApplicationWhenNoIncomingGrpcCalls <- endApplicationWhenNoIncomingGrpcCallsSenderData

	// Current user
	userID := "Unknown Worker"

	// Check if Client is using correct proto files version
	returnMessage := common_config.IsClientUsingCorrectTestDataProtoFileVersion(userID,
		fenixExecutionServerGrpcApi.CurrentFenixExecutionServerProtoFileVersionEnum(
			processTestInstructionExecutionResponseStatus.AckNackResponse.GetProtoFileVersionUsedByClient()))
	if returnMessage != nil {

		// Exiting
		return returnMessage, nil
	}

	// returnMessage = fenixExecutionServerObject.prepareReportCompleteTestInstructionExecutionResultSaveToCloudDB(processTestInstructionExecutionResponseStatus)

	// Define Execution Track based on "lowest "TestCaseExecutionUuid
	var executionTrackNumber int
	executionTrackNumber = common_config.CalculateExecutionTrackNumber(
		processTestInstructionExecutionResponseStatus.TestInstructionExecutionUuid)

	// *** Check if the TestInstruction is kept in this ExecutionServer-instance ***

	// Create Response channel from TimeOutEngine to get answer if TestInstructionExecution is handled by this instance
	var timeOutResponseChannelForIsThisHandledByThisExecutionInstance common_config.
		TimeOutResponseChannelForVerifyIfTestInstructionIsHandledByThisInstanceType
	timeOutResponseChannelForIsThisHandledByThisExecutionInstance = make(chan common_config.
		TimeOutResponseChannelForVerifyIfTestInstructionIsHandledByThisInstanceStruct)

	var tempTimeOutChannelTestInstructionExecutions common_config.TimeOutChannelCommandTestInstructionExecutionStruct
	tempTimeOutChannelTestInstructionExecutions = common_config.TimeOutChannelCommandTestInstructionExecutionStruct{
		TestCaseExecutionUuid:                   "",
		TestCaseExecutionVersion:                0,
		TestInstructionExecutionUuid:            processTestInstructionExecutionResponseStatus.TestInstructionExecutionUuid,
		TestInstructionExecutionVersion:         1,
		TestInstructionExecutionCanBeReExecuted: false,
		TimeOutTime:                             time.Time{},
	}

	var tempTimeOutChannelCommand common_config.TimeOutChannelCommandStruct
	tempTimeOutChannelCommand = common_config.TimeOutChannelCommandStruct{
		TimeOutChannelCommand: common_config.
			TimeOutChannelCommandVerifyIfTestInstructionIsHandledByThisExecutionInstance,
		TimeOutChannelTestInstructionExecutions:                                 tempTimeOutChannelTestInstructionExecutions,
		TimeOutReturnChannelForTimeOutHasOccurred:                               nil,
		TimeOutResponseChannelForDurationUntilTimeOutOccurs:                     nil,
		TimeOutResponseChannelForVerifyIfTestInstructionIsHandledByThisInstance: &timeOutResponseChannelForIsThisHandledByThisExecutionInstance,
		SendID: "30303c99-11ca-494d-a082-9f0e46bc3364",
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
			ChannelCommand: testInstructionExecutionEngine.ChannelCommandProcessTestInstructionExecutionResponseStatus,
			ProcessTestInstructionExecutionResponseStatus: processTestInstructionExecutionResponseStatus,
		}

		// Send Message to TestInstructionExecutionEngine via channel
		*fenixExecutionServerObject.executionEngine.CommandChannelReferenceSlice[executionTrackNumber] <- channelCommandMessage

	} else {
		// TestInstructionExecution is NOT handled by this Execution-instance
		common_config.Logger.WithFields(logrus.Fields{
			"id": "24873427-747c-49b9-8c42-0bc745651abb",
			"processTestInstructionExecutionResponseStatus.TestInstructionExecutionUuid": processTestInstructionExecutionResponseStatus.TestInstructionExecutionUuid,
		}).Info("TestInstructionExecutionUuid is not handled by this Execution-instance")

		// Create Message to be sent to TestInstructionExecutionEngine
		channelCommandMessage := testInstructionExecutionEngine.ChannelCommandStruct{
			ChannelCommand: testInstructionExecutionEngine.ChannelCommandProcessTestInstructionExecutionResponseStatusIsNotHandledByThisExecutionInstance,
			ProcessTestInstructionExecutionResponseStatus: processTestInstructionExecutionResponseStatus,
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
