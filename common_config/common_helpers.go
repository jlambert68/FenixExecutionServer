package common_config

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	fenixTestDataSyncServerGrpcApi "github.com/jlambert68/FenixGrpcApi/Fenix/fenixTestDataSyncServerGrpcApi/go_grpc_api"
	"github.com/sirupsen/logrus"
	"log"
	"runtime"
	"strings"
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

	timeStampLayOut := "2006-01-02 15:04:05.000000 -0700" //milliseconds
	currentTimeStamp := time.Now().UTC()
	currentTimeStampAsString = currentTimeStamp.Format(timeStampLayOut)

	return currentTimeStampAsString
}

// GenerateDatetimeFromTimeInputForDB
// Generate DataBaseTimeStamp, eg '2022-02-08 17:35:04.000000'
func GenerateDatetimeFromTimeInputForDB(currentTime time.Time) (currentTimeStampAsString string) {

	timeStampLayOut := "2006-01-02 15:04:05.000000 -0700" //milliseconds
	currentTimeStampAsString = currentTime.Format(timeStampLayOut)

	return currentTimeStampAsString
}

func getWorkerVariablesReference(domainUuid string) (executionWorkerVariablesReference *ExecutionWorkerVariablesStruct) {

	executionWorkerVariablesReference, existInMap := ExecutionWorkerVariablesMap[domainUuid]
	if existInMap == false {
		errorID := "ab3986b4-e61d-4792-bf68-133a6c057c19"
		log.Fatalln(fmt.Sprintf("Couldn't find DomainUuid %s, [ErrorId: %s"), domainUuid, errorID)
	}

	return executionWorkerVariablesReference
}

// GenerateTimeStampParserLayout
// Extracts 'ParserLayout' from the TimeStamp(as string)
func GenerateTimeStampParserLayout(timeStampAsString string) (parserLayout string, err error) {
	// "2006-01-02 15:04:05.999999999 -0700 MST"

	var timeStampParts []string
	var timeParts []string
	var numberOfDecimals int

	// Split TimeStamp into separate parts
	timeStampParts = strings.Split(timeStampAsString, " ")

	// Validate that first part is a date with the following form '2006-01-02'
	if len(timeStampParts[0]) != 10 {

		Logger.WithFields(logrus.Fields{
			"Id":                "ffbf0682-ebc7-4e27-8ad1-0e5005fbc364",
			"timeStampAsString": timeStampAsString,
			"timeStampParts[0]": timeStampParts[0],
		}).Error("Date part has not the correct form, '2006-01-02'")

		err = errors.New(fmt.Sprintf("Date part, '%s' has not the correct form, '2006-01-02'", timeStampParts[0]))

		return "", err

	}

	// Add Date to Parser Layout
	parserLayout = "2006-01-02"

	// Add Time to Parser Layout
	parserLayout = parserLayout + " 15:04:05."

	// Split time into time and decimals
	timeParts = strings.Split(timeStampParts[1], ".")

	// Get number of decimals
	numberOfDecimals = len(timeParts[1])

	// Add Decimals to Parser Layout
	parserLayout = parserLayout + strings.Repeat("9", numberOfDecimals)

	// Add time zone if that information exists
	if len(timeStampParts) > 3 {
		parserLayout = parserLayout + " -0700 MST"
	}

	return parserLayout, err
}

// PrintMemUsage outputs the current, total and OS memory being used. As well as the number
// of garage collection cycles completed.
func PrintMemUsage() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	// For info on each, see: https://golang.org/pkg/runtime/#MemStats
	//fmt.Printf("Alloc = %v MiB", bToMb(m.Alloc))
	//fmt.Printf("\tTotalAlloc = %v MiB", bToMb(m.TotalAlloc))
	//fmt.Printf("\tSys = %v MiB", bToMb(m.Sys))
	//fmt.Printf("\tNumGC = %v\n", m.NumGC)
	alloc, _ := fmt.Printf("Alloc = %v MiB", bToMb(m.Alloc))
	totalAlloc, _ := fmt.Printf("\tTotalAlloc = %v MiB", bToMb(m.TotalAlloc))
	sys, _ := fmt.Printf("\tSys = %v MiB", bToMb(m.Sys))
	numGC, _ := fmt.Printf("\tNumGC = %v\n", m.NumGC)

	Logger.WithFields(logrus.Fields{
		"Id":            "ffbf0682-ebc7-4e27-8ad1-0e5005fbc364",
		"Alloc":         alloc,
		"TotalAlloc[0]": totalAlloc,
		"Sys[0]":        sys,
		"NumGC[0]":      numGC,
	}).Info("Memory usage")
}

func bToMb(b uint64) uint64 {
	return b / 1024 / 1024
}
