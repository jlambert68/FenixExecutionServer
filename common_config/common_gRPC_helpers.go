package common_config

import (
	fenixExecutionServerGrpcApi "github.com/jlambert68/FenixGrpcApi/FenixExecutionServer/fenixExecutionServerGrpcApi/go_grpc_api"
	fenixExecutionWorkerGrpcApi "github.com/jlambert68/FenixGrpcApi/FenixExecutionServer/fenixExecutionWorkerGrpcApi/go_grpc_api"
)

// IsCallerUsingCorrectWorkerProtoFileVersion ********************************************************************************************************************
// Check if Caller  is using correct proto-file version
func IsCallerUsingCorrectWorkerProtoFileVersion(callingClientUuid string, usedProtoFileVersion fenixExecutionWorkerGrpcApi.CurrentFenixExecutionWorkerProtoFileVersionEnum) (returnMessage *fenixExecutionWorkerGrpcApi.AckNackResponse) {

	var callerUseCorrectProtoFileVersion bool
	var protoFileExpected fenixExecutionWorkerGrpcApi.CurrentFenixExecutionWorkerProtoFileVersionEnum
	var protoFileUsed fenixExecutionWorkerGrpcApi.CurrentFenixExecutionWorkerProtoFileVersionEnum

	protoFileUsed = usedProtoFileVersion
	protoFileExpected = fenixExecutionWorkerGrpcApi.CurrentFenixExecutionWorkerProtoFileVersionEnum(GetHighestFenixExecutionServerProtoFileVersion())

	// Check if correct proto files is used
	if protoFileExpected == protoFileUsed {
		callerUseCorrectProtoFileVersion = true
	} else {
		callerUseCorrectProtoFileVersion = false
	}

	// Check if Client is using correct proto files version
	if callerUseCorrectProtoFileVersion == false {
		// Not correct proto-file version is used

		// Set Error codes to return message
		var errorCodes []fenixExecutionWorkerGrpcApi.ErrorCodesEnum
		var errorCode fenixExecutionWorkerGrpcApi.ErrorCodesEnum

		errorCode = fenixExecutionWorkerGrpcApi.ErrorCodesEnum_ERROR_WRONG_PROTO_FILE_VERSION
		errorCodes = append(errorCodes, errorCode)

		// Create Return message
		returnMessage = &fenixExecutionWorkerGrpcApi.AckNackResponse{
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

// ********************************************************************************************************************

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

// ********************************************************************************************************************
// Get the highest ClientProtoFileVersionEnumeration for Execution Worker
func GetHighestExecutionWorkerProtoFileVersion() int32 {

	// Check if there already is a 'highestclientProtoFileVersion' saved, if so use that one
	if highestExecutionWorkerProtoFileVersion != -1 {
		return highestExecutionWorkerProtoFileVersion
	}

	// Find the highest value for proto-file version
	var maxValue int32
	maxValue = 0

	for _, v := range fenixExecutionWorkerGrpcApi.CurrentFenixExecutionWorkerProtoFileVersionEnum_value {
		if v > maxValue {
			maxValue = v
		}
	}

	highestExecutionWorkerProtoFileVersion = maxValue

	return highestExecutionWorkerProtoFileVersion
}
