package messagesToExecutionWorker

import (
	"FenixExecutionServer/common_config"
	"context"
	"fmt"
	fenixExecutionWorkerGrpcApi "github.com/jlambert68/FenixGrpcApi/FenixExecutionServer/fenixExecutionWorkerGrpcApi/go_grpc_api"
	"github.com/sirupsen/logrus"
	"time"
)

// SendAreYouAliveToExecutionWorkerServer - Ask Fenix Execution Server to check if a certain Worker is up and running
func (fenixExecutionWorkerObject *MessagesToExecutionWorkerServerObjectStruct) SendAreYouAliveToExecutionWorkerServer(domainUuid string) (returnMessage *fenixExecutionWorkerGrpcApi.AckNackResponse) {

	var ctx context.Context
	var returnMessageAckNack bool
	var returnMessageString string

	// Get WorkerVariablesReference
	workerVariables := fenixExecutionWorkerObject.getWorkerVariablesReference(domainUuid)

	// Set up connection to Server
	err := fenixExecutionWorkerObject.SetConnectionToExecutionWorkerServer(domainUuid)
	if err != nil {

		// Set Error codes to return message
		var errorCodes []fenixExecutionWorkerGrpcApi.ErrorCodesEnum
		var errorCode fenixExecutionWorkerGrpcApi.ErrorCodesEnum

		errorCode = fenixExecutionWorkerGrpcApi.ErrorCodesEnum_ERROR_UNSPECIFIED
		errorCodes = append(errorCodes, errorCode)

		// Create Return message
		returnMessage = &fenixExecutionWorkerGrpcApi.AckNackResponse{
			AckNack:    false,
			Comments:   fmt.Sprintf("Couldn't set up connection to Worker with DomainUuid: %s", domainUuid),
			ErrorCodes: errorCodes,
		}

		return returnMessage
	}

	// Do gRPC-call
	//ctx := context.Background()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer func() {
		fenixExecutionWorkerObject.Logger.WithFields(logrus.Fields{
			"ID": "ba28e796-6873-4e2a-b1b8-935fdd1a0e71",
		}).Error("Running Defer Cancel function")
		cancel()
	}()

	// Only add access token when Worker is run on GCP
	if common_config.ExecutionLocationForWorker == common_config.GCP {

		// Add Access token
		ctx, returnMessageAckNack, returnMessageString = fenixExecutionWorkerObject.generateGCPAccessToken(ctx, domainUuid)
		if returnMessageAckNack == false {

			// Set Error codes to return message
			var errorCodes []fenixExecutionWorkerGrpcApi.ErrorCodesEnum
			var errorCode fenixExecutionWorkerGrpcApi.ErrorCodesEnum

			errorCode = fenixExecutionWorkerGrpcApi.ErrorCodesEnum_ERROR_UNSPECIFIED
			errorCodes = append(errorCodes, errorCode)

			// Create Return message
			returnMessage = &fenixExecutionWorkerGrpcApi.AckNackResponse{
				AckNack:    false,
				Comments:   returnMessageString,
				ErrorCodes: errorCodes,
			}

			return returnMessage

		}

	}

	// Create the message with all test data to be sent to Worker
	emptyParameter := &fenixExecutionWorkerGrpcApi.EmptyParameter{

		ProtoFileVersionUsedByClient: fenixExecutionWorkerGrpcApi.CurrentFenixExecutionWorkerProtoFileVersionEnum(common_config.GetHighestExecutionWorkerProtoFileVersion(domainUuid)),
	}

	// Do gRPC-call to Worker
	returnMessage, err = workerVariables.FenixExecutionWorkerServerGrpcClient.AreYouAlive(ctx, emptyParameter)

	// Shouldn't happen
	if err != nil {
		fenixExecutionWorkerObject.Logger.WithFields(logrus.Fields{
			"ID":         "818aaf0b-4112-4be4-97b9-21cc084c7b8b",
			"error":      err,
			"domainUuid": domainUuid,
		}).Error("Problem to do gRPC-call to FenixExecutionWorkerServer for 'SendAreYouAliveToExecutionWorkerServer'")

		// Set Error codes to return message
		var errorCodes []fenixExecutionWorkerGrpcApi.ErrorCodesEnum
		var errorCode fenixExecutionWorkerGrpcApi.ErrorCodesEnum

		errorCode = fenixExecutionWorkerGrpcApi.ErrorCodesEnum_ERROR_UNSPECIFIED
		errorCodes = append(errorCodes, errorCode)

		// Create Return message
		returnMessage = &fenixExecutionWorkerGrpcApi.AckNackResponse{
			AckNack:    false,
			Comments:   err.Error(),
			ErrorCodes: errorCodes,
		}

		return returnMessage

	} else if returnMessage.AckNack == false {
		// ExecutionWorker couldn't handle gPRC call
		fenixExecutionWorkerObject.Logger.WithFields(logrus.Fields{
			"ID":                                  "2ecbc800-2fb6-4e88-858d-a421b61c5529",
			"domainUuid":                          domainUuid,
			"Message from Fenix Execution Server": returnMessage.Comments,
		}).Error("Problem to do gRPC-call to FenixExecutionWorkerServer for 'SendAreYouAliveToExecutionWorkerServer'")

		// Set Error codes to return message
		var errorCodes []fenixExecutionWorkerGrpcApi.ErrorCodesEnum
		var errorCode fenixExecutionWorkerGrpcApi.ErrorCodesEnum

		errorCode = fenixExecutionWorkerGrpcApi.ErrorCodesEnum_ERROR_UNSPECIFIED
		errorCodes = append(errorCodes, errorCode)

		// Create Return message
		returnMessage = &fenixExecutionWorkerGrpcApi.AckNackResponse{
			AckNack:    false,
			Comments:   err.Error(),
			ErrorCodes: errorCodes,
		}

		return returnMessage
	}

	return returnMessage

}
