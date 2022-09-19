package common_config

import (
	"crypto/sha256"
	"encoding/hex"
	fenixTestDataSyncServerGrpcApi "github.com/jlambert68/FenixGrpcApi/Fenix/fenixTestDataSyncServerGrpcApi/go_grpc_api"
	"google.golang.org/protobuf/types/known/timestamppb"
	"time"
)

// Exctract Values, and create, for TestDataHeaderItemMessageHash
func CreateTestDataHeaderItemMessageHash(testDataHeaderItemMessage *fenixTestDataSyncServerGrpcApi.TestDataHeaderItemMessage) (testDataHeaderItemMessageHash string) {

	var valuesToHash []string
	var valueToHash string

	// Extract and add values to array
	// HeaderLabel
	valueToHash = testDataHeaderItemMessage.HeaderLabel
	valuesToHash = append(valuesToHash, valueToHash)

	// HeaderShouldBeUsedForTestDataFilter as 'true' or 'false'
	if testDataHeaderItemMessage.HeaderShouldBeUsedForTestDataFilter == false {
		valuesToHash = append(valuesToHash, "false")
	} else {
		valuesToHash = append(valuesToHash, "true")
	}

	// HeaderIsMandatoryInTestDataFilter as 'true' or 'false'
	if testDataHeaderItemMessage.HeaderIsMandatoryInTestDataFilter == false {
		valuesToHash = append(valuesToHash, "false")
	} else {
		valuesToHash = append(valuesToHash, "true")
	}

	// HeaderSelectionType
	valueToHash = testDataHeaderItemMessage.HeaderSelectionType.String()
	valuesToHash = append(valuesToHash, valueToHash)

	// HeaderFilterValues - An array thar is added
	for _, headerFilterValue := range testDataHeaderItemMessage.HeaderFilterValues {
		headerFilterValueToAdd := headerFilterValue.String()
		valuesToHash = append(valuesToHash, headerFilterValueToAdd)
	}

	// Hash all values in the array
	testDataHeaderItemMessageHash = HashValues(valuesToHash, true)

	return testDataHeaderItemMessageHash
}

// Hash a single value
func HashSingleValue(valueToHash string) (hashValue string) {

	hash := sha256.New()
	hash.Write([]byte(valueToHash))
	hashValue = hex.EncodeToString(hash.Sum(nil))

	return hashValue

}

// GenerateDatetimeTimeStampForDB
// Generate DataBaseTimeStamp, eg '2022-02-08 17:35:04.000000'
func GenerateDatetimeTimeStampForDB() (currentTimeStampAsString string) {

	timeStampLayOut := "2006-01-02 15:04:05.000000" //milliseconds
	currentTimeStamp := time.Now()
	currentTimeStampAsString = currentTimeStamp.Format(timeStampLayOut)

	return currentTimeStampAsString
}

// ConvertGrpcTimeStampToStringForDB
// Convert a gRPC-timestamp into a string that can be used to store in the database
func ConvertGrpcTimeStampToStringForDB(grpcTimeStamp *timestamppb.Timestamp) (grpcTimeStampAsTimeStampAsString string) {
	grpcTimeStampAsTimeStamp := grpcTimeStamp.AsTime()

	timeStampLayOut := "2006-01-02 15:04:05.000000" //milliseconds

	grpcTimeStampAsTimeStampAsString = grpcTimeStampAsTimeStamp.Format(timeStampLayOut)

	return grpcTimeStampAsTimeStampAsString
}
