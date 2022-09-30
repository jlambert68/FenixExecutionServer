package messagesToExecutionWorker

import (
	"FenixExecutionServer/common_config"
	"context"
	fenixExecutionServerGrpcApi "github.com/jlambert68/FenixGrpcApi/FenixExecutionServer/fenixExecutionServerGrpcApi/go_grpc_api"
	"github.com/sirupsen/logrus"
	"time"
)

// SendAreYouAliveToFenixExecutionServer - Ask Fenix Execution Server to check if it's up and running
func (fenixExecutionWorkerObject *MessagesToExecutionWorkerServerObjectStruct) SendAreYouAliveToFenixExecutionServer(domainUuid string) (bool, string) {

	var ctx context.Context
	var returnMessageAckNack bool
	var returnMessageString string

	// Get WorkerVariablesReference
	workerVariables := fenixExecutionWorkerObject.getWorkerVariablesReference(domainUuid)

	// Set up connection to Server
	err := fenixExecutionWorkerObject.SetConnectionToFenixTestExecutionServer(domainUuid)
	if err != nil {
		return false, err.Error()
	}

	// Create the message with all test data to be sent to Fenix
	emptyParameter := &fenixExecutionServerGrpcApi.EmptyParameter{

		ProtoFileVersionUsedByClient: fenixExecutionServerGrpcApi.CurrentFenixExecutionServerProtoFileVersionEnum(common_config.GetHighestExecutionWorkerProtoFileVersion(domainUuid)),
	}

	// Do gRPC-call
	//ctx := context.Background()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer func() {
		fenixExecutionWorkerObject.logger.WithFields(logrus.Fields{
			"ID": "c5ba19bd-75ff-4366-818d-745d4d7f1a52",
		}).Error("Running Defer Cancel function")
		cancel()
	}()

	// Only add access token when Worker is run on GCP
	if common_config.ExecutionLocationForWorker == common_config.GCP {

		// Add Access token
		ctx, returnMessageAckNack, returnMessageString = fenixExecutionWorkerObject.generateGCPAccessToken(ctx, domainUuid)
		if returnMessageAckNack == false {
			return false, returnMessageString
		}

	}

	returnMessage, err := workerVariables.FenixExecutionWorkerServerGrpcClient.AreYouAlive(ctx, emptyParameter)

	// Shouldn't happen
	if err != nil {
		fenixExecutionWorkerObject.logger.WithFields(logrus.Fields{
			"ID":         "818aaf0b-4112-4be4-97b9-21cc084c7b8b",
			"error":      err,
			"domainUuid": domainUuid,
		}).Error("Problem to do gRPC-call to FenixExecutionWorkerServer for 'SendAreYouAliveToFenixExecutionServer'")

		return false, err.Error()

	} else if returnMessage.AckNack == false {
		// ExecutionWorker couldn't handle gPRC call
		fenixExecutionWorkerObject.logger.WithFields(logrus.Fields{
			"ID":                                  "2ecbc800-2fb6-4e88-858d-a421b61c5529",
			"domainUuid":                          domainUuid,
			"Message from Fenix Execution Server": returnMessage.Comments,
		}).Error("Problem to do gRPC-call to FenixExecutionWorkerServer for 'SendAreYouAliveToFenixExecutionServer'")

		return false, err.Error()
	}

	return returnMessage.AckNack, returnMessage.Comments

}
