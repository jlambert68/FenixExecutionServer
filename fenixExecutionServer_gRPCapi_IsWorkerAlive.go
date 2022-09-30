package main

import (
	"FenixExecutionServer/common_config"
	"FenixExecutionServer/messagesToExecutionWorker"
	"fmt"
	fenixExecutionServerGrpcApi "github.com/jlambert68/FenixGrpcApi/FenixExecutionServer/fenixExecutionServerGrpcApi/go_grpc_api"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"
)

// AreYouAlive - *********************************************************************
// Ask Fenix Execution Server to call a specific Worker to see if the Worker is alive
func (s *fenixExecutionServerGrpcServicesServer) IsWorkerAlive(ctx context.Context, isWorkerAliveRequest *fenixExecutionServerGrpcApi.IsWorkerAliveRequest) (responseMessage *fenixExecutionServerGrpcApi.AckNackResponse, err error) {

	fenixExecutionServerObject.logger.WithFields(logrus.Fields{
		"id": "877d0a0d-8c6d-47ce-b072-7b93f7e6aea0",
	}).Debug("Incoming 'gRPC - IsWorkerAlive'")

	defer fenixExecutionServerObject.logger.WithFields(logrus.Fields{
		"id": "dc32da91-e261-4236-b9a3-01f79b988d95",
	}).Debug("Outgoing 'gRPC - IsWorkerAlive'")

	// Current user
	userID := "gRPC-api doesn't support UserId"

	// Check if Client is using correct proto files version
	returnMessage := common_config.IsClientUsingCorrectTestDataProtoFileVersion(userID, fenixExecutionServerGrpcApi.CurrentFenixExecutionServerProtoFileVersionEnum(isWorkerAliveRequest.ProtoFileVersionUsedByClient))
	if returnMessage != nil {

		// Exiting
		return returnMessage, nil
	}

	// Set up instance to use for execution gPRC
	var fenixExecutionWorkerObject *messagesToExecutionWorker.MessagesToExecutionWorkerServerObjectStruct
	fenixExecutionWorkerObject = &messagesToExecutionWorker.MessagesToExecutionWorkerServerObjectStruct{Logger: s.logger}

	responseFromWorker := fenixExecutionWorkerObject.SendAreYouAliveToExecutionWorkerServer(isWorkerAliveRequest.WorkersDomainUuid)

	// Convert Error Codes
	var errorCodes []fenixExecutionServerGrpcApi.ErrorCodesEnum
	var errorCode fenixExecutionServerGrpcApi.ErrorCodesEnum

	for _, errorCodeFromWorker := range responseFromWorker.ErrorCodes {
		errorCode = fenixExecutionServerGrpcApi.ErrorCodesEnum(errorCodeFromWorker)
		errorCodes = append(errorCodes, errorCode)
	}

	responseMessage = &fenixExecutionServerGrpcApi.AckNackResponse{
		AckNack:                      responseFromWorker.AckNack,
		Comments:                     fmt.Sprintf("The response from Worker with DomainUuid %s is '%s'", isWorkerAliveRequest.WorkersDomainUuid, responseFromWorker.Comments),
		ErrorCodes:                   errorCodes,
		ProtoFileVersionUsedByClient: fenixExecutionServerGrpcApi.CurrentFenixExecutionServerProtoFileVersionEnum(common_config.GetHighestFenixExecutionServerProtoFileVersion()),
	}

	return responseMessage, nil

}
