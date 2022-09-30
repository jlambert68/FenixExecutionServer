package common_config

import (
	fenixExecutionServerGrpcApi "github.com/jlambert68/FenixGrpcApi/FenixExecutionServer/fenixExecutionServerGrpcApi/go_grpc_api"
	fenixExecutionWorkerGrpcApi "github.com/jlambert68/FenixGrpcApi/FenixExecutionServer/fenixExecutionWorkerGrpcApi/go_grpc_api"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ConvertGrpcTimeStampToStringForDB
// Convert a gRPC-timestamp into a string that can be used to store in the database
func ConvertGrpcTimeStampToStringForDB(grpcTimeStamp *timestamppb.Timestamp) (grpcTimeStampAsTimeStampAsString string) {
	grpcTimeStampAsTimeStamp := grpcTimeStamp.AsTime()

	timeStampLayOut := "2006-01-02 15:04:05.000000" //milliseconds

	grpcTimeStampAsTimeStampAsString = grpcTimeStampAsTimeStamp.Format(timeStampLayOut)

	return grpcTimeStampAsTimeStampAsString
}

// IsClientUsingCorrectTestDataProtoFileVersion
// Check if Calling Client is using correct proto-file version
func IsClientUsingCorrectTestDataProtoFileVersion(callingClientUuid string, usedProtoFileVersion fenixExecutionServerGrpcApi.CurrentFenixExecutionServerProtoFileVersionEnum) (returnMessage *fenixExecutionServerGrpcApi.AckNackResponse) {

	var clientUseCorrectProtoFileVersion bool
	var protoFileExpected fenixExecutionServerGrpcApi.CurrentFenixExecutionServerProtoFileVersionEnum
	var protoFileUsed fenixExecutionServerGrpcApi.CurrentFenixExecutionServerProtoFileVersionEnum

	protoFileUsed = usedProtoFileVersion
	protoFileExpected = fenixExecutionServerGrpcApi.CurrentFenixExecutionServerProtoFileVersionEnum(GetHighestFenixExecutionServerProtoFileVersion())

	// Check if correct proto files is used
	if protoFileExpected == protoFileUsed {
		clientUseCorrectProtoFileVersion = true
	} else {
		clientUseCorrectProtoFileVersion = false
	}

	// Check if Client is using correct proto files version
	if clientUseCorrectProtoFileVersion == false {
		// Not correct proto-file version is used

		// Set Error codes to return message
		var errorCodes []fenixExecutionServerGrpcApi.ErrorCodesEnum
		var errorCode fenixExecutionServerGrpcApi.ErrorCodesEnum

		errorCode = fenixExecutionServerGrpcApi.ErrorCodesEnum_ERROR_WRONG_PROTO_FILE_VERSION
		errorCodes = append(errorCodes, errorCode)

		// Create Return message
		returnMessage = &fenixExecutionServerGrpcApi.AckNackResponse{
			AckNack:                      false,
			Comments:                     "Wrong proto file used. Expected: '" + protoFileExpected.String() + "', but got: '" + protoFileUsed.String() + "'",
			ErrorCodes:                   errorCodes,
			ProtoFileVersionUsedByClient: protoFileExpected,
		}

		return returnMessage

	} else {
		return nil
	}

}

// GetHighestFenixExecutionServerProtoFileVersion
// Get the highest FenixProtoFileVersionEnumeration for ExecutionServer-gRPC-api
func GetHighestFenixExecutionServerProtoFileVersion() int32 {

	// Check if there already is a 'highestFenixExecutionServerProtoFileVersion' saved, if so use that one
	if highestFenixExecutionServerProtoFileVersion != -1 {
		return highestFenixExecutionServerProtoFileVersion
	}

	// Find the highest value for proto-file version
	var maxValue int32
	maxValue = 0

	for _, v := range fenixExecutionServerGrpcApi.CurrentFenixExecutionServerProtoFileVersionEnum_value {
		if v > maxValue {
			maxValue = v
		}
	}

	highestFenixExecutionServerProtoFileVersion = maxValue

	return highestFenixExecutionServerProtoFileVersion
}

// GetHighestExecutionWorkerProtoFileVersion
// Get the highest ClientProtoFileVersionEnumeration for Execution Worker
func GetHighestExecutionWorkerProtoFileVersion(domainUuid string) int32 {

	// Get WorkerVariablesReference
	workerVariables := getWorkerVariablesReference(domainUuid)

	// Check if there already is a 'highestclientProtoFileVersion' saved, if so use that one
	if workerVariables.HighestExecutionWorkerProtoFileVersion != -1 {
		return workerVariables.HighestExecutionWorkerProtoFileVersion
	}

	// Find the highest value for proto-file version
	var maxValue int32
	maxValue = 0

	for _, v := range fenixExecutionWorkerGrpcApi.CurrentFenixExecutionWorkerProtoFileVersionEnum_value {
		if v > maxValue {
			maxValue = v
		}
	}

	workerVariables.HighestExecutionWorkerProtoFileVersion = maxValue

	return workerVariables.HighestExecutionWorkerProtoFileVersion
}
