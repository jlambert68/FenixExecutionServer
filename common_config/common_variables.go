package common_config

import (
	fenixExecutionServerGrpcApi "github.com/jlambert68/FenixGrpcApi/FenixExecutionServer/fenixExecutionServerGrpcApi/go_grpc_api"
	"google.golang.org/grpc"
)

// Used for keeping track of the proto file versions for ExecutionServer and this Worker
var highestFenixExecutionServerProtoFileVersion int32 = -1

//var highestExecutionWorkerProtoFileVersion int32 = -1

// ExecutionWorkerVariablesStruct
// Structure that keeps track of one individual Workers variables
type ExecutionWorkerVariablesStruct struct {
	HighestExecutionWorkerProtoFileVersion int32
	FenixExecutionWorkerServerAddress      string

	// Variables used for contacting Fenix Execution Worker Server
	RemoteFenixExecutionWorkerServerConnection *grpc.ClientConn
	FenixExecutionServerAddressToDial          string
	FenixExecutionWorkerServerGrpcClient       fenixExecutionServerGrpcApi.FenixExecutionServerGrpcServicesClient
	FenixExecutionServerWorkerAddressToUse     string
}

// ExecutionWorkerVariablesMap
// Map that keeps track of all individuals Workers variables
var ExecutionWorkerVariablesMap map[string]*ExecutionWorkerVariablesStruct //map[DomainUUID]*ExecutionWorkerVariablesStruct
