package messagesToExecutionWorker

import (
	"FenixExecutionServer/common_config"
	"context"
	"fmt"
	fenixExecutionWorkerGrpcApi "github.com/jlambert68/FenixGrpcApi/FenixExecutionServer/fenixExecutionWorkerGrpcApi/go_grpc_api"
	"github.com/sirupsen/logrus"
	"time"
)

// SendProcessTestInstructionExecutionToExecutionWorkerServer - Fenix Execution Server send a task to execute to correct Worker
func (fenixExecutionWorkerObject *MessagesToExecutionWorkerServerObjectStruct) SendProcessTestInstructionExecutionToExecutionWorkerServer(domainUuid string, processTestInstructionExecutionRequest *fenixExecutionWorkerGrpcApi.ProcessTestInstructionExecutionReveredRequest) (processTestInstructionExecutionResponse *fenixExecutionWorkerGrpcApi.ProcessTestInstructionExecutionResponse) {

	common_config.Logger.WithFields(logrus.Fields{
		"id":                                     "3d3de917-77fe-4768-a5a5-7e107173d74f",
		"domainUuid":                             domainUuid,
		"processTestInstructionExecutionRequest": processTestInstructionExecutionRequest,
	}).Debug("Incoming 'SendProcessTestInstructionExecutionToExecutionWorkerServer'")

	common_config.Logger.WithFields(logrus.Fields{
		"id": "787a2437-7a81-4629-a8ef-ca676a9e18d3",
	}).Debug("Outgoing 'SendProcessTestInstructionExecutionToExecutionWorkerServer'")

	var ctx context.Context
	var returnMessageAckNack bool
	var returnMessageString string
	var err error

	// Get WorkerVariablesReference
	workerVariables := fenixExecutionWorkerObject.getWorkerVariablesReference(domainUuid)

	// Only create a new connection of there are already are one
	if workerVariables.FenixExecutionWorkerServerGrpcClient == nil {

		// Set up connection to Server
		err = fenixExecutionWorkerObject.SetConnectionToExecutionWorkerServer(domainUuid)
		if err != nil {

			// Set Error codes to return message
			var errorCodes []fenixExecutionWorkerGrpcApi.ErrorCodesEnum
			var errorCode fenixExecutionWorkerGrpcApi.ErrorCodesEnum

			errorCode = fenixExecutionWorkerGrpcApi.ErrorCodesEnum_ERROR_UNSPECIFIED
			errorCodes = append(errorCodes, errorCode)

			// Create Return message
			ackNackResponse := &fenixExecutionWorkerGrpcApi.AckNackResponse{
				AckNack:    false,
				Comments:   fmt.Sprintf("Couldn't set up connection to Worker with DomainUuid: %s", domainUuid),
				ErrorCodes: errorCodes,
			}
			processTestInstructionExecutionResponse = &fenixExecutionWorkerGrpcApi.ProcessTestInstructionExecutionResponse{
				AckNackResponse:                ackNackResponse,
				TestInstructionExecutionUuid:   processTestInstructionExecutionRequest.TestInstruction.TestInstructionExecutionUuid,
				ExpectedExecutionDuration:      nil,
				TestInstructionCanBeReExecuted: true, // Can be reprocessed because it was only Set up connection that failed
			}

			return processTestInstructionExecutionResponse
		}
	}

	// Do gRPC-call
	//ctx = context.Background()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	// Add Client Deadline
	//clientDeadline := time.Now().Add(time.Duration(common_config.DeadlineForOutgoingGrpc) * time.Millisecond)
	//ctx, cancel := context.WithDeadline(ctx, clientDeadline)
	defer func() {
		common_config.Logger.WithFields(logrus.Fields{
			"ID": "e4992093-6d22-40d6-a30c-f1e14e05253d",
		}).Debug("Running Defer Cancel function")
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
			ackNackResponse := &fenixExecutionWorkerGrpcApi.AckNackResponse{
				AckNack:    false,
				Comments:   fmt.Sprintf("Couldn't generate GCPAccessToken for Worker with DomainUuid: '%s'. Return message: '%s'", domainUuid, returnMessageString),
				ErrorCodes: errorCodes,
			}
			processTestInstructionExecutionResponse = &fenixExecutionWorkerGrpcApi.ProcessTestInstructionExecutionResponse{
				AckNackResponse:                ackNackResponse,
				TestInstructionExecutionUuid:   processTestInstructionExecutionRequest.TestInstruction.TestInstructionExecutionUuid,
				ExpectedExecutionDuration:      nil,
				TestInstructionCanBeReExecuted: true, // Can be reprocessed because it was only problem generating GCP-token
			}

			return processTestInstructionExecutionResponse

		}

	}

	// Finalize message to be sent to Worker
	processTestInstructionExecutionRequest.ProtoFileVersionUsedByClient = fenixExecutionWorkerGrpcApi.CurrentFenixExecutionWorkerProtoFileVersionEnum(common_config.GetHighestExecutionWorkerProtoFileVersion(domainUuid))

	// slice with sleep time, in milliseconds, between each attempt to do gRPC-call to Worker
	var sleepTimeBetweenGrpcCallAttempts []int
	sleepTimeBetweenGrpcCallAttempts = []int{100, 200, 300, 300, 500, 500, 1000, 1000, 1000, 1000} // Total: 5.9 seconds

	// Do multiple attempts to do gRPC-call to Execution Worker, when it fails
	var numberOfgRPCCallAttempts int
	var gRPCCallAttemptCounter int
	numberOfgRPCCallAttempts = len(sleepTimeBetweenGrpcCallAttempts)
	gRPCCallAttemptCounter = 0

	for {

		// Do gRPC-call to Worker
		processTestInstructionExecutionResponse, err = workerVariables.FenixExecutionWorkerServerGrpcClient.ProcessTestInstructionExecution(ctx, processTestInstructionExecutionRequest)

		// Exit when there was a success call
		if err == nil && processTestInstructionExecutionResponse.AckNackResponse.AckNack == true {
			return processTestInstructionExecutionResponse
		}

		// Add to counter for how many gRPC-call-attempts to Worker that have been done
		gRPCCallAttemptCounter = gRPCCallAttemptCounter + 1

		// Shouldn't happen
		if err != nil {
			common_config.Logger.WithFields(logrus.Fields{
				"ID":                     "e0e2175f-6ea0-4437-92dd-5f83359c8ea5",
				"error":                  err,
				"domainUuid":             domainUuid,
				"gRPCCallAttemptCounter": gRPCCallAttemptCounter,
			}).Error("Problem to do gRPC-call to FenixExecutionWorkerServer for 'ProcessTestInstructionExecution'")

			// Only return the error after last attempt
			if gRPCCallAttemptCounter >= numberOfgRPCCallAttempts {

				// Set Error codes to return message
				var errorCodes []fenixExecutionWorkerGrpcApi.ErrorCodesEnum
				var errorCode fenixExecutionWorkerGrpcApi.ErrorCodesEnum

				errorCode = fenixExecutionWorkerGrpcApi.ErrorCodesEnum_ERROR_UNSPECIFIED
				errorCodes = append(errorCodes, errorCode)

				// Create Return message
				ackNackResponse := &fenixExecutionWorkerGrpcApi.AckNackResponse{
					AckNack:    false,
					Comments:   fmt.Sprintf("Error when doing gRPC-call for Worker with DomainUuid: %s. Error message is: '%s'", domainUuid, err.Error()),
					ErrorCodes: errorCodes,
				}
				processTestInstructionExecutionResponse = &fenixExecutionWorkerGrpcApi.ProcessTestInstructionExecutionResponse{
					AckNackResponse:                ackNackResponse,
					TestInstructionExecutionUuid:   processTestInstructionExecutionRequest.TestInstruction.TestInstructionExecutionUuid,
					ExpectedExecutionDuration:      nil,
					TestInstructionCanBeReExecuted: true, // Can be reprocessed because it was some problem doing the gRPC-call
				}

				return processTestInstructionExecutionResponse

			}

			// Sleep for some time before retrying to connect
			time.Sleep(time.Millisecond * time.Duration(sleepTimeBetweenGrpcCallAttempts[gRPCCallAttemptCounter-1]))

		} else if processTestInstructionExecutionResponse.AckNackResponse.AckNack == false {
			// ExecutionWorker couldn't handle gPRC call
			common_config.Logger.WithFields(logrus.Fields{
				"ID":                  "c104fc85-c6ca-4084-a756-409e53491bfe",
				"domainUuid":          domainUuid,
				"Message from Worker": processTestInstructionExecutionResponse.AckNackResponse.Comments,
			}).Error("Problem to do gRPC-call to FenixExecutionWorkerServer for 'SendProcessTestInstructionExecutionToExecutionWorkerServer'")

			// Set Error codes to return message
			var errorCodes []fenixExecutionWorkerGrpcApi.ErrorCodesEnum
			var errorCode fenixExecutionWorkerGrpcApi.ErrorCodesEnum

			errorCode = fenixExecutionWorkerGrpcApi.ErrorCodesEnum_ERROR_UNSPECIFIED
			errorCodes = append(errorCodes, errorCode)

			// Create Return message
			ackNackResponse := &fenixExecutionWorkerGrpcApi.AckNackResponse{
				AckNack:    false,
				Comments:   fmt.Sprintf("AckNack=false when doing gRPC-call for Worker with DomainUuid: %s. Message is: '%s'", domainUuid, processTestInstructionExecutionResponse.AckNackResponse.Comments),
				ErrorCodes: errorCodes,
			}
			processTestInstructionExecutionResponse = &fenixExecutionWorkerGrpcApi.ProcessTestInstructionExecutionResponse{
				AckNackResponse:                ackNackResponse,
				TestInstructionExecutionUuid:   processTestInstructionExecutionRequest.TestInstruction.TestInstructionExecutionUuid,
				ExpectedExecutionDuration:      nil,
				TestInstructionCanBeReExecuted: true, // Can be reprocessed because it was some problem doing the gRPC-call
			}

			return processTestInstructionExecutionResponse

		}

	}

	return processTestInstructionExecutionResponse

}
