package main

import (
	"FenixExecutionServer/common_config"
	fenixExecutionServerGrpcApi "github.com/jlambert68/FenixGrpcApi/FenixExecutionServer/fenixExecutionServerGrpcApi/go_grpc_api"
	"github.com/sirupsen/logrus"
	"io"
	"time"
)

// ReportCurrentTestInstructionExecutionResult - *********************************************************************
// During a TestInstruction execution the Client use this to inform the current of the execution result to the Server
func (s *fenixExecutionServerGrpcServicesServer) ReportCurrentTestInstructionExecutionResult(currentTestInstructionExecutionResultMessageStream fenixExecutionServerGrpcApi.FenixExecutionServerGrpcServices_ReportCurrentTestInstructionExecutionResultServer) error {

	fenixExecutionServerObject.logger.WithFields(logrus.Fields{
		"id": "8a01617a-6cfc-4684-98de-52631edfd2c4",
	}).Debug("Incoming 'gRPC - ReportCurrentTestInstructionExecutionResult'")

	defer fenixExecutionServerObject.logger.WithFields(logrus.Fields{
		"id": "2a5d06a5-730d-447e-8a95-dbe4298122ce",
	}).Debug("Outgoing 'gRPC - ReportCurrentTestInstructionExecutionResult'")

	// Reset Application shut time timer when there is an incoming gRPC-call
	var endApplicationWhenNoIncomingGrpcCallsSenderData endApplicationWhenNoIncomingGrpcCallsStruct
	endApplicationWhenNoIncomingGrpcCallsSenderData = endApplicationWhenNoIncomingGrpcCallsStruct{
		gRPCTimeStamp: time.Now(),
		senderName:    "gRPC - ReportCurrentTestInstructionExecutionResult",
	}
	endApplicationWhenNoIncomingGrpcCalls <- endApplicationWhenNoIncomingGrpcCallsSenderData

	// Container to store all messages before process them
	var currentTestInstructionExecutionResultMessages []*fenixExecutionServerGrpcApi.CurrentTestInstructionExecutionResultMessage
	var currentTestInstructionExecutionResultMessage *fenixExecutionServerGrpcApi.CurrentTestInstructionExecutionResultMessage

	var err error
	var returnMessage *fenixExecutionServerGrpcApi.AckNackResponse
	var firstRowVerification bool = true

	// Retrieve stream from Client
	for {
		// Receive message and add it to 'currentTestInstructionExecutionResultMessages'
		currentTestInstructionExecutionResultMessage, err = currentTestInstructionExecutionResultMessageStream.Recv()

		// Only check Protofile-version on first post
		if firstRowVerification == true {

			// Current user
			userID := currentTestInstructionExecutionResultMessage.ClientSystemIdentification.DomainUuid

			// Check if Client is using correct proto files version
			returnMessage := common_config.IsClientUsingCorrectTestDataProtoFileVersion(userID, fenixExecutionServerGrpcApi.CurrentFenixExecutionServerProtoFileVersionEnum(currentTestInstructionExecutionResultMessage.ClientSystemIdentification.ProtoFileVersionUsedByClient))
			if returnMessage != nil {

				// Exiting
				return currentTestInstructionExecutionResultMessageStream.SendAndClose(returnMessage)
			}
		}

		currentTestInstructionExecutionResultMessages = append(currentTestInstructionExecutionResultMessages, currentTestInstructionExecutionResultMessage)

		// When no more messages is received then continue
		if err == io.EOF {
			break
		}
	}

	common_config.Logger.WithFields(logrus.Fields{
		"id": "4f92676d-89d8-458a-a86b-6af841d523a6",
		"currentTestInstructionExecutionResultMessages": currentTestInstructionExecutionResultMessages,
	}).Debug("All received messages that was added to 'currentTestInstructionExecutionResultMessages'")

	returnMessage = &fenixExecutionServerGrpcApi.AckNackResponse{
		AckNack:  true,
		Comments: ""}

	return currentTestInstructionExecutionResultMessageStream.SendAndClose(returnMessage)

}
