package main

import (
	"fmt"
	fenixExecutionServerGrpcApi "github.com/jlambert68/FenixGrpcApi/FenixExecutionServer/fenixExecutionServerGrpcApi/go_grpc_api"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	"io"
)

// AreYouAlive - *********************************************************************
//Anyone can check if Fenix TestCase Builder server is alive with this service
func (s *fenixExecutionServerGrpcServicesServer) AreYouAlive(ctx context.Context, emptyParameter *fenixExecutionServerGrpcApi.EmptyParameter) (*fenixExecutionServerGrpcApi.AckNackResponse, error) {

	fenixExecutionServerObject.logger.WithFields(logrus.Fields{
		"id": "1ff67695-9a8b-4821-811d-0ab8d33c4d8b",
	}).Debug("Incoming 'gRPC - AreYouAlive'")

	defer fenixExecutionServerObject.logger.WithFields(logrus.Fields{
		"id": "9c7f0c3d-7e9f-4c91-934e-8d7a22926d84",
	}).Debug("Outgoing 'gRPC - AreYouAlive'")

	return &fenixExecutionServerGrpcApi.AckNackResponse{AckNack: true, Comments: "I'am alive."}, nil
}

// InformThatThereAreNewTestCasesOnExecutionQueue - *********************************************************************
// ExecutionServerGui-server inform ExecutionServer that there is a new TestCase that is ready on the Execution-queue
func (s *fenixExecutionServerGrpcServicesServer) InformThatThereAreNewTestCasesOnExecutionQueue(ctx context.Context, emptyParameter *fenixExecutionServerGrpcApi.EmptyParameter) (*fenixExecutionServerGrpcApi.AckNackResponse, error) {

	fenixExecutionServerObject.logger.WithFields(logrus.Fields{
		"id": "862cb663-daea-4f33-9f6e-03594d3005df",
	}).Debug("Incoming 'gRPC - InformThatThereAreNewTestCasesOnExecutionQueue'")

	defer fenixExecutionServerObject.logger.WithFields(logrus.Fields{
		"id": "6507f7a9-4def-4a38-90c7-7bb19311f10f",
	}).Debug("Outgoing 'gRPC - InformThatThereAreNewTestCasesOnExecutionQueue'")

	return &fenixExecutionServerGrpcApi.AckNackResponse{AckNack: true, Comments: ""}, nil
}

// ReportProcessingCapability - *********************************************************************
// Client can inform Server of Client capability to execute requests in parallell, serial or no processing at all
func (s *fenixExecutionServerGrpcServicesServer) ReportProcessingCapability(ctx context.Context, processingCapabilityMessage *fenixExecutionServerGrpcApi.ProcessingCapabilityMessage) (*fenixExecutionServerGrpcApi.AckNackResponse, error) {

	fenixExecutionServerObject.logger.WithFields(logrus.Fields{
		"id": "fd6f4e08-7708-454c-932c-231269628031",
	}).Debug("Incoming 'gRPC - ReportProcessingCapability'")

	defer fenixExecutionServerObject.logger.WithFields(logrus.Fields{
		"id": "6b75f162-17c5-4c6c-a31c-502a9e76a826",
	}).Debug("Outgoing 'gRPC - ReportProcessingCapability'")

	return &fenixExecutionServerGrpcApi.AckNackResponse{AckNack: true, Comments: ""}, nil
}

// ReportCompleteTestInstructionExecutionResult - *********************************************************************
// When a TestInstruction has been fully executed the Client use this to inform the results of the execution result to the Server
func (s *fenixExecutionServerGrpcServicesServer) ReportCompleteTestInstructionExecutionResult(ctx context.Context, finalTestInstructionExecutionResultMessage *fenixExecutionServerGrpcApi.FinalTestInstructionExecutionResultMessage) (*fenixExecutionServerGrpcApi.AckNackResponse, error) {

	fenixExecutionServerObject.logger.WithFields(logrus.Fields{
		"id": "299bac9a-bb4c-4dcd-9ca6-e486efc9e112",
	}).Debug("Incoming 'gRPC - ReportCompleteTestInstructionExecutionResult'")

	defer fenixExecutionServerObject.logger.WithFields(logrus.Fields{
		"id": "61d0939d-bc96-46ea-9623-190cd2942d3e",
	}).Debug("Outgoing 'gRPC - ReportCompleteTestInstructionExecutionResult'")

	return &fenixExecutionServerGrpcApi.AckNackResponse{AckNack: true, Comments: ""}, nil
}

// ReportCurrentTestInstructionExecutionResult - *********************************************************************
// During a TestInstruction execution the Client use this to inform the current of the execution result to the Server
func (s *fenixExecutionServerGrpcServicesServer) ReportCurrentTestInstructionExecutionResult(currentTestInstructionExecutionResultMessageStream fenixExecutionServerGrpcApi.FenixExecutionServerGrpcServices_ReportCurrentTestInstructionExecutionResultServer) (*fenixExecutionServerGrpcApi.AckNackResponse, error) {

	fenixExecutionServerObject.logger.WithFields(logrus.Fields{
		"id": "8a01617a-6cfc-4684-98de-52631edfd2c4",
	}).Debug("Incoming 'gRPC - ReportCurrentTestInstructionExecutionResult'")

	defer fenixExecutionServerObject.logger.WithFields(logrus.Fields{
		"id": "2a5d06a5-730d-447e-8a95-dbe4298122ce",
	}).Debug("Outgoing 'gRPC - ReportCurrentTestInstructionExecutionResult'")

	// Container to store all messages before process them
	var currentTestInstructionExecutionResultMessages []*fenixExecutionServerGrpcApi.CurrentTestInstructionExecutionResultMessage
	var currentTestInstructionExecutionResultMessage *fenixExecutionServerGrpcApi.CurrentTestInstructionExecutionResultMessage

	var err error
	var returnMessage *fenixExecutionServerGrpcApi.AckNackResponse

	// Retrieve stream from Client
	for {
		// Receive message and add it to 'currentTestInstructionExecutionResultMessages'
		currentTestInstructionExecutionResultMessage, err = currentTestInstructionExecutionResultMessageStream.Recv()
		currentTestInstructionExecutionResultMessages = append(currentTestInstructionExecutionResultMessages, currentTestInstructionExecutionResultMessage)

		// When no more messages is received then continue
		if err == io.EOF {
			break
		}
	}

	fmt.Println(currentTestInstructionExecutionResultMessages)

	returnMessage = &fenixExecutionServerGrpcApi.AckNackResponse{
		AckNack:  true,
		Comments: ""}

	return returnMessage, nil
}

// SendLogPostForExecution - *********************************************************************
// During the execution the Client can send log information that can be shown to the user
func (s *fenixExecutionServerGrpcServicesServer) SendLogPostForExecution(logPostsMessageStream fenixExecutionServerGrpcApi.FenixExecutionServerGrpcServices_SendLogPostForExecutionServer) (*fenixExecutionServerGrpcApi.AckNackResponse, error) {

	fenixExecutionServerObject.logger.WithFields(logrus.Fields{
		"id": "ce04f087-1721-44f9-8534-e4c8ae87f18d",
	}).Debug("Incoming 'gRPC - SendLogPostForExecution'")

	defer fenixExecutionServerObject.logger.WithFields(logrus.Fields{
		"id": "3cabce48-2b4a-4e02-a8fa-00be9d108bbd",
	}).Debug("Outgoing 'gRPC - SendLogPostForExecution'")

	// Container to store all messages before process them
	var logPostsMessages []*fenixExecutionServerGrpcApi.LogPostsMessage
	var logPostsMessage *fenixExecutionServerGrpcApi.LogPostsMessage

	var err error
	var returnMessage *fenixExecutionServerGrpcApi.AckNackResponse

	// Retrieve stream from Client
	for {
		// Receive message and add it to 'currentTestInstructionExecutionResultMessages'
		logPostsMessage, err = logPostsMessageStream.Recv()
		logPostsMessages = append(logPostsMessages, logPostsMessage)

		// When no more messages is received then continue
		if err == io.EOF {
			break
		}
	}

	fmt.Println(logPostsMessages)

	returnMessage = &fenixExecutionServerGrpcApi.AckNackResponse{
		AckNack:  true,
		Comments: ""}

	return returnMessage, nil

}
