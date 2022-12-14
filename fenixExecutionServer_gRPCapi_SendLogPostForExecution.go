package main

import (
	"FenixExecutionServer/common_config"
	fenixExecutionServerGrpcApi "github.com/jlambert68/FenixGrpcApi/FenixExecutionServer/fenixExecutionServerGrpcApi/go_grpc_api"
	"github.com/sirupsen/logrus"
	"io"
	"time"
)

// SendLogPostForExecution - *********************************************************************
// During the execution the Client can send log information that can be shown to the user
func (s *fenixExecutionServerGrpcServicesServer) SendLogPostForExecution(logPostsMessageStream fenixExecutionServerGrpcApi.FenixExecutionServerGrpcServices_SendLogPostForExecutionServer) error {

	fenixExecutionServerObject.logger.WithFields(logrus.Fields{
		"id": "ce04f087-1721-44f9-8534-e4c8ae87f18d",
	}).Debug("Incoming 'gRPC - SendLogPostForExecution'")

	defer fenixExecutionServerObject.logger.WithFields(logrus.Fields{
		"id": "3cabce48-2b4a-4e02-a8fa-00be9d108bbd",
	}).Debug("Outgoing 'gRPC - SendLogPostForExecution'")

	// Reset Application shut time timer when there is an incoming gRPC-call
	var endApplicationWhenNoIncomingGrpcCallsSenderData endApplicationWhenNoIncomingGrpcCallsStruct
	endApplicationWhenNoIncomingGrpcCallsSenderData = endApplicationWhenNoIncomingGrpcCallsStruct{
		gRPCTimeStamp: time.Now(),
		senderName:    "gRPC - SendLogPostForExecution",
	}
	endApplicationWhenNoIncomingGrpcCalls <- endApplicationWhenNoIncomingGrpcCallsSenderData

	// Container to store all messages before process them
	var logPostsMessages []*fenixExecutionServerGrpcApi.LogPostsMessage
	var logPostsMessage *fenixExecutionServerGrpcApi.LogPostsMessage

	var err error
	var returnMessage *fenixExecutionServerGrpcApi.AckNackResponse
	var firstRowVerification bool = true

	// Retrieve stream from Client
	for {
		// Receive message and add it to 'currentTestInstructionExecutionResultMessages'
		logPostsMessage, err = logPostsMessageStream.Recv()

		// Only check Protofile-version on first post
		if firstRowVerification == true {

			// Current user
			userID := logPostsMessage.ClientSystemIdentification.DomainUuid

			// Check if Client is using correct proto files version
			returnMessage := common_config.IsClientUsingCorrectTestDataProtoFileVersion(userID, fenixExecutionServerGrpcApi.CurrentFenixExecutionServerProtoFileVersionEnum(logPostsMessage.ClientSystemIdentification.ProtoFileVersionUsedByClient))
			if returnMessage != nil {

				// Exiting
				return logPostsMessageStream.SendAndClose(returnMessage)
			}
		}

		logPostsMessages = append(logPostsMessages, logPostsMessage)

		// When no more messages is received then continue
		if err == io.EOF {
			break
		}
	}

	common_config.Logger.WithFields(logrus.Fields{
		"id":               "00927bf8-a36e-4cd6-9c4d-f8edcaf620e8",
		"logPostsMessages": logPostsMessages,
	}).Debug("logPostsMessages that were received")

	returnMessage = &fenixExecutionServerGrpcApi.AckNackResponse{
		AckNack:  true,
		Comments: ""}

	return logPostsMessageStream.SendAndClose(returnMessage)

}
