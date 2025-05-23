package common_config

import (
	fenixExecutionWorkerGrpcApi "github.com/jlambert68/FenixGrpcApi/FenixExecutionServer/fenixExecutionWorkerGrpcApi/go_grpc_api"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

// ApplicationRuntimeUuid
// Keeps a unique id for the runtime instance of the application
var ApplicationRuntimeUuid string

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
	FenixExecutionServerWorkerAddressToDial    string
	FenixExecutionWorkerServerGrpcClient       fenixExecutionWorkerGrpcApi.FenixExecutionWorkerGrpcServicesClient
	FenixExecutionServerWorkerAddressToUse     string
}

// ExecutionWorkerVariablesMap
// Map that keeps track of all individuals Workers variables
var ExecutionWorkerVariablesMap map[string]*ExecutionWorkerVariablesStruct //map[DomainUUID]*ExecutionWorkerVariablesStruct

// Logger that can be used by all part of the Execution Server
var Logger *logrus.Logger

// NumberOfParallellTimeOutChannels
// The number of parallell executions tracks for TimerOut-engine
const NumberOfParallellTimeOutChannels = 20

// Used to calculate which TimeOut-track to use
const NumberOfCharactersToUseFromTestInstructionExecutionUuid = 4

// NumberOfParallellExecutionEngineCommandChannels
// The number of parallell executions tracks for ExecutionEngine
const NumberOfParallellExecutionEngineCommandChannels = 20

// Deadline for outgoing gRPC-call, in millisecond
const DeadlineForOutgoingGrpc = 10000

// Max number of Resent to Worker before error
const MaxResendToWorkerWhenNoAnswer = 10

const ZeroUuid = "00000000-0000-0000-0000-000000000000"
