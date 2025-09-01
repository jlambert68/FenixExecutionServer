package testInstructionExecutionEngine

import (
	"FenixExecutionServer/common_config"
	"context"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	uuidGenerator "github.com/google/uuid"
	"github.com/jackc/pgx/v4"
	fenixExecutionServerGrpcApi "github.com/jlambert68/FenixGrpcApi/FenixExecutionServer/fenixExecutionServerGrpcApi/go_grpc_api"
	fenixTestCaseBuilderServerGrpcApi "github.com/jlambert68/FenixGrpcApi/FenixTestCaseBuilderServer/fenixTestCaseBuilderServerGrpcApi/go_grpc_api"
	fenixSyncShared "github.com/jlambert68/FenixSyncShared"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/encoding/protojson"
	"sort"
	"strconv"
	"strings"
	"time"
)

func (executionEngine *TestInstructionExecutionEngineStruct) commitOrRoleBack(
	executionTrackNumberReference *int,
	dbTransactionReference *pgx.Tx,
	doCommitNotRoleBackReference *bool,
	testCaseExecutionsToProcessReference *[]ChannelCommandTestCaseExecutionStruct) {

	executionTrackNumber := *executionTrackNumberReference
	dbTransaction := *dbTransactionReference
	doCommitNotRoleBack := *doCommitNotRoleBackReference
	testCaseExecutionsToProcess := *testCaseExecutionsToProcessReference

	if doCommitNotRoleBack == true {
		dbTransaction.Commit(context.Background())

		// Trigger TestInstructionEngine to check if there are any TestInstructions on the ExecutionQueue
		go func() {
			channelCommandMessage := ChannelCommandStruct{
				ChannelCommand:                   ChannelCommandCheckForTestInstructionExecutionWaitingOnQueue,
				ChannelCommandTestCaseExecutions: testCaseExecutionsToProcess,
			}

			ExecutionEngineCommandChannelSlice[executionTrackNumber] <- channelCommandMessage

		}()
	} else {
		dbTransaction.Rollback(context.Background())
	}
}

// PrepareInformThatThereAreNewTestCasesOnExecutionQueueSaveToCloudDB
// Exposed version
// Prepare for Saving the ongoing Execution of a new TestCaseExecutionUuid in the CloudDB
func (executionEngine *TestInstructionExecutionEngineStruct) PrepareInformThatThereAreNewTestCasesOnExecutionQueueSaveToCloudDB(
	testCaseExecutionsToProcess []ChannelCommandTestCaseExecutionStruct) (
	ackNackResponse *fenixExecutionServerGrpcApi.AckNackResponse) {

	ackNackResponse = executionEngine.prepareInformThatThereAreNewTestCasesOnExecutionQueueSaveToCloudDB(
		testCaseExecutionsToProcess)

	return ackNackResponse
}

// Prepare for Saving the ongoing Execution of a new TestCaseExecutionUuid in the CloudDB
func (executionEngine *TestInstructionExecutionEngineStruct) prepareInformThatThereAreNewTestCasesOnExecutionQueueSaveToCloudDB(
	testCaseExecutionsToProcess []ChannelCommandTestCaseExecutionStruct) (
	ackNackResponse *fenixExecutionServerGrpcApi.AckNackResponse) {

	// Begin SQL Transaction
	txn, err := fenixSyncShared.DbPool.Begin(context.Background())
	if err != nil {
		common_config.Logger.WithFields(logrus.Fields{
			"id":    "306edce0-7a5a-4a0f-992b-5c9b69b0bcc6",
			"error": err,
		}).Error("Problem to do 'DbPool.Begin'  in 'prepareInformThatThereAreNewTestCasesOnExecutionQueueSaveToCloudDB'")

		// Set Error codes to return message
		var errorCodes []fenixExecutionServerGrpcApi.ErrorCodesEnum
		var errorCode fenixExecutionServerGrpcApi.ErrorCodesEnum

		errorCode = fenixExecutionServerGrpcApi.ErrorCodesEnum_ERROR_DATABASE_PROBLEM
		errorCodes = append(errorCodes, errorCode)

		// Create Return message
		ackNackResponse = &fenixExecutionServerGrpcApi.AckNackResponse{
			AckNack:                      false,
			Comments:                     "Problem when saving to database",
			ErrorCodes:                   errorCodes,
			ProtoFileVersionUsedByClient: fenixExecutionServerGrpcApi.CurrentFenixExecutionServerProtoFileVersionEnum(common_config.GetHighestFenixExecutionServerProtoFileVersion()),
		}

		return ackNackResponse
	}

	// After all stuff is done, then Commit or Rollback depending on result
	var doCommitNotRoleBack bool

	// Standard is to do a Rollback
	doCommitNotRoleBack = false

	var executionTrackNumber int

	defer executionEngine.commitOrRoleBack(
		&executionTrackNumber,
		&txn,
		&doCommitNotRoleBack,
		&testCaseExecutionsToProcess) //txn.Commit(context.Background())

	// Generate a new TestCaseExecutionUuid-UUID
	//testCaseExecutionUuid := uuidGenerator.New().String()

	// Generate TimeStamp
	//placedOnTestExecutionQueueTimeStamp := time.Now()

	// Extract TestCaseExecutionQueue-messages to be added to data for ongoing Executions
	var testCaseExecutionQueueMessages []*tempTestCaseExecutionQueueInformationStruct
	testCaseExecutionQueueMessages, err = executionEngine.loadTestCaseExecutionQueueMessages(
		txn, testCaseExecutionsToProcess)
	if err != nil {

		// Set Error codes to return message
		var errorCodes []fenixExecutionServerGrpcApi.ErrorCodesEnum
		var errorCode fenixExecutionServerGrpcApi.ErrorCodesEnum

		errorCode = fenixExecutionServerGrpcApi.ErrorCodesEnum_ERROR_DATABASE_PROBLEM
		errorCodes = append(errorCodes, errorCode)

		// Create Return message
		ackNackResponse := &fenixExecutionServerGrpcApi.AckNackResponse{
			AckNack:                      false,
			Comments:                     "Problem when Loading TestCaseExecutions from ExecutionQueue from database",
			ErrorCodes:                   errorCodes,
			ProtoFileVersionUsedByClient: fenixExecutionServerGrpcApi.CurrentFenixExecutionServerProtoFileVersionEnum(common_config.GetHighestFenixExecutionServerProtoFileVersion()),
		}

		return ackNackResponse
	}

	// If there are no TestCases on Queue the exit
	if testCaseExecutionQueueMessages == nil {
		ackNackResponse = &fenixExecutionServerGrpcApi.AckNackResponse{
			AckNack:                      true,
			Comments:                     "",
			ErrorCodes:                   []fenixExecutionServerGrpcApi.ErrorCodesEnum{},
			ProtoFileVersionUsedByClient: fenixExecutionServerGrpcApi.CurrentFenixExecutionServerProtoFileVersionEnum(common_config.GetHighestFenixExecutionServerProtoFileVersion()),
		}

		return ackNackResponse
	}

	// Extract "lowest" TestCaseExecutionUuid
	if len(testCaseExecutionsToProcess) > 0 {
		var uuidSlice []string
		for _, uuid := range testCaseExecutionsToProcess {
			uuidSlice = append(uuidSlice, uuid.TestCaseExecutionUuid)
		}
		sort.Strings(uuidSlice)

		// Define Execution Track based on "lowest "TestCaseExecutionUuid
		executionTrackNumber = common_config.CalculateExecutionTrackNumber(uuidSlice[0])
	}

	// Save the Initiation of a new TestCaseExecutionUuid in the CloudDB
	err = executionEngine.saveTestCasesOnOngoingExecutionsQueueSaveToCloudDB(
		txn,
		testCaseExecutionQueueMessages)

	if err != nil {

		common_config.Logger.WithFields(logrus.Fields{
			"id":    "bc6f1da5-3c8c-493e-9882-0b20e0da9e2e",
			"error": err,
		}).Error("Couldn't Save TestCaseExecutionQueueMessages to queue for ongoing executions in CloudDB")

		// Rollback any SQL transactions
		txn.Rollback(context.Background())

		// Set Error codes to return message
		var errorCodes []fenixExecutionServerGrpcApi.ErrorCodesEnum
		var errorCode fenixExecutionServerGrpcApi.ErrorCodesEnum

		errorCode = fenixExecutionServerGrpcApi.ErrorCodesEnum_ERROR_DATABASE_PROBLEM
		errorCodes = append(errorCodes, errorCode)

		// Create Return message
		ackNackResponse := &fenixExecutionServerGrpcApi.AckNackResponse{
			AckNack:    false,
			Comments:   "Problem when saving to database",
			ErrorCodes: errorCodes,
			ProtoFileVersionUsedByClient: fenixExecutionServerGrpcApi.
				CurrentFenixExecutionServerProtoFileVersionEnum(common_config.GetHighestFenixExecutionServerProtoFileVersion()),
		}

		return ackNackResponse

	}

	// Delete messages in ExecutionQueue that has been put to ongoing executions
	err = executionEngine.clearTestCasesExecutionQueueSaveToCloudDB(
		txn,
		testCaseExecutionQueueMessages)
	if err != nil {

		common_config.Logger.WithFields(logrus.Fields{
			"id":    "c4836b67-3634-4fe0-bc89-551b2a56ce79",
			"error": err,
		}).Error("Couldn't clear TestCaseExecutionQueue in CloudDB")

		// Rollback any SQL transactions
		txn.Rollback(context.Background())

		// Set Error codes to return message
		var errorCodes []fenixExecutionServerGrpcApi.ErrorCodesEnum
		var errorCode fenixExecutionServerGrpcApi.ErrorCodesEnum

		errorCode = fenixExecutionServerGrpcApi.ErrorCodesEnum_ERROR_DATABASE_PROBLEM
		errorCodes = append(errorCodes, errorCode)

		// Create Return message
		ackNackResponse := &fenixExecutionServerGrpcApi.AckNackResponse{
			AckNack:    false,
			Comments:   "Problem when saving to database",
			ErrorCodes: errorCodes,
			ProtoFileVersionUsedByClient: fenixExecutionServerGrpcApi.
				CurrentFenixExecutionServerProtoFileVersionEnum(common_config.GetHighestFenixExecutionServerProtoFileVersion()),
		}

		return ackNackResponse

	}

	// Save a 'copy' of TestCaseExecution in Table 'TestCasesExecutionsForListings'
	err = executionEngine.saveTestCasesExecutionToTestCasesExecutionsForListingsSaveToCloudDB(
		txn,
		testCaseExecutionQueueMessages)

	if err != nil {

		common_config.Logger.WithFields(logrus.Fields{
			"id":    "d51b884f-9c0d-4712-8687-b7d8be8d90c5",
			"error": err,
		}).Error("Couldn't Save to 'TestCasesExecutionsForListings' in CloudDB")

		// Rollback any SQL transactions
		txn.Rollback(context.Background())

		// Set Error codes to return message
		var errorCodes []fenixExecutionServerGrpcApi.ErrorCodesEnum
		var errorCode fenixExecutionServerGrpcApi.ErrorCodesEnum

		errorCode = fenixExecutionServerGrpcApi.ErrorCodesEnum_ERROR_DATABASE_PROBLEM
		errorCodes = append(errorCodes, errorCode)

		// Create Return message
		ackNackResponse := &fenixExecutionServerGrpcApi.AckNackResponse{
			AckNack:    false,
			Comments:   "Problem when saving to database",
			ErrorCodes: errorCodes,
			ProtoFileVersionUsedByClient: fenixExecutionServerGrpcApi.
				CurrentFenixExecutionServerProtoFileVersionEnum(common_config.GetHighestFenixExecutionServerProtoFileVersion()),
		}

		return ackNackResponse

	}

	// Slice to hold all TestCasesPreview per TestSuite
	var testCasesPreviewMapPerTestSuite map[string][]*fenixTestCaseBuilderServerGrpcApi.TestCasePreviewMessage
	testCasesPreviewMapPerTestSuite = make(map[string][]*fenixTestCaseBuilderServerGrpcApi.TestCasePreviewMessage)
	var existInMap bool
	// Retrieve "TestCasePreview" and add to database row
	for _, tempTestCaseExecutionQueueMessage := range testCaseExecutionQueueMessages {

		var testCasePreview *fenixTestCaseBuilderServerGrpcApi.TestCasePreviewMessage
		testCasePreview, err = executionEngine.loadTestCasePreview(
			txn,
			tempTestCaseExecutionQueueMessage)

		if err != nil {

			common_config.Logger.WithFields(logrus.Fields{
				"id":    "e0fb2bc6-7297-4d26-bb97-0940a3e9f885",
				"error": err,
			}).Error("Couldn't Load 'TestCasePreview'from CloudDB")

			// Rollback any SQL transactions
			txn.Rollback(context.Background())

			// Set Error codes to return message
			var errorCodes []fenixExecutionServerGrpcApi.ErrorCodesEnum
			var errorCode fenixExecutionServerGrpcApi.ErrorCodesEnum

			errorCode = fenixExecutionServerGrpcApi.ErrorCodesEnum_ERROR_DATABASE_PROBLEM
			errorCodes = append(errorCodes, errorCode)

			// Create Return message
			ackNackResponse := &fenixExecutionServerGrpcApi.AckNackResponse{
				AckNack:    false,
				Comments:   "Problem when saving to database",
				ErrorCodes: errorCodes,
				ProtoFileVersionUsedByClient: fenixExecutionServerGrpcApi.
					CurrentFenixExecutionServerProtoFileVersionEnum(common_config.GetHighestFenixExecutionServerProtoFileVersion()),
			}

			return ackNackResponse

		}

		// Add TestCasePreview to slice of preview for the specific TestSuiteUuid
		_, existInMap = testCasesPreviewMapPerTestSuite[tempTestCaseExecutionQueueMessage.testSuiteUuid]
		if existInMap == false {
			testCasesPreviewMapPerTestSuite[tempTestCaseExecutionQueueMessage.testSuiteUuid] = make([]*fenixTestCaseBuilderServerGrpcApi.TestCasePreviewMessage, 0)
		}
		testCasesPreviewMapPerTestSuite[tempTestCaseExecutionQueueMessage.testSuiteUuid] = append(
			testCasesPreviewMapPerTestSuite[tempTestCaseExecutionQueueMessage.testSuiteUuid],
			testCasePreview)

		// Add "TestCasePreview" to 'TestCasesExecutionsForListings'
		err = executionEngine.addTestCasePreviewIntoDatabase(
			txn,
			tempTestCaseExecutionQueueMessage,
			testCasePreview)

		if err != nil {

			common_config.Logger.WithFields(logrus.Fields{
				"id":    "cea6f97f-da6a-4eee-9d2e-eeb85d3d5e20",
				"error": err,
			}).Error("Couldn't Add 'TestCasePreview' and 'ExecutionStatusPreviewValue' to 'TestCasesExecutionsForListings' in CloudDB")

			// Rollback any SQL transactions
			txn.Rollback(context.Background())

			// Set Error codes to return message
			var errorCodes []fenixExecutionServerGrpcApi.ErrorCodesEnum
			var errorCode fenixExecutionServerGrpcApi.ErrorCodesEnum

			errorCode = fenixExecutionServerGrpcApi.ErrorCodesEnum_ERROR_DATABASE_PROBLEM
			errorCodes = append(errorCodes, errorCode)

			// Create Return message
			ackNackResponse := &fenixExecutionServerGrpcApi.AckNackResponse{
				AckNack:    false,
				Comments:   "Problem when saving to database",
				ErrorCodes: errorCodes,
				ProtoFileVersionUsedByClient: fenixExecutionServerGrpcApi.
					CurrentFenixExecutionServerProtoFileVersionEnum(common_config.GetHighestFenixExecutionServerProtoFileVersion()),
			}

			return ackNackResponse

		}
	}

	// Save a TestSuiteExecution in Table 'TestSuitesExecutionsForListings'
	var testCaseUsedForTestSuiteExecutionQueueMessages []*tempTestCaseExecutionQueueInformationStruct
	testCaseUsedForTestSuiteExecutionQueueMessages, err = executionEngine.saveTestSuiteExecutionToTestSuitesExecutionsForListingsSaveToCloudDB(
		txn,
		testCaseExecutionQueueMessages)

	if err != nil {

		common_config.Logger.WithFields(logrus.Fields{
			"id":    "51194244-01f1-4c23-927d-7c47875eb915",
			"error": err,
		}).Error("Couldn't Save to 'TestSuitesExecutionsForListings' in CloudDB")

		// Rollback any SQL transactions
		txn.Rollback(context.Background())

		// Set Error codes to return message
		var errorCodes []fenixExecutionServerGrpcApi.ErrorCodesEnum
		var errorCode fenixExecutionServerGrpcApi.ErrorCodesEnum

		errorCode = fenixExecutionServerGrpcApi.ErrorCodesEnum_ERROR_DATABASE_PROBLEM
		errorCodes = append(errorCodes, errorCode)

		// Create Return message
		ackNackResponse := &fenixExecutionServerGrpcApi.AckNackResponse{
			AckNack:    false,
			Comments:   "Problem when saving to database",
			ErrorCodes: errorCodes,
			ProtoFileVersionUsedByClient: fenixExecutionServerGrpcApi.
				CurrentFenixExecutionServerProtoFileVersionEnum(common_config.GetHighestFenixExecutionServerProtoFileVersion()),
		}

		return ackNackResponse

	}

	// Load all TestSuitePreView-data needed. For all TestSuites
	var testSuitesPreview []*fenixTestCaseBuilderServerGrpcApi.TestSuitePreviewMessage
	testSuitesPreview, err = executionEngine.loadAllTestSuitePreViews(
		txn,
		testCaseUsedForTestSuiteExecutionQueueMessages,
		testCasesPreviewMapPerTestSuite)

	if err != nil {

		common_config.Logger.WithFields(logrus.Fields{
			"id":    "870366e2-09c5-4d03-95e7-48d709c59876",
			"error": err,
		}).Error("Couldn't Load 'TestSuitePreViews' from CloudDB")

		// Rollback any SQL transactions
		txn.Rollback(context.Background())

		// Set Error codes to return message
		var errorCodes []fenixExecutionServerGrpcApi.ErrorCodesEnum
		var errorCode fenixExecutionServerGrpcApi.ErrorCodesEnum

		errorCode = fenixExecutionServerGrpcApi.ErrorCodesEnum_ERROR_DATABASE_PROBLEM
		errorCodes = append(errorCodes, errorCode)

		// Create Return message
		ackNackResponse := &fenixExecutionServerGrpcApi.AckNackResponse{
			AckNack:    false,
			Comments:   "Problem when Loading from database",
			ErrorCodes: errorCodes,
			ProtoFileVersionUsedByClient: fenixExecutionServerGrpcApi.
				CurrentFenixExecutionServerProtoFileVersionEnum(common_config.GetHighestFenixExecutionServerProtoFileVersion()),
		}

		return ackNackResponse

	}

	// Add TestSuitesPreview to database
	err = executionEngine.addTestSuitePreviewIntoDatabase(
		txn,
		testSuitesPreview,
		testCaseUsedForTestSuiteExecutionQueueMessages)

	//Load all data around TestCase to be used for putting TestInstructions on the TestInstructionExecutionQueue
	var allDataAroundAllTestCase []*tempTestInstructionInTestCaseStruct
	allDataAroundAllTestCase, err = executionEngine.
		loadTestCaseModelAndTestInstructionsAndTestInstructionContainersToBeAddedToExecutionQueueLoadFromCloudDB(
			txn,
			testCaseExecutionQueueMessages)

	if err != nil {

		common_config.Logger.WithFields(logrus.Fields{
			"id":    "7c778c1e-c5c2-46c3-a4e3-d59f2208d73b",
			"error": err,
		}).Error("Couldn't load TestInstructions that should be added to the TestInstructionExecutionQueue in CloudDB")

		// Rollback any SQL transactions
		txn.Rollback(context.Background())

		// Set Error codes to return message
		var errorCodes []fenixExecutionServerGrpcApi.ErrorCodesEnum
		var errorCode fenixExecutionServerGrpcApi.ErrorCodesEnum

		errorCode = fenixExecutionServerGrpcApi.ErrorCodesEnum_ERROR_DATABASE_PROBLEM
		errorCodes = append(errorCodes, errorCode)

		// Create Return message
		ackNackResponse := &fenixExecutionServerGrpcApi.AckNackResponse{
			AckNack:                      false,
			Comments:                     "Problem when saving to database",
			ErrorCodes:                   errorCodes,
			ProtoFileVersionUsedByClient: fenixExecutionServerGrpcApi.CurrentFenixExecutionServerProtoFileVersionEnum(common_config.GetHighestFenixExecutionServerProtoFileVersion()),
		}

		return ackNackResponse

	}

	// Add TestInstructions to TestInstructionsExecutionQueue
	var testInstructionUuidTotestInstructionExecutionUuidMap map[string]tempAttributesType
	testInstructionUuidTotestInstructionExecutionUuidMap, err = executionEngine.
		SaveTestInstructionsToExecutionQueueSaveToCloudDB(
			txn,
			testCaseExecutionQueueMessages,
			allDataAroundAllTestCase)

	if err != nil {

		common_config.Logger.WithFields(logrus.Fields{
			"id":    "4bb68279-0dff-426f-a31d-927a7459f324",
			"error": err,
		}).Error("Couldn't save TestInstructions to the TestInstructionExecutionQueue in CloudDB")

		// Rollback any SQL transactions
		txn.Rollback(context.Background())

		// Set Error codes to return message
		var errorCodes []fenixExecutionServerGrpcApi.ErrorCodesEnum
		var errorCode fenixExecutionServerGrpcApi.ErrorCodesEnum

		errorCode = fenixExecutionServerGrpcApi.ErrorCodesEnum_ERROR_DATABASE_PROBLEM
		errorCodes = append(errorCodes, errorCode)

		// Create Return message
		ackNackResponse := &fenixExecutionServerGrpcApi.AckNackResponse{
			AckNack:    false,
			Comments:   "Problem when saving to database",
			ErrorCodes: errorCodes,
			ProtoFileVersionUsedByClient: fenixExecutionServerGrpcApi.CurrentFenixExecutionServerProtoFileVersionEnum(
				common_config.GetHighestFenixExecutionServerProtoFileVersion()),
		}

		return ackNackResponse

	}

	// Add attributes to table for 'TestInstructionAttributesUnderExecution'
	err = executionEngine.saveTestInstructionAttributesUnderExecutionSaveToCloudDB(
		txn,
		testInstructionUuidTotestInstructionExecutionUuidMap)

	if err != nil {

		common_config.Logger.WithFields(logrus.Fields{
			"id":    "d50bded9-f541-4367-b141-b640e7f6fc67",
			"error": err,
		}).Error("Couldn't add TestInstructionAttributes to table in CloudDB")

		return
	}

	ackNackResponse = &fenixExecutionServerGrpcApi.AckNackResponse{
		AckNack:                      true,
		Comments:                     "",
		ErrorCodes:                   []fenixExecutionServerGrpcApi.ErrorCodesEnum{},
		ProtoFileVersionUsedByClient: fenixExecutionServerGrpcApi.CurrentFenixExecutionServerProtoFileVersionEnum(common_config.GetHighestFenixExecutionServerProtoFileVersion()),
	}

	// Commit every database change
	doCommitNotRoleBack = true

	return ackNackResponse
}

// Struct to use with variable to hold TestCaseExecutionQueue-messages
type tempTestCaseExecutionQueueInformationStruct struct {
	domainUuid                 string
	domainName                 string
	testSuiteUuid              string
	testSuiteName              string
	testSuiteVersion           int
	testSuiteExecutionUuid     string
	testSuiteExecutionVersion  int
	testCaseUuid               string
	testCaseName               string
	testCaseVersion            int
	testCaseExecutionUuid      string
	testCaseExecutionVersion   int
	queueTimeStamp             time.Time
	testDataSetUuid            string
	executionPriority          int
	uniqueCounter              int
	executionStatusReportLevel int
}

// Struct to use with variable to hold TestInstructionExecutionQueue-messages
type tempTestInstructionExecutionQueueInformationStruct struct {
	domainUuid                        string
	domainName                        string
	executionDomainUuid               string
	executionDomainName               string
	testInstructionExecutionUuid      string
	matureTestInstructionUuid         string
	testInstructionName               string
	testInstructionMajorVersionNumber int
	testInstructionMinorVersionNumber int
	queueTimeStamp                    string
	executionPriority                 int
	testCaseExecutionUuid             string
	testDataSetUuid                   string
	testCaseExecutionVersion          int
	testInstructionExecutionVersion   int
	testInstructionExecutionOrder     int
	uniqueCounter                     int
	testInstructionOriginalUuid       string
	executionStatusReportLevel        int
}

// Struct to be used when extracting TestInstructions from TestCases
type tempTestInstructionInTestCaseStruct struct {
	domainUuid                       string
	domainName                       string
	executionDomainUuid              string
	executionDomainName              string
	testCaseUuid                     string
	testCaseName                     string
	testCaseVersion                  int
	testCaseBasicInformationAsJsonb  string
	testInstructionsAsJsonb          string
	testInstructionContainersAsJsonb string
	uniqueCounter                    int
}

// Stores a slice of attributes to be stored in Cloud-DB
type tempAttributesType []tempAttributeStruct
type tempAttributeStruct struct {
	testCaseExecutionUuid            string
	testInstructionName              string
	testInstructionExecutionUuid     string
	testInstructionAttributeType     int
	TestInstructionAttributeUuid     string
	TestInstructionAttributeName     string
	AttributeValueAsString           string
	AttributeValueUuid               string
	testInstructionAttributeTypeUuid string
	testInstructionAttributeTypeName string
	testInstructionExecutionVersion  int
}

// Load TestCaseExecutionQueue-Messages be able to populate the ongoing TestCaseExecutionUuid-table
func (executionEngine *TestInstructionExecutionEngineStruct) loadTestCaseExecutionQueueMessages(
	dbTransaction pgx.Tx,
	testCaseExecutionsToProcess []ChannelCommandTestCaseExecutionStruct) (
	testCaseExecutionQueueMessages []*tempTestCaseExecutionQueueInformationStruct, err error) {

	usedDBSchema := "FenixExecution" // TODO should this env variable be used? fenixSyncShared.GetDBSchemaName()

	// Generate WHERE-values to only target correct 'TestCaseExecutionUuid' together with 'TestCaseExecutionVersion'
	var correctTestCaseExecutionUuidAndTestCaseExecutionVersionPar string
	var correctTestCaseExecutionUuidAndTestCaseExecutionVersionPars string
	for testCaseExecutionCounter, testCaseExecution := range testCaseExecutionsToProcess {
		correctTestCaseExecutionUuidAndTestCaseExecutionVersionPar =
			"(TCEQ.\"TestCaseExecutionUuid\" = '" + testCaseExecution.TestCaseExecutionUuid + "' AND " +
				"TCEQ.\"TestCaseExecutionVersion\" = " + strconv.Itoa(int(testCaseExecution.TestCaseExecutionVersion)) + ") "

		switch testCaseExecutionCounter {
		case 0:
			// When this is the first then we need to add 'AND before'
			// *NOT NEEDED* in this Query
			//correctTestCaseExecutionUuidAndTestCaseExecutionVersionPars = "AND "

		default:
			// When this is not the first then we need to add 'OR' after previous
			correctTestCaseExecutionUuidAndTestCaseExecutionVersionPars =
				correctTestCaseExecutionUuidAndTestCaseExecutionVersionPars + "OR "
		}

		// Add the WHERE-values
		correctTestCaseExecutionUuidAndTestCaseExecutionVersionPars =
			correctTestCaseExecutionUuidAndTestCaseExecutionVersionPars + correctTestCaseExecutionUuidAndTestCaseExecutionVersionPar

	}

	sqlToExecute := ""
	sqlToExecute = sqlToExecute + "SELECT TCEQ.* "
	sqlToExecute = sqlToExecute + "FROM \"" + usedDBSchema + "\".\"TestCaseExecutionQueue\" TCEQ "
	sqlToExecute = sqlToExecute + "WHERE "
	sqlToExecute = sqlToExecute + correctTestCaseExecutionUuidAndTestCaseExecutionVersionPars
	sqlToExecute = sqlToExecute + "ORDER BY TCEQ.\"QueueTimeStamp\" ASC; "

	// Query DB
	var ctx context.Context
	ctx, timeOutCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer timeOutCancel()

	rows, err := dbTransaction.Query(ctx, sqlToExecute)
	defer rows.Close()

	if err != nil {
		common_config.Logger.WithFields(logrus.Fields{
			"Id":           "85459587-0c1e-4db9-b257-742ff3a660fc",
			"Error":        err,
			"sqlToExecute": sqlToExecute,
		}).Error("Something went wrong when executing SQL")

		return []*tempTestCaseExecutionQueueInformationStruct{}, err
	}

	// USed to secure that exactly one row was found
	numberOfRowFromDB := 0

	// Extract data from DB result set
	for rows.Next() {

		var testCaseExecutionQueueMessage tempTestCaseExecutionQueueInformationStruct

		numberOfRowFromDB = numberOfRowFromDB + 1

		err := rows.Scan(
			&testCaseExecutionQueueMessage.domainUuid,
			&testCaseExecutionQueueMessage.domainName,
			&testCaseExecutionQueueMessage.testSuiteUuid,
			&testCaseExecutionQueueMessage.testSuiteName,
			&testCaseExecutionQueueMessage.testSuiteVersion,
			&testCaseExecutionQueueMessage.testSuiteExecutionUuid,
			&testCaseExecutionQueueMessage.testSuiteExecutionVersion,
			&testCaseExecutionQueueMessage.testCaseUuid,
			&testCaseExecutionQueueMessage.testCaseName,
			&testCaseExecutionQueueMessage.testCaseVersion,
			&testCaseExecutionQueueMessage.testCaseExecutionUuid,
			&testCaseExecutionQueueMessage.testCaseExecutionVersion,
			&testCaseExecutionQueueMessage.queueTimeStamp,
			&testCaseExecutionQueueMessage.testDataSetUuid,
			&testCaseExecutionQueueMessage.executionPriority,
			&testCaseExecutionQueueMessage.uniqueCounter,

			// ReportingLevel
			&testCaseExecutionQueueMessage.executionStatusReportLevel,
		)

		if err != nil {

			common_config.Logger.WithFields(logrus.Fields{
				"Id":           "6ec31a99-d2d9-4ecd-b0ee-2e9a05df336e",
				"Error":        err,
				"sqlToExecute": sqlToExecute,
			}).Error("Something went wrong when processing result from database")

			return []*tempTestCaseExecutionQueueInformationStruct{}, err
		}

		// Add Queue-message to slice of messages
		testCaseExecutionQueueMessages = append(testCaseExecutionQueueMessages, &testCaseExecutionQueueMessage)

	}

	return testCaseExecutionQueueMessages, err

}

// Put all messages found on TestCaseExecutionQueue to the ongoing executions table
func (executionEngine *TestInstructionExecutionEngineStruct) saveTestCasesOnOngoingExecutionsQueueSaveToCloudDB(
	dbTransaction pgx.Tx,
	testCaseExecutionQueueMessages []*tempTestCaseExecutionQueueInformationStruct) (
	err error) {

	common_config.Logger.WithFields(logrus.Fields{
		"Id": "8e857aa4-3f15-4415-bc08-5ac97bf64446",
	}).Debug("Entering: saveTestCasesOnOngoingExecutionsQueueSaveToCloudDB()")

	defer func() {
		common_config.Logger.WithFields(logrus.Fields{
			"Id": "8dd6bc1b-361b-4f82-83a8-dbe49114649b",
		}).Debug("Exiting: saveTestCasesOnOngoingExecutionsQueueSaveToCloudDB()")
	}()

	// Get a common dateTimeStamp to use
	currentDataTimeStamp := fenixSyncShared.GenerateDatetimeTimeStampForDB()

	var dataRowToBeInsertedMultiType []interface{}
	var dataRowsToBeInsertedMultiType [][]interface{}

	usedDBSchema := "FenixExecution" // TODO should this env variable be used? fenixSyncShared.GetDBSchemaName()

	sqlToExecute := ""

	// Create Insert Statement for TestCaseExecutionUuid that will be put on ExecutionQueue
	// Data to be inserted in the DB-table
	dataRowsToBeInsertedMultiType = nil

	for _, testCaseExecutionQueueMessage := range testCaseExecutionQueueMessages {

		dataRowToBeInsertedMultiType = nil

		dataRowToBeInsertedMultiType = append(dataRowToBeInsertedMultiType, testCaseExecutionQueueMessage.domainUuid)
		dataRowToBeInsertedMultiType = append(dataRowToBeInsertedMultiType, testCaseExecutionQueueMessage.domainName)

		dataRowToBeInsertedMultiType = append(dataRowToBeInsertedMultiType, testCaseExecutionQueueMessage.testSuiteUuid)
		dataRowToBeInsertedMultiType = append(dataRowToBeInsertedMultiType, testCaseExecutionQueueMessage.testSuiteName)
		dataRowToBeInsertedMultiType = append(dataRowToBeInsertedMultiType, testCaseExecutionQueueMessage.testSuiteVersion)
		dataRowToBeInsertedMultiType = append(dataRowToBeInsertedMultiType, testCaseExecutionQueueMessage.testSuiteExecutionUuid)
		dataRowToBeInsertedMultiType = append(dataRowToBeInsertedMultiType, testCaseExecutionQueueMessage.testSuiteExecutionVersion)

		dataRowToBeInsertedMultiType = append(dataRowToBeInsertedMultiType, testCaseExecutionQueueMessage.testCaseUuid)
		dataRowToBeInsertedMultiType = append(dataRowToBeInsertedMultiType, testCaseExecutionQueueMessage.testCaseName)
		dataRowToBeInsertedMultiType = append(dataRowToBeInsertedMultiType, testCaseExecutionQueueMessage.testCaseVersion)
		dataRowToBeInsertedMultiType = append(dataRowToBeInsertedMultiType, testCaseExecutionQueueMessage.testCaseExecutionUuid)
		dataRowToBeInsertedMultiType = append(dataRowToBeInsertedMultiType, testCaseExecutionQueueMessage.testCaseExecutionVersion)
		dataRowToBeInsertedMultiType = append(dataRowToBeInsertedMultiType, testCaseExecutionQueueMessage.queueTimeStamp)
		dataRowToBeInsertedMultiType = append(dataRowToBeInsertedMultiType, testCaseExecutionQueueMessage.testDataSetUuid)
		dataRowToBeInsertedMultiType = append(dataRowToBeInsertedMultiType, testCaseExecutionQueueMessage.executionPriority)

		dataRowToBeInsertedMultiType = append(dataRowToBeInsertedMultiType, currentDataTimeStamp)
		dataRowToBeInsertedMultiType = append(dataRowToBeInsertedMultiType, currentDataTimeStamp)
		dataRowToBeInsertedMultiType = append(dataRowToBeInsertedMultiType, int(fenixExecutionServerGrpcApi.TestCaseExecutionStatusEnum_TCE_INITIATED))
		dataRowToBeInsertedMultiType = append(dataRowToBeInsertedMultiType, false)
		dataRowToBeInsertedMultiType = append(dataRowToBeInsertedMultiType, currentDataTimeStamp)

		dataRowToBeInsertedMultiType = append(dataRowToBeInsertedMultiType, testCaseExecutionQueueMessage.executionStatusReportLevel)

		dataRowsToBeInsertedMultiType = append(dataRowsToBeInsertedMultiType, dataRowToBeInsertedMultiType)

	}

	sqlToExecute = sqlToExecute + "INSERT INTO \"" + usedDBSchema + "\".\"TestCasesUnderExecution\" "
	sqlToExecute = sqlToExecute + "(\"DomainUuid\", \"DomainName\", \"TestSuiteUuid\", \"TestSuiteName\", \"TestSuiteVersion\", " +
		"\"TestSuiteExecutionUuid\", \"TestSuiteExecutionVersion\", \"TestCaseUuid\", \"TestCaseName\", \"TestCaseVersion\"," +
		" \"TestCaseExecutionUuid\", \"TestCaseExecutionVersion\", \"QueueTimeStamp\", \"TestDataSetUuid\", \"ExecutionPriority\", " +
		"\"ExecutionStartTimeStamp\", \"ExecutionStopTimeStamp\", \"TestCaseExecutionStatus\", \"ExecutionHasFinished\", " +
		"\"ExecutionStatusUpdateTimeStamp\", \"ExecutionStatusReportLevel\")  "
	sqlToExecute = sqlToExecute + common_config.GenerateSQLInsertValues(dataRowsToBeInsertedMultiType)
	sqlToExecute = sqlToExecute + ";"

	// Log SQL to be executed if Environment variable is true
	if common_config.LogAllSQLs == true {
		common_config.Logger.WithFields(logrus.Fields{
			"Id":           "65e7793b-a5c7-4fa1-8506-5ea9e6a8d7af",
			"sqlToExecute": sqlToExecute,
		}).Debug("SQL to be executed within 'saveTestCasesOnOngoingExecutionsQueueSaveToCloudDB'")
	}

	// Execute Query CloudDB
	comandTag, err := dbTransaction.Exec(context.Background(), sqlToExecute)

	if err != nil {
		common_config.Logger.WithFields(logrus.Fields{
			"Id":           "c7f30078-20ff-4a34-a066-fa49fa2ca475",
			"Error":        err,
			"sqlToExecute": sqlToExecute,
		}).Error("Something went wrong when executing SQL")

		return err
	}

	// Log response from CloudDB
	common_config.Logger.WithFields(logrus.Fields{
		"Id":                       "dcb110c2-822a-4dde-8bc6-9ebbe9fcbdb0",
		"comandTag.Insert()":       comandTag.Insert(),
		"comandTag.Delete()":       comandTag.Delete(),
		"comandTag.Select()":       comandTag.Select(),
		"comandTag.Update()":       comandTag.Update(),
		"comandTag.RowsAffected()": comandTag.RowsAffected(),
		"comandTag.String()":       comandTag.String(),
	}).Debug("Return data for SQL executed in database")

	// No errors occurred
	return nil

}

// Save a 'copy' of TestCaseExecution in Table 'TestCasesExecutionsForListings'
func (executionEngine *TestInstructionExecutionEngineStruct) saveTestCasesExecutionToTestCasesExecutionsForListingsSaveToCloudDB(
	dbTransaction pgx.Tx,
	testCaseExecutionQueueMessages []*tempTestCaseExecutionQueueInformationStruct) (
	err error) {

	common_config.Logger.WithFields(logrus.Fields{
		"Id": "761b3aec-23c4-4a12-9a4b-0aa55b33362b",
	}).Debug("Entering: saveTestCasesExecutionToTestCasesExecutionsForListingsSaveToCloudDB()")

	defer func() {
		common_config.Logger.WithFields(logrus.Fields{
			"Id": "5392ea69-7cc6-4fa5-8a22-8ed476101093",
		}).Debug("Exiting: saveTestCasesExecutionToTestCasesExecutionsForListingsSaveToCloudDB()")
	}()

	sqlToExecute := ""
	var testCaseExecutions string

	// No Message should give an error
	if len(testCaseExecutionQueueMessages) == 0 {
		common_config.Logger.WithFields(logrus.Fields{
			"Id": "86d63010-7b02-47b1-a3f0-7846894395f6",
		}).Debug("No transactions to process in 'saveTestCasesExecutionToTestCasesExecutionsForListingsSaveToCloudDB'. Shouldn't be like that!")

		errorId := "d91ea8e0-973c-42d8-9446-a9ddbf8024ef"

		err = errors.New(fmt.Sprintf("no transactions to process in 'saveTestCasesExecutionToTestCasesExecutionsForListingsSaveToCloudDB'. Shouldn't be like that. [ErrorId: %s]", errorId))

		return err
	}

	for _, testCaseExecutionQueueMessage := range testCaseExecutionQueueMessages {

		if len(testCaseExecutions) == 0 {
			// First TestCaseExecution
			testCaseExecutions = fmt.Sprintf("\"TestCaseExecutionUuid\" = '%s' AND \"TestCaseExecutionVersion\" = %d",
				testCaseExecutionQueueMessage.testCaseExecutionUuid,
				testCaseExecutionQueueMessage.testCaseExecutionVersion)

		} else {
			// There are already TestCaseExecutions
			testCaseExecutions = testCaseExecutions + " OR "

			testCaseExecutions = testCaseExecutions + fmt.Sprintf("\"TestCaseExecutionUuid\" = '%s' AND \"TestCaseExecutionVersion\" = %d",
				testCaseExecutionQueueMessage.testCaseExecutionUuid,
				testCaseExecutionQueueMessage.testCaseExecutionVersion)

		}
	}

	sqlToExecute = sqlToExecute + "INSERT INTO \"FenixExecution\".\"TestCasesExecutionsForListings\" "
	sqlToExecute = sqlToExecute + "SELECT * FROM \"FenixExecution\".\"TestCasesUnderExecution\" "
	sqlToExecute = sqlToExecute + "WHERE " + testCaseExecutions
	sqlToExecute = sqlToExecute + " "
	sqlToExecute = sqlToExecute + ";"

	// Log SQL to be executed if Environment variable is true
	if common_config.LogAllSQLs == true {
		common_config.Logger.WithFields(logrus.Fields{
			"Id":           "00701aa2-1617-47fe-8749-5ba1779b8d34",
			"sqlToExecute": sqlToExecute,
		}).Debug("SQL to be executed within 'saveTestCasesExecutionToTestCasesExecutionsForListingsSaveToCloudDB'")
	}

	// Execute Query CloudDB
	comandTag, err := dbTransaction.Exec(context.Background(), sqlToExecute)

	if err != nil {
		common_config.Logger.WithFields(logrus.Fields{
			"Id":           "e734c819-b953-4ba6-aecb-ec38a4d0a7fb",
			"Error":        err,
			"sqlToExecute": sqlToExecute,
		}).Error("Something went wrong when executing SQL")

		return err
	}

	// Log response from CloudDB
	common_config.Logger.WithFields(logrus.Fields{
		"Id":                       "3beacafe-90e7-4588-a22a-e27190964a6b",
		"comandTag.Insert()":       comandTag.Insert(),
		"comandTag.Delete()":       comandTag.Delete(),
		"comandTag.Select()":       comandTag.Select(),
		"comandTag.Update()":       comandTag.Update(),
		"comandTag.RowsAffected()": comandTag.RowsAffected(),
		"comandTag.String()":       comandTag.String(),
	}).Debug("Return data for SQL executed in database")

	// No errors occurred
	return nil

}

// Save a TestSuiteExecution in Table 'TestSuitesExecutionsForListings'
func (executionEngine *TestInstructionExecutionEngineStruct) saveTestSuiteExecutionToTestSuitesExecutionsForListingsSaveToCloudDB(
	dbTransaction pgx.Tx,
	testCaseExecutionQueueMessages []*tempTestCaseExecutionQueueInformationStruct) (
	testCaseUsedForTestSuiteExecutionQueueMessages []*tempTestCaseExecutionQueueInformationStruct,
	err error) {

	common_config.Logger.WithFields(logrus.Fields{
		"Id": "6326006a-9ec0-4680-8cca-af0d8e4caf4f",
	}).Debug("Entering: saveTestSuiteExecutionToTestSuitesExecutionsForListingsSaveToCloudDB()")

	defer func() {
		common_config.Logger.WithFields(logrus.Fields{
			"Id": "9d20c8b3-01fc-4715-8141-64ab6c78e99b",
		}).Debug("Exiting: saveTestSuiteExecutionToTestSuitesExecutionsForListingsSaveToCloudDB()")
	}()

	sqlToExecute := ""
	var testCaseExecutions string

	// No Message should give an error
	if len(testCaseExecutionQueueMessages) == 0 {
		common_config.Logger.WithFields(logrus.Fields{
			"Id": "a85d81ed-fe26-439b-8397-6f8edf01ce73",
		}).Debug("No transactions to process in 'saveTestSuiteExecutionToTestSuitesExecutionsForListingsSaveToCloudDB'. Shouldn't be like that!")

		errorId := "ee37bb19-9c16-4dd5-8c53-3e1a087c7add"

		err = errors.New(fmt.Sprintf("no transactions to process in 'saveTestSuiteExecutionToTestSuitesExecutionsForListingsSaveToCloudDB'. Shouldn't be like that. [ErrorId: %s]", errorId))

		return nil, err
	}

	// Map for keep track of used TestSuiteExecutionUuid's
	var usedTestSuiteExecutionUuidsMap map[string]bool
	usedTestSuiteExecutionUuidsMap = make(map[string]bool)
	var usedTestSuiteExecutionUuidKey string
	var existInMap bool

	for _, testCaseExecutionQueueMessage := range testCaseExecutionQueueMessages {

		// Add Used TestSuiteExecutionUuid to map if not already there
		usedTestSuiteExecutionUuidKey = testCaseExecutionQueueMessage.testSuiteExecutionUuid + strconv.
			Itoa(testCaseExecutionQueueMessage.testSuiteExecutionVersion)

		// Check if TestSuiteExecutionUuid is used, if so then go to next
		_, existInMap = usedTestSuiteExecutionUuidsMap[usedTestSuiteExecutionUuidKey]
		if existInMap == true {
			continue
		}

		// Add the TestSuiteExecutionUuid to the map
		usedTestSuiteExecutionUuidsMap[usedTestSuiteExecutionUuidKey] = true

		if len(testCaseExecutions) == 0 {
			// First TestCaseExecution
			testCaseExecutions = fmt.Sprintf("\"TestCaseExecutionUuid\" = '%s' AND \"TestCaseExecutionVersion\" = %d",
				testCaseExecutionQueueMessage.testCaseExecutionUuid,
				testCaseExecutionQueueMessage.testCaseExecutionVersion)

		} else {
			// There are already TestCaseExecutions
			testCaseExecutions = testCaseExecutions + " OR "

			testCaseExecutions = testCaseExecutions + fmt.Sprintf("\"TestCaseExecutionUuid\" = '%s' AND \"TestCaseExecutionVersion\" = %d",
				testCaseExecutionQueueMessage.testCaseExecutionUuid,
				testCaseExecutionQueueMessage.testCaseExecutionVersion)

		}

		// Add TestCase to 'testCaseUsedForTestSuiteExecutionQueueMessages'
		testCaseUsedForTestSuiteExecutionQueueMessages = append(testCaseUsedForTestSuiteExecutionQueueMessages, testCaseExecutionQueueMessage)
	}

	sqlToExecute = sqlToExecute + "INSERT INTO \"FenixExecution\".\"TestSuitesExecutionsForListings\" "
	sqlToExecute = sqlToExecute + "SELECT * FROM \"FenixExecution\".\"TestCasesUnderExecution\" "
	sqlToExecute = sqlToExecute + "WHERE " + testCaseExecutions
	sqlToExecute = sqlToExecute + " "
	sqlToExecute = sqlToExecute + ";"

	// Log SQL to be executed if Environment variable is true
	if common_config.LogAllSQLs == true {
		common_config.Logger.WithFields(logrus.Fields{
			"Id":           "47af69ef-e70f-40e3-9c97-781cf6824993",
			"sqlToExecute": sqlToExecute,
		}).Debug("SQL to be executed within 'saveTestSuiteExecutionToTestSuitesExecutionsForListingsSaveToCloudDB'")
	}

	// Execute Query CloudDB
	comandTag, err := dbTransaction.Exec(context.Background(), sqlToExecute)

	if err != nil {
		common_config.Logger.WithFields(logrus.Fields{
			"Id":           "56986a35-e1c2-4a93-b21d-988af30fbfe9",
			"Error":        err,
			"sqlToExecute": sqlToExecute,
		}).Error("Something went wrong when executing SQL")

		return nil, err
	}

	// Log response from CloudDB
	common_config.Logger.WithFields(logrus.Fields{
		"Id":                       "1c5831f6-713d-4c5e-9b0b-400be5670f03",
		"comandTag.Insert()":       comandTag.Insert(),
		"comandTag.Delete()":       comandTag.Delete(),
		"comandTag.Select()":       comandTag.Select(),
		"comandTag.Update()":       comandTag.Update(),
		"comandTag.RowsAffected()": comandTag.RowsAffected(),
		"comandTag.String()":       comandTag.String(),
	}).Debug("Return data for SQL executed in database")

	// No errors occurred
	return testCaseUsedForTestSuiteExecutionQueueMessages, nil

}

// Retrieve "TestCasePreview"
func (executionEngine *TestInstructionExecutionEngineStruct) loadTestCasePreview(
	dbTransaction pgx.Tx,
	tempTestCaseExecutionQueueMessage *tempTestCaseExecutionQueueInformationStruct) (
	tempTestCasePreview *fenixTestCaseBuilderServerGrpcApi.TestCasePreviewMessage,
	err error) {

	// Load 'TestCasePreview'

	sqlToExecute := ""
	sqlToExecute = sqlToExecute + "SELECT TC.\"TestCasePreview\" "
	sqlToExecute = sqlToExecute + "FROM \"FenixBuilder\".\"TestCases\" TC "
	sqlToExecute = sqlToExecute + "WHERE "
	sqlToExecute = sqlToExecute + fmt.Sprintf("\"TestCaseUuid\" = '%s' AND \"TestCaseVersion\" = %d ",
		tempTestCaseExecutionQueueMessage.testCaseUuid,
		tempTestCaseExecutionQueueMessage.testCaseVersion)
	sqlToExecute = sqlToExecute + ";"

	// Query DB
	var ctx context.Context
	ctx, timeOutCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer timeOutCancel()

	rows, err := dbTransaction.Query(ctx, sqlToExecute)
	defer rows.Close()

	if err != nil {
		common_config.Logger.WithFields(logrus.Fields{
			"Id":           "5334c148-7d63-4333-b6b5-eeb403ae0e2d",
			"Error":        err,
			"sqlToExecute": sqlToExecute,
		}).Error("Something went wrong when executing SQL")

		return nil, err
	}

	// USed to secure that exactly one row was found
	numberOfRowFromDB := 0

	var testCasePreviewAsString string
	var testCasePreviewAsByteArray []byte

	// Extract data from DB result set
	for rows.Next() {

		numberOfRowFromDB = numberOfRowFromDB + 1

		err := rows.Scan(
			&testCasePreviewAsString,
		)

		if err != nil {

			common_config.Logger.WithFields(logrus.Fields{
				"Id":           "7cbefcf9-b8ce-403f-b340-bbf8112cc646",
				"Error":        err,
				"sqlToExecute": sqlToExecute,
			}).Error("Something went wrong when processing result from database")

			return nil, err
		}
	}

	// There should always be exact one row
	if numberOfRowFromDB != 1 {

		common_config.Logger.WithFields(logrus.Fields{
			"Id":                "b0887097-44ab-4132-9049-04dd2c566890",
			"numberOfRowFromDB": numberOfRowFromDB,
			"sqlToExecute":      sqlToExecute,
		}).Error("There should always be exact one rows")

		errorId := "24dd7924-0ff2-4899-b1b6-8e92ec5eb903"

		err = errors.New(fmt.Sprintf("there should always be exact one rows. [ErrorId: %s]", errorId))

		return nil, err

	}

	// Convert json-string into byte-array
	testCasePreviewAsByteArray = []byte(testCasePreviewAsString)

	// Convert json-byte-arrays into proto-messages
	var testCasePreview fenixTestCaseBuilderServerGrpcApi.TestCasePreviewMessage
	err = protojson.Unmarshal(testCasePreviewAsByteArray, &testCasePreview)
	if err != nil {
		common_config.Logger.WithFields(logrus.Fields{
			"Id":    "84ceea8b-b24d-42c6-a45f-9fa9128a54b3",
			"Error": err,
		}).Error("Something went wrong when converting 'testCasePreviewAsByteArray' into proto-message")

		return nil, err
	}

	return &testCasePreview, err
}

// Add "TestCasePreview" 'TestCasesExecutionsForListings'
func (executionEngine *TestInstructionExecutionEngineStruct) addTestCasePreviewIntoDatabase(
	dbTransaction pgx.Tx,
	testCaseExecutionQueueMessage *tempTestCaseExecutionQueueInformationStruct,
	testCasePreview *fenixTestCaseBuilderServerGrpcApi.TestCasePreviewMessage) (
	err error) {

	// If there are nothing to update then just exit
	if testCasePreview == nil {
		return nil
	}

	testCasePreviewAsJsonb := protojson.Format(testCasePreview)

	// Create Update Statement  TestCasesExecutionsForListings
	sqlToExecute := ""
	sqlToExecute = sqlToExecute + "UPDATE \"FenixExecution\".\"TestCasesExecutionsForListings\" "
	sqlToExecute = sqlToExecute + fmt.Sprintf("SET ")
	sqlToExecute = sqlToExecute + fmt.Sprintf("\"TestCasePreview\" = '%s' ",
		testCasePreviewAsJsonb)
	sqlToExecute = sqlToExecute + fmt.Sprintf("WHERE \"TestCaseExecutionUuid\" = '%s' ",
		testCaseExecutionQueueMessage.testCaseExecutionUuid)
	sqlToExecute = sqlToExecute + fmt.Sprintf("AND ")
	sqlToExecute = sqlToExecute + fmt.Sprintf("\"TestCaseExecutionVersion\" = %d ",
		testCaseExecutionQueueMessage.testCaseExecutionVersion)
	sqlToExecute = sqlToExecute + "; "

	// Log SQL to be executed if Environment variable is true
	if common_config.LogAllSQLs == true {
		common_config.Logger.WithFields(logrus.Fields{
			"Id":           "97ec8cb4-4ce8-40dc-873e-4c506f01f2a4",
			"sqlToExecute": sqlToExecute,
		}).Debug("SQL to be executed within 'addTestCasePreviewIntoDatabase'")
	}

	// Execute Query CloudDB
	comandTag, err := dbTransaction.Exec(context.Background(), sqlToExecute)

	if err != nil {
		common_config.Logger.WithFields(logrus.Fields{
			"Id":           "fa9cadc9-d801-4383-b1b9-e42e4637b961",
			"sqlToExecute": sqlToExecute,
		}).Error("Something went wrong when executing SQL")

		return err
	}

	// Log response from CloudDB
	common_config.Logger.WithFields(logrus.Fields{
		"Id":                       "3497a29b-852d-4477-9e07-aed1ddd13a9e",
		"comandTag.Insert()":       comandTag.Insert(),
		"comandTag.Delete()":       comandTag.Delete(),
		"comandTag.Select()":       comandTag.Select(),
		"comandTag.Update()":       comandTag.Update(),
		"comandTag.RowsAffected()": comandTag.RowsAffected(),
		"comandTag.String()":       comandTag.String(),
	}).Debug("Return data for SQL executed in database")

	// If No(zero) rows were affected then TestInstructionExecutionUuid is missing in Table
	if comandTag.RowsAffected() != 1 {
		errorId := "b465295f-ebff-479f-8a13-42994212be7d"
		err = errors.New(fmt.Sprintf("TestCaseExecutionUuid '%s' with TestCaseExecutionVersion '%d' is missing in Table: 'TestCasesExecutionsForListings' [ErroId: %s]",
			testCaseExecutionQueueMessage.testCaseExecutionUuid,
			testCaseExecutionQueueMessage.testCaseExecutionVersion,
			errorId))

		common_config.Logger.WithFields(logrus.Fields{
			"Id":                       "d2e72a4a-abd6-4123-9d24-2c95847d149d",
			"comandTag.RowsAffected()": comandTag.RowsAffected(),
		}).Error(err.Error())

		return err
	}

	// No errors occurred
	return err

}

// Add "TestSuitePreview" to 'TestSuitesExecutionsForListings'
func (executionEngine *TestInstructionExecutionEngineStruct) addTestSuitePreviewIntoDatabase(
	dbTransaction pgx.Tx,
	testSuitesPreview []*fenixTestCaseBuilderServerGrpcApi.TestSuitePreviewMessage,
	testCaseUsedForTestSuiteExecutionQueueMessages []*tempTestCaseExecutionQueueInformationStruct) (
	err error) {

	// If there are nothing to update then just exit
	if testSuitesPreview == nil {
		return nil
	}

	// Create a TestSuitePreViewMap, key TestSuiteUuid
	var testSuitePreviewMap map[string]*fenixTestCaseBuilderServerGrpcApi.TestSuitePreviewMessage
	testSuitePreviewMap = make(map[string]*fenixTestCaseBuilderServerGrpcApi.TestSuitePreviewMessage)
	for _, testSuitePreview := range testSuitesPreview {
		testSuitePreviewMap[testSuitePreview.TestSuitePreview.TestSuiteUuid] = testSuitePreview
	}

	// Create a reference between TestSuiteUuid and all possible TestSuiteExecutionUuid
	var testSuiteUuidToTestSuiteExecutionsUuidMap map[string][]*tempTestCaseExecutionQueueInformationStruct
	var existInMap bool
	testSuiteUuidToTestSuiteExecutionsUuidMap = make(map[string][]*tempTestCaseExecutionQueueInformationStruct)
	for _, testCaseUsedForTestSuiteExecutionQueueMessage := range testCaseUsedForTestSuiteExecutionQueueMessages {

		_, existInMap = testSuiteUuidToTestSuiteExecutionsUuidMap[testCaseUsedForTestSuiteExecutionQueueMessage.testSuiteUuid]
		if existInMap == false {
			testSuiteUuidToTestSuiteExecutionsUuidMap[testCaseUsedForTestSuiteExecutionQueueMessage.testSuiteUuid] = make([]*tempTestCaseExecutionQueueInformationStruct, 0)
		}

		testSuiteUuidToTestSuiteExecutionsUuidMap[testCaseUsedForTestSuiteExecutionQueueMessage.testSuiteUuid] = append(
			testSuiteUuidToTestSuiteExecutionsUuidMap[testCaseUsedForTestSuiteExecutionQueueMessage.testSuiteUuid],
			testCaseUsedForTestSuiteExecutionQueueMessage)

	}

	// Loop all TestSuitePreview and process SQL towards database
	for _, tempTestCaseExecutionQueueInformationMessage := range testSuiteUuidToTestSuiteExecutionsUuidMap {
		for _, tempTestCaseExecutionQueueInformation := range tempTestCaseExecutionQueueInformationMessage {
			testSuitePreviewAsJsonb := protojson.Format(testSuitePreviewMap[tempTestCaseExecutionQueueInformation.testSuiteUuid])

			// Create Update Statement  TestCasesExecutionsForListings
			sqlToExecute := ""
			sqlToExecute = sqlToExecute + "UPDATE \"FenixExecution\".\"TestSuitesExecutionsForListings\" "
			sqlToExecute = sqlToExecute + fmt.Sprintf("SET ")
			sqlToExecute = sqlToExecute + fmt.Sprintf("\"TestSuitePreview\" = '%s' ",
				testSuitePreviewAsJsonb)
			sqlToExecute = sqlToExecute + fmt.Sprintf("WHERE \"TestSuiteExecutionUuid\" = '%s' ",
				tempTestCaseExecutionQueueInformation.testSuiteExecutionUuid)
			sqlToExecute = sqlToExecute + fmt.Sprintf("AND ")
			sqlToExecute = sqlToExecute + fmt.Sprintf("\"TestSuiteExecutionVersion\" = %d ",
				tempTestCaseExecutionQueueInformation.testSuiteExecutionVersion)
			sqlToExecute = sqlToExecute + "; "

			// Log SQL to be executed if Environment variable is true
			if common_config.LogAllSQLs == true {
				common_config.Logger.WithFields(logrus.Fields{
					"Id":           "af0b852a-5b5a-4b70-aaf4-a6515fe29817",
					"sqlToExecute": sqlToExecute,
				}).Debug("SQL to be executed within 'addTestSuitePreviewIntoDatabase'")
			}

			// Execute Query CloudDB
			comandTag, err := dbTransaction.Exec(context.Background(), sqlToExecute)

			if err != nil {
				common_config.Logger.WithFields(logrus.Fields{
					"Id":           "a6203bc4-3c75-4b42-9cd5-bbc4c52ddf79",
					"sqlToExecute": sqlToExecute,
				}).Error("Something went wrong when executing SQL")

				return err
			}

			// Log response from CloudDB
			common_config.Logger.WithFields(logrus.Fields{
				"Id":                       "02f93b43-c9e6-427d-9b40-3a297a733f85",
				"comandTag.Insert()":       comandTag.Insert(),
				"comandTag.Delete()":       comandTag.Delete(),
				"comandTag.Select()":       comandTag.Select(),
				"comandTag.Update()":       comandTag.Update(),
				"comandTag.RowsAffected()": comandTag.RowsAffected(),
				"comandTag.String()":       comandTag.String(),
			}).Debug("Return data for SQL executed in database")

			// If No(zero) rows were affected then TestInstructionExecutionUuid is missing in Table
			if comandTag.RowsAffected() != 1 {
				errorId := "a127c111-75f5-4e67-bb73-14de93e94cf8"
				err = errors.New(fmt.Sprintf("TestSuiteExecutionUuid '%s' with TestSuiteExecutionVersion '%d' is missing in Table: 'TestSuitesExecutionsForListings' [ErroId: %s]",
					tempTestCaseExecutionQueueInformation.testCaseExecutionUuid,
					tempTestCaseExecutionQueueInformation.testCaseExecutionVersion,
					errorId))

				common_config.Logger.WithFields(logrus.Fields{
					"Id":                       "32ee64d3-77ed-4919-a82a-782c68a219a0",
					"comandTag.RowsAffected()": comandTag.RowsAffected(),
				}).Error(err.Error())

				return err
			}
		}
	}

	// No errors occurred
	return err

}

// Clear all messages found on TestCaseExecutionQueue that were put on table for the ongoing executions
func (executionEngine *TestInstructionExecutionEngineStruct) clearTestCasesExecutionQueueSaveToCloudDB(dbTransaction pgx.Tx, testCaseExecutionQueueMessages []*tempTestCaseExecutionQueueInformationStruct) (err error) {

	common_config.Logger.WithFields(logrus.Fields{
		"Id":                             "7703b634-a46d-4494-897f-1f139b5858c5",
		"testCaseExecutionQueueMessages": testCaseExecutionQueueMessages,
	}).Debug("Entering: clearTestCasesExecutionQueueSaveToCloudDB()")

	defer func() {
		common_config.Logger.WithFields(logrus.Fields{
			"Id": "fed261d1-3757-46f7-bc10-476e045606a2",
		}).Debug("Exiting: clearTestCasesExecutionQueueSaveToCloudDB()")
	}()

	var testCaseExecutionsToBeDeletedFromQueue []int

	// Loop over TestCaseExecutionQueue-messages and extract  "UniqueCounter"
	for _, testCaseExecutionQueueMessage := range testCaseExecutionQueueMessages {
		testCaseExecutionsToBeDeletedFromQueue = append(testCaseExecutionsToBeDeletedFromQueue, testCaseExecutionQueueMessage.uniqueCounter)
	}

	usedDBSchema := "FenixExecution" // TODO should this env variable be used? fenixSyncShared.GetDBSchemaName()

	sqlToExecute := ""

	sqlToExecute = sqlToExecute + "DELETE FROM \"" + usedDBSchema + "\".\"TestCaseExecutionQueue\" TCEQ "
	sqlToExecute = sqlToExecute + "WHERE TCEQ.\"UniqueCounter\" IN "
	sqlToExecute = sqlToExecute + common_config.GenerateSQLINIntegerArray(testCaseExecutionsToBeDeletedFromQueue)
	sqlToExecute = sqlToExecute + ";"

	// Log SQL to be executed if Environment variable is true
	if common_config.LogAllSQLs == true {
		common_config.Logger.WithFields(logrus.Fields{
			"Id":           "3050217d-45aa-4d64-be44-61bd4cf9e165",
			"sqlToExecute": sqlToExecute,
		}).Debug("SQL to be executed within 'clearTestCasesExecutionQueueSaveToCloudDB'")
	}

	// Execute Query CloudDB
	comandTag, err := dbTransaction.Exec(context.Background(), sqlToExecute)

	if err != nil {
		common_config.Logger.WithFields(logrus.Fields{
			"Id":           "38a5ca13-c108-427a-a24a-20c3b6d6c4be",
			"Error":        err,
			"sqlToExecute": sqlToExecute,
		}).Error("Something went wrong when executing SQL")

		return err
	}

	// Log response from CloudDB
	common_config.Logger.WithFields(logrus.Fields{
		"Id":                       "dcb110c2-822a-4dde-8bc6-9ebbe9fcbdb0",
		"comandTag.Insert()":       comandTag.Insert(),
		"comandTag.Delete()":       comandTag.Delete(),
		"comandTag.Select()":       comandTag.Select(),
		"comandTag.Update()":       comandTag.Update(),
		"comandTag.RowsAffected()": comandTag.RowsAffected(),
		"comandTag.String()":       comandTag.String(),
	}).Debug("Return data for SQL executed in database")

	// No errors occurred
	return nil

}

// Load all data around TestCase to bes used for putting TestInstructions on the TestInstructionExecutionQueue
func (executionEngine *TestInstructionExecutionEngineStruct) loadTestCaseModelAndTestInstructionsAndTestInstructionContainersToBeAddedToExecutionQueueLoadFromCloudDB(
	dbTransaction pgx.Tx, testCaseExecutionQueueMessages []*tempTestCaseExecutionQueueInformationStruct) (
	testInstructionsInTestCases []*tempTestInstructionInTestCaseStruct, err error) {

	var testCasesUuidsToBeUsedInSQL []string

	// Loop over TestCaseExecutionQueue-messages and extract  "UniqueCounter"
	for _, testCaseExecutionQueueMessage := range testCaseExecutionQueueMessages {
		testCasesUuidsToBeUsedInSQL = append(testCasesUuidsToBeUsedInSQL, testCaseExecutionQueueMessage.testCaseUuid)
	}

	usedDBSchema := "FenixBuilder" // TODO should this env variable be used? fenixSyncShared.GetDBSchemaName()

	sqlToExecute := ""
	sqlToExecute = sqlToExecute + "SELECT DISTINCT ON (TC.\"TestCaseUuid\") "
	sqlToExecute = sqlToExecute + "TC.\"DomainUuid\", TC.\"DomainName\", TC.\"TestCaseUuid\", TC.\"TestCaseName\", TC.\"TestCaseVersion\", \"TestCaseBasicInformationAsJsonb\", \"TestInstructionsAsJsonb\", \"TestInstructionContainersAsJsonb\", TC.\"UniqueCounter\" "
	sqlToExecute = sqlToExecute + "FROM \"" + usedDBSchema + "\".\"TestCases\" TC "
	sqlToExecute = sqlToExecute + "WHERE TC.\"TestCaseUuid\" IN " + common_config.GenerateSQLINArray(testCasesUuidsToBeUsedInSQL) + " "
	sqlToExecute = sqlToExecute + "ORDER BY TC.\"TestCaseUuid\" ASC, TC.\"TestCaseVersion\" DESC; "

	// Log SQL to be executed if Environment variable is true
	if common_config.LogAllSQLs == true {
		common_config.Logger.WithFields(logrus.Fields{
			"Id":           "936d48b6-f9ec-4144-8fdd-5e04c6801370",
			"sqlToExecute": sqlToExecute,
		}).Debug("SQL to be executed within 'loadTestCaseModelAndTestInstructionsAndTestInstructionContainersToBeAddedToExecutionQueueLoadFromCloudDB'")
	}

	// Query DB
	var ctx context.Context
	ctx, timeOutCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer timeOutCancel()

	rows, err := dbTransaction.Query(ctx, sqlToExecute)
	defer rows.Close()

	if err != nil {
		common_config.Logger.WithFields(logrus.Fields{
			"Id":           "e7cef945-e58b-43b9-b8e2-f5d264e0fd21",
			"Error":        err,
			"sqlToExecute": sqlToExecute,
		}).Error("Something went wrong when executing SQL")

		return []*tempTestInstructionInTestCaseStruct{}, err
	}

	// Extract data from DB result set
	for rows.Next() {

		var tempTestCaseModelAndTestInstructionsInTestCases tempTestInstructionInTestCaseStruct

		err := rows.Scan(
			&tempTestCaseModelAndTestInstructionsInTestCases.domainUuid,
			&tempTestCaseModelAndTestInstructionsInTestCases.domainName,
			&tempTestCaseModelAndTestInstructionsInTestCases.testCaseUuid,
			&tempTestCaseModelAndTestInstructionsInTestCases.testCaseName,
			&tempTestCaseModelAndTestInstructionsInTestCases.testCaseVersion,
			&tempTestCaseModelAndTestInstructionsInTestCases.testCaseBasicInformationAsJsonb,
			&tempTestCaseModelAndTestInstructionsInTestCases.testInstructionsAsJsonb,
			&tempTestCaseModelAndTestInstructionsInTestCases.testInstructionContainersAsJsonb,
			&tempTestCaseModelAndTestInstructionsInTestCases.uniqueCounter,
		)

		if err != nil {

			common_config.Logger.WithFields(logrus.Fields{
				"Id":           "4573547c-f4a6-46b9-b8c8-6189ebb5f721",
				"Error":        err,
				"sqlToExecute": sqlToExecute,
			}).Error("Something went wrong when processing result from database")

			return []*tempTestInstructionInTestCaseStruct{}, err
		}

		// Add Queue-message to slice of messages
		testInstructionsInTestCases = append(testInstructionsInTestCases, &tempTestCaseModelAndTestInstructionsInTestCases)

	}

	return testInstructionsInTestCases, err

}

// Save all TestInstructions in 'TestInstructionExecutionQueue'
func (executionEngine *TestInstructionExecutionEngineStruct) SaveTestInstructionsToExecutionQueueSaveToCloudDB(
	dbTransaction pgx.Tx,
	testCaseExecutionQueueMessages []*tempTestCaseExecutionQueueInformationStruct,
	testInstructionsInTestCases []*tempTestInstructionInTestCaseStruct) (
	testInstructionAttributesForInstructionExecutionUuidMap map[string]tempAttributesType,
	err error) {

	// Get a common dateTimeStamp to use
	currentDataTimeStamp := fenixSyncShared.GenerateDatetimeTimeStampForDB()

	var dataRowToBeInsertedMultiType []interface{}
	var dataRowsToBeInsertedMultiType [][]interface{}
	var newTestInstructionExecutionUuid string

	//Initiate response-map
	testInstructionAttributesForInstructionExecutionUuidMap = make(map[string]tempAttributesType)

	usedDBSchema := "FenixExecution" // TODO should this env variable be used? fenixSyncShared.GetDBSchemaName()

	sqlToExecute := ""

	// Convert TestInstruction slice into map-structure
	testCaseExecutionQueueMessagesMap := make(map[string]*tempTestCaseExecutionQueueInformationStruct)
	for _, testCaseExecutionQueueMessage := range testCaseExecutionQueueMessages {
		testCaseExecutionQueueMessagesMap[testCaseExecutionQueueMessage.testCaseUuid] = testCaseExecutionQueueMessage
	}

	// Create Insert Statement for TestCaseExecutionUuid that will be put on ExecutionQueue
	// Data to be inserted in the DB-table
	dataRowsToBeInsertedMultiType = nil

	for _, testInstructionsInTestCase := range testInstructionsInTestCases {

		var testInstructions fenixTestCaseBuilderServerGrpcApi.MatureTestInstructionsMessage
		var testInstructionContainers fenixTestCaseBuilderServerGrpcApi.MatureTestInstructionContainersMessage
		var testCaseBasicInformationMessage fenixTestCaseBuilderServerGrpcApi.TestCaseBasicInformationMessage

		// Convert json-objects into their gRPC-structs
		err := protojson.Unmarshal([]byte(testInstructionsInTestCase.testInstructionsAsJsonb), &testInstructions)
		if err != nil {
			return testInstructionAttributesForInstructionExecutionUuidMap, err
		}
		err = protojson.Unmarshal([]byte(testInstructionsInTestCase.testInstructionContainersAsJsonb), &testInstructionContainers)
		if err != nil {
			return testInstructionAttributesForInstructionExecutionUuidMap, err
		}
		err = protojson.Unmarshal([]byte(testInstructionsInTestCase.testCaseBasicInformationAsJsonb), &testCaseBasicInformationMessage)
		if err != nil {
			return testInstructionAttributesForInstructionExecutionUuidMap, err
		}

		// Generate TestCaseElementModel-map
		testCaseElementModelMap := make(map[string]*fenixTestCaseBuilderServerGrpcApi.MatureTestCaseModelElementMessage) //map[testCaseUuid]*fenixTestCaseBuilderServerGrpcApi.MatureTestCaseModelElementMessage
		for _, testCaseModelElement := range testCaseBasicInformationMessage.TestCaseModel.TestCaseModelElements {
			testCaseElementModelMap[testCaseModelElement.MatureElementUuid] = testCaseModelElement
		}

		// Generate TestCaseTestInstruction-map
		testInstructionContainerMap := make(map[string]*fenixTestCaseBuilderServerGrpcApi.MatureTestInstructionContainersMessage_MatureTestInstructionContainerMessage)
		for _, testInstructionContainer := range testInstructionContainers.MatureTestInstructionContainers {
			testInstructionContainerMap[testInstructionContainer.MatureTestInstructionContainerInformation.
				MatureTestInstructionContainerInformation.TestInstructionContainerMatureUuid] = testInstructionContainer
		}

		// Initiate map for TestInstructionExecution Order
		testInstructionExecutionOrder := make(map[string]*testInstructionsRawExecutionOrderStruct) //map[matureTestInstructionUuid]*testInstructionsRawExecutionOrderStruct

		err = executionEngine.testInstructionExecutionOrderCalculator(
			testCaseBasicInformationMessage.TestCaseModel.FirstMatureElementUuid,
			&testCaseElementModelMap,
			&testInstructionExecutionOrder,
			&testInstructionContainerMap)

		if err != nil {
			if err != nil {
				common_config.Logger.WithFields(logrus.Fields{
					"Id":    "dbe7f121-1256-4bcf-883b-c6ee1bf85c4f",
					"Error": err,
				}).Error("Couldn't calculate Execution Order for TestInstructions")

				return testInstructionAttributesForInstructionExecutionUuidMap, err
			}
		}

		// Loop all TestInstructions in TestCase and add them
		for _, testInstruction := range testInstructions.MatureTestInstructions {

			dataRowToBeInsertedMultiType = nil

			// Generate the Execution-Uuid for the TestInstruction
			newTestInstructionExecutionUuid = uuidGenerator.New().String()

			// Loop attributes for TestInstructions and add Map, to be saved in Cloud-DB (on other function)
			var attributesSliceForTestInstructionToStoreInDB tempAttributesType

			for _, attribute := range testInstruction.MatureTestInstructionInformation.TestInstructionAttributesList {

				var attributeToStoreInDB tempAttributeStruct

				// Switch type of attribute, e.g. TextBox, ComboBox and so on
				switch attribute.BaseAttributeInformation.TestInstructionAttributeType {

				case fenixTestCaseBuilderServerGrpcApi.TestInstructionAttributeTypeEnum_TEXTBOX:
					attributeToStoreInDB = tempAttributeStruct{
						testCaseExecutionUuid:            testCaseExecutionQueueMessagesMap[testInstructionsInTestCase.testCaseUuid].testCaseExecutionUuid,
						testInstructionName:              testInstruction.BasicTestInstructionInformation.NonEditableInformation.TestInstructionOriginalName,
						testInstructionExecutionUuid:     newTestInstructionExecutionUuid,
						testInstructionAttributeType:     int(attribute.BaseAttributeInformation.TestInstructionAttributeType),
						TestInstructionAttributeUuid:     attribute.AttributeInformation.InputTextBoxProperty.TestInstructionAttributeInputTextBoUuid,
						TestInstructionAttributeName:     attribute.AttributeInformation.InputTextBoxProperty.TestInstructionAttributeInputTextBoxName,
						AttributeValueAsString:           attribute.AttributeInformation.InputTextBoxProperty.TextBoxAttributeValue,
						AttributeValueUuid:               common_config.ZeroUuid,
						testInstructionAttributeTypeUuid: attribute.BaseAttributeInformation.TestInstructionAttributeTypeUuid,
						testInstructionAttributeTypeName: attribute.BaseAttributeInformation.TestInstructionAttributeTypeName,
						testInstructionExecutionVersion:  1,
					}

				case fenixTestCaseBuilderServerGrpcApi.TestInstructionAttributeTypeEnum_COMBOBOX:
					attributeToStoreInDB = tempAttributeStruct{
						testCaseExecutionUuid:            testCaseExecutionQueueMessagesMap[testInstructionsInTestCase.testCaseUuid].testCaseExecutionUuid,
						testInstructionName:              testInstruction.BasicTestInstructionInformation.NonEditableInformation.TestInstructionOriginalName,
						testInstructionExecutionUuid:     newTestInstructionExecutionUuid,
						testInstructionAttributeType:     int(attribute.BaseAttributeInformation.TestInstructionAttributeType),
						TestInstructionAttributeUuid:     attribute.BaseAttributeInformation.TestInstructionAttributeUuid,
						TestInstructionAttributeName:     attribute.BaseAttributeInformation.TestInstructionAttributeName,
						AttributeValueAsString:           attribute.AttributeInformation.InputComboBoxProperty.ComboBoxAttributeValue,
						AttributeValueUuid:               attribute.AttributeInformation.InputComboBoxProperty.ComboBoxAttributeValueUuid,
						testInstructionAttributeTypeUuid: attribute.BaseAttributeInformation.TestInstructionAttributeTypeUuid,
						testInstructionAttributeTypeName: attribute.BaseAttributeInformation.TestInstructionAttributeTypeName,
						testInstructionExecutionVersion:  1,
					}

				case fenixTestCaseBuilderServerGrpcApi.TestInstructionAttributeTypeEnum_RESPONSE_VARIABLE_COMBOBOX:
					attributeToStoreInDB = tempAttributeStruct{
						testCaseExecutionUuid:        testCaseExecutionQueueMessagesMap[testInstructionsInTestCase.testCaseUuid].testCaseExecutionUuid,
						testInstructionName:          testInstruction.BasicTestInstructionInformation.NonEditableInformation.TestInstructionOriginalName,
						testInstructionExecutionUuid: newTestInstructionExecutionUuid,
						testInstructionAttributeType: int(attribute.BaseAttributeInformation.TestInstructionAttributeType),
						TestInstructionAttributeUuid: attribute.AttributeInformation.ResponseVariableComboBoxProperty.
							TestInstructionAttributeResponseVariableComboBoxUuid,
						TestInstructionAttributeName: attribute.AttributeInformation.ResponseVariableComboBoxProperty.
							TestInstructionAttributeResponseVariableComboBoxName,
						AttributeValueAsString:           attribute.AttributeInformation.ResponseVariableComboBoxProperty.ComboBoxAttributeValueAsString,
						AttributeValueUuid:               attribute.AttributeInformation.ResponseVariableComboBoxProperty.ChosenResponseVariableTypeUuid,
						testInstructionAttributeTypeUuid: attribute.BaseAttributeInformation.TestInstructionAttributeTypeUuid,
						testInstructionAttributeTypeName: attribute.BaseAttributeInformation.TestInstructionAttributeTypeName,
						testInstructionExecutionVersion:  1,
					}

				default:
					common_config.Logger.WithFields(logrus.Fields{
						"Id": "f9c124ba-beb2-40c7-a1b3-52d5b9997b2b",
						"attribute.BaseAttributeInformation.TestInstructionAttributeType": attribute.BaseAttributeInformation.TestInstructionAttributeType,
					}).Fatalln("Unknown attribute type. Exiting")
				}

				// Add attribute to slice of attributes
				attributesSliceForTestInstructionToStoreInDB = append(attributesSliceForTestInstructionToStoreInDB, attributeToStoreInDB)
			}

			// Store slice of attributes in response-map
			testInstructionAttributesForInstructionExecutionUuidMap[testInstruction.MatureTestInstructionInformation.MatureBasicTestInstructionInformation.TestInstructionMatureUuid] = attributesSliceForTestInstructionToStoreInDB

			dataRowToBeInsertedMultiType = append(dataRowToBeInsertedMultiType, testInstruction.BasicTestInstructionInformation.NonEditableInformation.DomainUuid)
			dataRowToBeInsertedMultiType = append(dataRowToBeInsertedMultiType, testInstruction.BasicTestInstructionInformation.NonEditableInformation.DomainName)
			dataRowToBeInsertedMultiType = append(dataRowToBeInsertedMultiType, newTestInstructionExecutionUuid)
			dataRowToBeInsertedMultiType = append(dataRowToBeInsertedMultiType, testInstruction.MatureTestInstructionInformation.MatureBasicTestInstructionInformation.TestInstructionMatureUuid)
			dataRowToBeInsertedMultiType = append(dataRowToBeInsertedMultiType, testInstruction.BasicTestInstructionInformation.NonEditableInformation.TestInstructionOriginalName)
			dataRowToBeInsertedMultiType = append(dataRowToBeInsertedMultiType, testInstruction.BasicTestInstructionInformation.NonEditableInformation.MajorVersionNumber)
			dataRowToBeInsertedMultiType = append(dataRowToBeInsertedMultiType, testInstruction.BasicTestInstructionInformation.NonEditableInformation.MinorVersionNumber)
			dataRowToBeInsertedMultiType = append(dataRowToBeInsertedMultiType, currentDataTimeStamp)
			dataRowToBeInsertedMultiType = append(dataRowToBeInsertedMultiType, testCaseExecutionQueueMessagesMap[testInstructionsInTestCase.testCaseUuid].executionPriority)
			dataRowToBeInsertedMultiType = append(dataRowToBeInsertedMultiType, testCaseExecutionQueueMessagesMap[testInstructionsInTestCase.testCaseUuid].testCaseExecutionUuid)
			dataRowToBeInsertedMultiType = append(dataRowToBeInsertedMultiType, testCaseExecutionQueueMessagesMap[testInstructionsInTestCase.testCaseUuid].testDataSetUuid)
			dataRowToBeInsertedMultiType = append(dataRowToBeInsertedMultiType, testCaseExecutionQueueMessagesMap[testInstructionsInTestCase.testCaseUuid].testCaseExecutionVersion)
			dataRowToBeInsertedMultiType = append(dataRowToBeInsertedMultiType, 1) //TestInstructionExecutionVersion
			dataRowToBeInsertedMultiType = append(dataRowToBeInsertedMultiType, testInstructionExecutionOrder[testInstruction.MatureTestInstructionInformation.MatureBasicTestInstructionInformation.TestInstructionMatureUuid].temporaryOrderNumber)
			dataRowToBeInsertedMultiType = append(dataRowToBeInsertedMultiType, testInstruction.BasicTestInstructionInformation.NonEditableInformation.TestInstructionOriginalUuid)
			dataRowToBeInsertedMultiType = append(dataRowToBeInsertedMultiType, testCaseExecutionQueueMessagesMap[testInstructionsInTestCase.testCaseUuid].executionStatusReportLevel)
			dataRowToBeInsertedMultiType = append(dataRowToBeInsertedMultiType, testInstruction.BasicTestInstructionInformation.NonEditableInformation.ExecutionDomainUuid)
			dataRowToBeInsertedMultiType = append(dataRowToBeInsertedMultiType, testInstruction.BasicTestInstructionInformation.NonEditableInformation.ExecutionDomainName)

			dataRowsToBeInsertedMultiType = append(dataRowsToBeInsertedMultiType, dataRowToBeInsertedMultiType)
		}
	}

	sqlToExecute = sqlToExecute + "INSERT INTO \"" + usedDBSchema + "\".\"TestInstructionExecutionQueue\" "
	sqlToExecute = sqlToExecute + "(\"DomainUuid\", \"DomainName\", \"TestInstructionExecutionUuid\", " +
		"\"MatureTestInstructionUuid\", \"TestInstructionName\", \"TestInstructionMajorVersionNumber\", " +
		"\"TestInstructionMinorVersionNumber\", \"QueueTimeStamp\", \"ExecutionPriority\", \"TestCaseExecutionUuid\"," +
		" \"TestDataSetUuid\", \"TestCaseExecutionVersion\", \"TestInstructionExecutionVersion\", " +
		"\"TestInstructionExecutionOrder\", \"TestInstructionOriginalUuid\", \"ExecutionStatusReportLevel\", " +
		"\"ExecutionDomainUuid\", \"ExecutionDomainName\" ) "
	sqlToExecute = sqlToExecute + common_config.GenerateSQLInsertValues(dataRowsToBeInsertedMultiType)
	sqlToExecute = sqlToExecute + ";"

	// Log SQL to be executed if Environment variable is true
	if common_config.LogAllSQLs == true {
		common_config.Logger.WithFields(logrus.Fields{
			"Id":           "455bfa83-d0d1-48ae-9978-daab419ac5cf",
			"sqlToExecute": sqlToExecute,
		}).Debug("SQL to be executed within 'SaveTestInstructionsToExecutionQueueSaveToCloudDB'")
	}

	// Execute Query CloudDB
	comandTag, err := dbTransaction.Exec(context.Background(), sqlToExecute)

	if err != nil {
		common_config.Logger.WithFields(logrus.Fields{
			"Id":           "7b2447a0-5790-47b5-af28-5f069c80c88a",
			"Error":        err,
			"sqlToExecute": sqlToExecute,
		}).Error("Something went wrong when executing SQL")

		return testInstructionAttributesForInstructionExecutionUuidMap, err
	}

	// Log response from CloudDB
	common_config.Logger.WithFields(logrus.Fields{
		"Id":                       "dcb110c2-822a-4dde-8bc6-9ebbe9fcbdb0",
		"comandTag.Insert()":       comandTag.Insert(),
		"comandTag.Delete()":       comandTag.Delete(),
		"comandTag.Select()":       comandTag.Select(),
		"comandTag.Update()":       comandTag.Update(),
		"comandTag.RowsAffected()": comandTag.RowsAffected(),
		"comandTag.String()":       comandTag.String(),
	}).Debug("Return data for SQL executed in database")

	// No errors occurred
	return testInstructionAttributesForInstructionExecutionUuidMap, nil

}

// Save the attributes for the TestInstructions waiting on Execution queue
func (executionEngine *TestInstructionExecutionEngineStruct) saveTestInstructionAttributesUnderExecutionSaveToCloudDB(
	dbTransaction pgx.Tx,
	testInstructionAttributesForInstructionExecutionUuidMap map[string]tempAttributesType) (
	err error) {

	common_config.Logger.WithFields(logrus.Fields{
		"Id": "23993342-01dc-40b5-b3c0-b83d1d0b2eb7",
		"testInstructionAttributesForInstructionExecutionUuidMap": testInstructionAttributesForInstructionExecutionUuidMap,
	}).Debug("Entering: saveTestInstructionAttributesUnderExecutionSaveToCloudDB()")

	defer func() {
		common_config.Logger.WithFields(logrus.Fields{
			"Id": "9c36c17f-b4f9-4383-820a-32c22c050b71",
		}).Debug("Exiting: saveTestInstructionAttributesUnderExecutionSaveToCloudDB()")
	}()

	// Get a common dateTimeStamp to use
	//currentDataTimeStamp := fenixSyncShared.GenerateDatetimeTimeStampForDB()

	var dataRowToBeInsertedMultiType []interface{}
	var dataRowsToBeInsertedMultiType [][]interface{}
	var allAttributesSlice tempAttributesType

	usedDBSchema := "FenixExecution" // TODO should this env variable be used? fenixSyncShared.GetDBSchemaName()

	sqlToExecute := ""

	// Create Insert Statement for Ongoing TestInstructionExecution
	// Data to be inserted in the DB-table
	dataRowsToBeInsertedMultiType = nil

	for _, testInstructionsAttributesPerTestInstructionExecutionUuid := range testInstructionAttributesForInstructionExecutionUuidMap {

		for _, testInstructionsAttribute := range testInstructionsAttributesPerTestInstructionExecutionUuid {

			dataRowToBeInsertedMultiType = nil

			dataRowToBeInsertedMultiType = append(dataRowToBeInsertedMultiType, testInstructionsAttribute.testInstructionExecutionUuid)
			dataRowToBeInsertedMultiType = append(dataRowToBeInsertedMultiType, testInstructionsAttribute.testInstructionAttributeType)
			dataRowToBeInsertedMultiType = append(dataRowToBeInsertedMultiType, testInstructionsAttribute.TestInstructionAttributeUuid)
			dataRowToBeInsertedMultiType = append(dataRowToBeInsertedMultiType, testInstructionsAttribute.TestInstructionAttributeName)
			dataRowToBeInsertedMultiType = append(dataRowToBeInsertedMultiType, testInstructionsAttribute.AttributeValueAsString)
			dataRowToBeInsertedMultiType = append(dataRowToBeInsertedMultiType, testInstructionsAttribute.AttributeValueUuid)
			dataRowToBeInsertedMultiType = append(dataRowToBeInsertedMultiType, testInstructionsAttribute.testInstructionAttributeTypeUuid)
			dataRowToBeInsertedMultiType = append(dataRowToBeInsertedMultiType, testInstructionsAttribute.testInstructionAttributeTypeName)
			dataRowToBeInsertedMultiType = append(dataRowToBeInsertedMultiType, testInstructionsAttribute.testInstructionExecutionVersion)
			dataRowToBeInsertedMultiType = append(dataRowToBeInsertedMultiType, testInstructionsAttribute.testCaseExecutionUuid)
			dataRowToBeInsertedMultiType = append(dataRowToBeInsertedMultiType, testInstructionsAttribute.testInstructionName)

			dataRowsToBeInsertedMultiType = append(dataRowsToBeInsertedMultiType, dataRowToBeInsertedMultiType)

			// Add attribute to list of all attributes
			allAttributesSlice = append(allAttributesSlice, testInstructionsAttribute)

		}

	}

	sqlToExecute = sqlToExecute + "INSERT INTO \"" + usedDBSchema + "\".\"TestInstructionAttributesUnderExecution\" "
	sqlToExecute = sqlToExecute + "(\"TestInstructionExecutionUuid\", \"TestInstructionAttributeType\", \"TestInstructionAttributeUuid\", " +
		"\"TestInstructionAttributeName\", \"AttributeValueAsString\", \"AttributeValueUuid\", " +
		"\"TestInstructionAttributeTypeUuid\", \"TestInstructionAttributeTypeName\", \"TestInstructionExecutionVersion\"," +
		"\"TestCaseExecutionUuid\", \"TestInstructionName\") "
	sqlToExecute = sqlToExecute + common_config.GenerateSQLInsertValues(dataRowsToBeInsertedMultiType)
	sqlToExecute = sqlToExecute + ";"

	// Log SQL to be executed if Environment variable is true
	if common_config.LogAllSQLs == true {
		common_config.Logger.WithFields(logrus.Fields{
			"Id":           "63d409bd-eea6-4724-b886-388dceadae17",
			"sqlToExecute": sqlToExecute,
		}).Debug("SQL to be executed within 'saveTestInstructionAttributesUnderExecutionSaveToCloudDB'")
	}

	// Execute Query CloudDB
	comandTag, err := dbTransaction.Exec(context.Background(), sqlToExecute)

	if err != nil {
		common_config.Logger.WithFields(logrus.Fields{
			"Id":           "8abe1477-351f-49e5-a563-94ed227dfad1",
			"Error":        err,
			"sqlToExecute": sqlToExecute,
		}).Error("Something went wrong when executing SQL")

		return err
	}

	// Log response from CloudDB
	common_config.Logger.WithFields(logrus.Fields{
		"Id":                       "dcb110c2-822a-4dde-8bc6-9ebbe9fcbdb0",
		"comandTag.Insert()":       comandTag.Insert(),
		"comandTag.Delete()":       comandTag.Delete(),
		"comandTag.Select()":       comandTag.Select(),
		"comandTag.Update()":       comandTag.Update(),
		"comandTag.RowsAffected()": comandTag.RowsAffected(),
		"comandTag.String()":       comandTag.String(),
	}).Debug("Return data for SQL executed in database")

	// Make a copy of all Attributes and them in the History table for attributes
	var (
		tempTestInstructionExecutionUuid    string
		tempTestInstructionExecutionVersion int
		tempTestInstructionAttributeUuid    string
	)

	// Loop all attributes
	for _, attribute := range allAttributesSlice {
		tempTestInstructionExecutionUuid = attribute.testInstructionExecutionUuid
		tempTestInstructionExecutionVersion = attribute.testInstructionExecutionVersion
		tempTestInstructionAttributeUuid = attribute.TestInstructionAttributeUuid

		// Add attributes to table for 'TestInstructionAttributesUnderExecution'
		err = executionEngine.makeAttributeExecutionCopyInChangeHistoryInCloudDB(
			dbTransaction,
			tempTestInstructionExecutionUuid,
			tempTestInstructionExecutionVersion,
			tempTestInstructionAttributeUuid)

		if err != nil {

			common_config.Logger.WithFields(logrus.Fields{
				"id":                                  "3a10288d-a138-43d3-8498-95bbc70fd730",
				"error":                               err,
				"tempTestInstructionExecutionUuid":    tempTestInstructionExecutionUuid,
				"tempTestInstructionExecutionVersion": tempTestInstructionExecutionVersion,
				"tempTestInstructionAttributeUuid":    tempTestInstructionAttributeUuid,
			}).Error("Couldn't add TestInstructionAttributes to history table in CloudDB")

			return err
		}
	}

	// No errors occurred
	return nil

}

// Load all TestSuitePreView-data needed. For all TestSuites
func (executionEngine *TestInstructionExecutionEngineStruct) loadAllTestSuitePreViews(
	dbTransaction pgx.Tx,
	testCaseUsedForTestSuiteExecutionQueueMessages []*tempTestCaseExecutionQueueInformationStruct,
	testCasesPreviewMapPerTestSuite map[string][]*fenixTestCaseBuilderServerGrpcApi.TestCasePreviewMessage) (
	testSuitesPreview []*fenixTestCaseBuilderServerGrpcApi.TestSuitePreviewMessage,
	err error) {

	// Generate TestSuiteUuid-slice
	var testSuitesUuidToLoad []string
	for _, testCaseUsedForTestSuiteExecutionQueueMessage := range testCaseUsedForTestSuiteExecutionQueueMessages {
		testSuitesUuidToLoad = append(testSuitesUuidToLoad, testCaseUsedForTestSuiteExecutionQueueMessage.testSuiteUuid)

	}

	// Load the TestSuite
	var fullTestSuiteMessages []*fenixTestCaseBuilderServerGrpcApi.FullTestSuiteMessage
	fullTestSuiteMessages, err = executionEngine.loadFullTestSuites(
		dbTransaction,
		testSuitesUuidToLoad)

	// Error when retrieving TestSuites
	if err != nil {
		common_config.Logger.WithFields(logrus.Fields{
			"id":    "4e3885e1-8500-4d6c-a69f-11872a975580",
			"error": err,
		}).Error("Got some problem when loading TestSuites from database")

		return nil, err
	}

	// Loop all TestSuite and create TestSuite-previews for the TestSuites
	for _, fullTestSuiteMessage := range fullTestSuiteMessages {

		// Create TestSuite-preview and add to TestSuite-Object
		// Create the 'TestSuitePreview'
		var tempSelectedTestSuiteMetaDataValuesMap map[string]*fenixTestCaseBuilderServerGrpcApi.
			TestSuitePreviewStructureMessage_SelectedTestSuiteMetaDataValueMessage
		tempSelectedTestSuiteMetaDataValuesMap = make(map[string]*fenixTestCaseBuilderServerGrpcApi.
			TestSuitePreviewStructureMessage_SelectedTestSuiteMetaDataValueMessage)

		var tempTestSuitePreview *fenixTestCaseBuilderServerGrpcApi.TestSuitePreviewMessage
		tempTestSuitePreview = &fenixTestCaseBuilderServerGrpcApi.TestSuitePreviewMessage{
			TestSuitePreview: &fenixTestCaseBuilderServerGrpcApi.TestSuitePreviewStructureMessage{
				TestSuiteUuid:                 fullTestSuiteMessage.TestSuiteBasicInformation.GetTestSuiteUuid(),
				TestSuiteName:                 fullTestSuiteMessage.TestSuiteBasicInformation.GetTestSuiteName(),
				TestSuiteVersion:              strconv.Itoa(int(fullTestSuiteMessage.TestSuiteBasicInformation.GetTestSuiteVersion())),
				DomainUuidThatOwnTheTestSuite: fullTestSuiteMessage.TestSuiteBasicInformation.GetDomainUuid(),
				DomainNameThatOwnTheTestSuite: fullTestSuiteMessage.TestSuiteBasicInformation.GetDomainName(),
				TestSuiteDescription:          fullTestSuiteMessage.GetTestSuiteBasicInformation().GetTestSuiteDescription(),
				TestSuiteStructureObjects: &fenixTestCaseBuilderServerGrpcApi.
					TestSuitePreviewStructureMessage_TestSuiteStructureObjectMessage{TestCasePreViews: testCasesPreviewMapPerTestSuite[fullTestSuiteMessage.TestSuiteBasicInformation.GetTestSuiteUuid()]},
				LastSavedByUserOnComputer:          fullTestSuiteMessage.UpdatedByAndWhen.GetUserIdOnComputer(),
				LastSavedByUserGCPAuthorization:    fullTestSuiteMessage.UpdatedByAndWhen.GetGCPAuthenticatedUser(),
				LastSavedTimeStamp:                 fullTestSuiteMessage.UpdatedByAndWhen.GetUpdateTimeStamp(),
				SelectedTestSuiteMetaDataValuesMap: tempSelectedTestSuiteMetaDataValuesMap,
			},
			TestSuitePreviewHash: "",
		}

		// Generate 'SelectedTestSuiteMetaDataValuesMap'
		for _, tempMetaDataGroupMessagePtr := range fullTestSuiteMessage.TestSuiteMetaData.MetaDataGroupsMap {

			// Get the 'MetaDataGroupMessage' from ptr
			tempMetaDataGroupMessage := *tempMetaDataGroupMessagePtr

			// Loop 'MetaDataInGroupMap'
			for SelectedTestSuiteMetaDataValuesMap, tempMetaDataInGroupMessagePtr := range tempMetaDataGroupMessage.MetaDataInGroupMap {

				switch tempMetaDataInGroupMessagePtr.GetSelectType() {

				case fenixTestCaseBuilderServerGrpcApi.MetaDataSelectTypeEnum_MetaDataSelectType_SingleSelect:

					// Create 'SelectedTestSuiteMetaDataValueMessage' to be stored in Map
					var tempSelectedTestSuiteMetaDataValueMessage *fenixTestCaseBuilderServerGrpcApi.
						TestSuitePreviewStructureMessage_SelectedTestSuiteMetaDataValueMessage

					tempSelectedTestSuiteMetaDataValueMessage = &fenixTestCaseBuilderServerGrpcApi.
						TestSuitePreviewStructureMessage_SelectedTestSuiteMetaDataValueMessage{
						OwnerDomainUuid:   tempSelectedTestSuiteMetaDataValueMessage.GetOwnerDomainUuid(),
						OwnerDomainName:   tempSelectedTestSuiteMetaDataValueMessage.GetOwnerDomainName(),
						MetaDataGroupName: tempMetaDataInGroupMessagePtr.GetMetaDataGroupName(),
						MetaDataName:      tempMetaDataInGroupMessagePtr.GetMetaDataName(),
						MetaDataNameValue: tempMetaDataInGroupMessagePtr.GetSelectedMetaDataValueForSingleSelect(),
						SelectType:        tempSelectedTestSuiteMetaDataValueMessage.GetSelectType(),
						IsMandatory:       tempSelectedTestSuiteMetaDataValueMessage.GetIsMandatory(),
					}

					// Add value to map
					tempSelectedTestSuiteMetaDataValuesMap[SelectedTestSuiteMetaDataValuesMap] = tempSelectedTestSuiteMetaDataValueMessage

				case fenixTestCaseBuilderServerGrpcApi.MetaDataSelectTypeEnum_MetaDataSelectType_MultiSelect:

					// Loop all selected values and add to map
					for _, tempSelectedMetaDataValue := range tempMetaDataInGroupMessagePtr.GetSelectedMetaDataValuesForMultiSelect() {

						// Create 'SelectedTestSuiteMetaDataValueMessage' to be stored in Map
						var tempSelectedTestSuiteMetaDataValueMessage *fenixTestCaseBuilderServerGrpcApi.
							TestSuitePreviewStructureMessage_SelectedTestSuiteMetaDataValueMessage

						tempSelectedTestSuiteMetaDataValueMessage = &fenixTestCaseBuilderServerGrpcApi.
							TestSuitePreviewStructureMessage_SelectedTestSuiteMetaDataValueMessage{
							OwnerDomainUuid:   tempSelectedTestSuiteMetaDataValueMessage.GetOwnerDomainUuid(),
							OwnerDomainName:   tempSelectedTestSuiteMetaDataValueMessage.GetOwnerDomainName(),
							MetaDataGroupName: tempMetaDataInGroupMessagePtr.GetMetaDataGroupName(),
							MetaDataName:      tempMetaDataInGroupMessagePtr.GetMetaDataName(),
							MetaDataNameValue: tempSelectedMetaDataValue,
							SelectType:        tempSelectedTestSuiteMetaDataValueMessage.GetSelectType(),
							IsMandatory:       tempSelectedTestSuiteMetaDataValueMessage.GetIsMandatory(),
						}

						// Add value to map
						tempSelectedTestSuiteMetaDataValuesMap[SelectedTestSuiteMetaDataValuesMap] = tempSelectedTestSuiteMetaDataValueMessage

					}

				default:
					common_config.Logger.WithFields(logrus.Fields{
						"Id":                   "6972f543-5fba-414e-b9d2-079a374d0f48",
						"fullTestSuiteMessage": fullTestSuiteMessage,
						"tempMetaDataInGroupMessagePtr.GetSelectType()": tempMetaDataInGroupMessagePtr.GetSelectType(),
					}).Error("Unknown SelectType in 'MetaDataInGroupMap'")

					errID := "e532e909-05a5-4642-90f0-f0fc33cffecb"
					err = errors.New(fmt.Sprintf("Unknown SelectType in 'MetaDataInGroupMap', %d [ErrorId=%s]",
						tempMetaDataInGroupMessagePtr.GetSelectType(), errID))

					return nil, err

				}
			}
		}

		// Add 'tempSelectedTestSuiteMetaDataValuesMap' to 'tempTestSuitePreview'
		tempTestSuitePreview.TestSuitePreview.SelectedTestSuiteMetaDataValuesMap = tempSelectedTestSuiteMetaDataValuesMap

		// Calculate 'TestSuitePreviewHash'
		var tempTestSuitePreviewHash string
		tempJson := protojson.Format(tempTestSuitePreview)
		tempTestSuitePreviewHash = common_config.HashSingleValue(tempJson)

		tempTestSuitePreview.TestSuitePreviewHash = tempTestSuitePreviewHash

		// Add TestSuite to slice of TestSuites
		testSuitesPreview = append(testSuitesPreview, tempTestSuitePreview)

	}

	return testSuitesPreview, nil
}

// This date is used as delete date when the TestSuite is not deleted
const testSuiteNotDeletedDate = "2068-11-18"

// Load the TestSuite with all its data
func (executionEngine *TestInstructionExecutionEngineStruct) loadFullTestSuites(
	dbTransaction pgx.Tx,
	testSuitesUuidToLoad []string) (
	fullTestSuiteMessages []*fenixTestCaseBuilderServerGrpcApi.FullTestSuiteMessage,
	err error) {

	sqlToExecute := ""
	sqlToExecute = sqlToExecute + "WITH uniquecounters AS ( "
	sqlToExecute = sqlToExecute + "SELECT Distinct ON (\"TestSuiteUuid\")  \"UniqueCounter\" "
	sqlToExecute = sqlToExecute + "FROM \"FenixBuilder\".\"TestSuites\" "
	sqlToExecute = sqlToExecute + fmt.Sprintf("WHERE \"TestSuiteUuid\" IN %s ", common_config.GenerateSQLINArray(testSuitesUuidToLoad))
	sqlToExecute = sqlToExecute + "ORDER BY \"TestSuiteUuid\", \"UniqueCounter\" DESC "
	sqlToExecute = sqlToExecute + ") "

	sqlToExecute = sqlToExecute + "SELECT TS.\"DomainUuid\", da.\"domain_name\", TS.\"TestSuiteUuid\", TS.\"TestSuiteName\", " +
		"TS.\"TestSuiteVersion\", TS.\"TestSuiteDescription\", TS.\"TestSuiteExecutionEnvironment\", TS.\"TestSuiteHash\", TS.\"DeleteTimestamp\", " +
		"TS.\"InsertTimeStamp\", TS.\"InsertedByUserIdOnComputer\", TS.\"InsertedByGCPAuthenticatedUser\", " +
		"TS.\"TestCasesInTestSuite\", TS.\"TestSuitePreview\", TS.\"TestSuiteMetaData\", TS.\"TestSuiteTestData\", " +
		"TS.\"TestSuiteType\", TS.\"TestSuiteTypeName\", TS.\"TestSuiteImplementedFunctions\" "

	sqlToExecute = sqlToExecute + "FROM \"FenixBuilder\".\"TestSuites\" TS, \"FenixDomainAdministration\".\"domains\" da  "
	sqlToExecute = sqlToExecute + fmt.Sprintf("WHERE TS.\"TestSuiteUuid\" IN %s ", common_config.GenerateSQLINArray(testSuitesUuidToLoad))
	sqlToExecute = sqlToExecute + "AND "
	sqlToExecute = sqlToExecute + "TS.\"UniqueCounter\" IN (SELECT * FROM uniquecounters) AND "
	sqlToExecute = sqlToExecute + "TS.\"DeleteTimestamp\" > now() AND "
	sqlToExecute = sqlToExecute + "TS.\"DomainUuid\" = da.\"domain_uuid\" "
	sqlToExecute = sqlToExecute + "; "

	// Log SQL to be executed if Environment variable is true
	if common_config.LogAllSQLs == true {
		common_config.Logger.WithFields(logrus.Fields{
			"Id":           "f4d9fb06-b042-4bf4-8140-5dc3062cc60f",
			"sqlToExecute": sqlToExecute,
		}).Debug("SQL to be executed within 'loadFullTestSuites'")
	}

	// Query DB
	var ctx context.Context
	ctx, timeOutCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer timeOutCancel()

	rows, err := dbTransaction.Query(ctx, sqlToExecute)
	defer rows.Close()

	if err != nil {
		common_config.Logger.WithFields(logrus.Fields{
			"Id":           "f1745336-7242-4131-bc89-832109dfc789",
			"Error":        err,
			"sqlToExecute": sqlToExecute,
		}).Error("Something went wrong when executing SQL")

		return nil, err
	}

	var (
		tempDomainUuid                          string
		tempDomainName                          string
		tempTestSuiteUuid                       string
		tempTestSuiteName                       string
		tempTestSuiteDescription                string
		tempTestSuiteExecutionEnvironment       string
		tempTestSuiteVersion                    int
		tempTestSuiteHash                       string
		tempDeleteTimestampAsString             string
		tempUpdatedByAndWhenAsString            string
		tempInsertedByUserIdOnComputer          string
		tempInsertedByGCPAuthenticatedUser      string
		tempTestCasesInTestSuiteAsJson          string
		tempTestSuitePreviewAsJson              string
		tempTestSuiteMetaDataAsJson             string
		tempTestSuiteTestDataAsJson             string
		tempTestSuiteType                       int
		tempTestSuiteTypeName                   string
		tempTestSuiteImplementedFunctionsAsJson string

		tempDeleteTimestampAsTimeStamp  time.Time
		tempUpdatedByAndWhenAsTimeStamp time.Time

		tempTestCasesInTestSuiteAsByteArray          []byte
		tempTestSuitePreviewAsByteArray              []byte
		tempTestSuiteMetaDataAsByteArray             []byte
		tempTestSuiteTestDataAsByteArray             []byte
		tempTestSuiteImplementedFunctionsAsByteArray []byte

		tempTestCasesInTestSuiteAsGrpc          fenixTestCaseBuilderServerGrpcApi.TestCasesInTestSuiteMessage
		tempTestSuitePreviewAsGrpc              fenixTestCaseBuilderServerGrpcApi.TestSuitePreviewMessage
		tempTestSuiteMetaDataAsGrpc             fenixTestCaseBuilderServerGrpcApi.UserSpecifiedTestSuiteMetaDataMessage
		tempTestSuiteTestDataAsGrpc             fenixTestCaseBuilderServerGrpcApi.UsersChosenTestDataForTestSuiteMessage
		tempTestSuiteImplementedFunctionsAsGrpc map[int32]bool
	)

	// Extract data from DB result set
	for rows.Next() {

		err = rows.Scan(
			&tempDomainUuid,
			&tempDomainName,
			&tempTestSuiteUuid,
			&tempTestSuiteName,

			&tempTestSuiteVersion,
			&tempTestSuiteDescription,
			&tempTestSuiteExecutionEnvironment,
			&tempTestSuiteHash,
			&tempDeleteTimestampAsTimeStamp,

			&tempUpdatedByAndWhenAsTimeStamp,
			&tempInsertedByUserIdOnComputer,
			&tempInsertedByGCPAuthenticatedUser,

			&tempTestCasesInTestSuiteAsJson,
			&tempTestSuitePreviewAsJson,
			&tempTestSuiteMetaDataAsJson,
			&tempTestSuiteTestDataAsJson,

			&tempTestSuiteType,
			&tempTestSuiteTypeName,
			&tempTestSuiteImplementedFunctionsAsJson,
		)

		if err != nil {

			common_config.Logger.WithFields(logrus.Fields{
				"Id":           "7c3b623e-cac5-4cf2-99e3-2e3d7694d70c",
				"Error":        err,
				"sqlToExecute": sqlToExecute,
			}).Error("Something went wrong when processing result from database")

			return nil, err
		}

		// Format Dates
		// Format The Delete Date into a string
		tempDeleteTimestampAsString = tempDeleteTimestampAsTimeStamp.Format("2006-01-02")

		// When the TestCase is not deleted then it uses a Delete date far away in the future. If so then clear the Date sent to TesterGui

		if tempDeleteTimestampAsString == testSuiteNotDeletedDate {
			tempDeleteTimestampAsString = ""
		}

		// Format Insert date
		tempUpdatedByAndWhenAsString = tempUpdatedByAndWhenAsTimeStamp.String()

		// Convert json-strings into byte-arrays
		tempTestCasesInTestSuiteAsByteArray = []byte(tempTestCasesInTestSuiteAsJson)
		tempTestSuitePreviewAsByteArray = []byte(tempTestSuitePreviewAsJson)
		tempTestSuiteMetaDataAsByteArray = []byte(tempTestSuiteMetaDataAsJson)
		tempTestSuiteTestDataAsByteArray = []byte(tempTestSuiteTestDataAsJson)
		tempTestSuiteImplementedFunctionsAsByteArray = []byte(tempTestSuiteImplementedFunctionsAsJson)

		// Convert json-byte-arrays into proto-messages
		err = protojson.Unmarshal(tempTestCasesInTestSuiteAsByteArray, &tempTestCasesInTestSuiteAsGrpc)
		if err != nil {
			common_config.Logger.WithFields(logrus.Fields{
				"Id":    "2e73987c-1693-4673-b366-0b56efcfbc09",
				"Error": err,
			}).Error("Something went wrong when converting 'tempTestCasesInTestSuiteAsByteArray' into proto-message")

			return nil, err
		}

		err = protojson.Unmarshal(tempTestSuitePreviewAsByteArray, &tempTestSuitePreviewAsGrpc)
		if err != nil {
			common_config.Logger.WithFields(logrus.Fields{
				"Id":    "9a4b309a-68af-49c9-8120-096e894788cd",
				"Error": err,
			}).Error("Something went wrong when converting 'tempTestSuitePreviewAsByteArray' into proto-message")

			return nil, err
		}

		err = protojson.Unmarshal(tempTestSuiteMetaDataAsByteArray, &tempTestSuiteMetaDataAsGrpc)
		if err != nil {
			common_config.Logger.WithFields(logrus.Fields{
				"Id":    "4643c3d5-87bf-4a5f-a171-b01a9d3c0d4b",
				"Error": err,
			}).Error("Something went wrong when converting 'tempTestSuiteMetaDataAsByteArray' into proto-message")

			return nil, err
		}

		err = protojson.Unmarshal(tempTestSuiteTestDataAsByteArray, &tempTestSuiteTestDataAsGrpc)
		if err != nil {
			common_config.Logger.WithFields(logrus.Fields{
				"Id":    "ef74050a-d52b-4282-a520-32f9ca1ceecf",
				"Error": err,
			}).Error("Something went wrong when converting 'tempTestSuiteTestDataAsByteArray' into proto-message")

			return nil, err
		}

		tempTestSuiteImplementedFunctionsAsGrpc = make(map[int32]bool)
		err = json.Unmarshal(tempTestSuiteImplementedFunctionsAsByteArray, &tempTestSuiteImplementedFunctionsAsGrpc)
		if err != nil {
			common_config.Logger.WithFields(logrus.Fields{
				"Id":    "2a737639-2fbc-470b-80bc-c09972c6768d",
				"Error": err,
			}).Error("Something went wrong when converting 'tempTestSuiteImplementedFunctionsAsByteArray' into proto-message")

			return nil, err
		}

		// Add the different parts into full TestSuite-message
		var fullTestSuiteMessage *fenixTestCaseBuilderServerGrpcApi.FullTestSuiteMessage
		fullTestSuiteMessage = &fenixTestCaseBuilderServerGrpcApi.FullTestSuiteMessage{
			TestSuiteBasicInformation: &fenixTestCaseBuilderServerGrpcApi.TestSuiteBasicInformationMessage{
				DomainUuid:                    tempDomainUuid,
				DomainName:                    tempDomainName,
				TestSuiteUuid:                 tempTestSuiteUuid,
				TestSuiteVersion:              uint32(tempTestSuiteVersion),
				TestSuiteName:                 tempTestSuiteName,
				TestSuiteDescription:          tempTestSuiteDescription,
				TestSuiteExecutionEnvironment: tempTestSuiteExecutionEnvironment,
			},
			TestSuiteTestData:    &tempTestSuiteTestDataAsGrpc,
			TestSuitePreview:     &tempTestSuitePreviewAsGrpc,
			TestSuiteMetaData:    &tempTestSuiteMetaDataAsGrpc,
			TestCasesInTestSuite: &tempTestCasesInTestSuiteAsGrpc,
			DeletedDate:          tempDeleteTimestampAsString,
			UpdatedByAndWhen: &fenixTestCaseBuilderServerGrpcApi.UpdatedByAndWhenMessage{
				UserIdOnComputer:     tempInsertedByUserIdOnComputer,
				GCPAuthenticatedUser: tempInsertedByGCPAuthenticatedUser,
				UpdateTimeStamp:      tempUpdatedByAndWhenAsString,
			},
			TestSuiteType: &fenixTestCaseBuilderServerGrpcApi.TestSuiteTypeMessage{
				TestSuiteType:     fenixTestCaseBuilderServerGrpcApi.TestSuiteTypeEnum(tempTestSuiteType),
				TestSuiteTypeName: tempTestSuiteTypeName,
			},
			TestSuiteImplementedFunctionsMap: tempTestSuiteImplementedFunctionsAsGrpc,
			MessageHash:                      tempTestSuiteHash,
		}

		fullTestSuiteMessages = append(fullTestSuiteMessages, fullTestSuiteMessage)

	}

	return fullTestSuiteMessages, err

}

// Add who created the TestSuite and when it was created
func (executionEngine *TestInstructionExecutionEngineStruct) addTestSuiteCreator(
	dbTransaction pgx.Tx,
	testSuiteUuidToLoad string,
	fullTestSuiteMessage *fenixTestCaseBuilderServerGrpcApi.FullTestSuiteMessage) (
	err error) {

	sqlToExecute := ""
	sqlToExecute = sqlToExecute + "SELECT \"InsertTimeStamp\", \"InsertedByUserIdOnComputer\", \"InsertedByGCPAuthenticatedUser\" "
	sqlToExecute = sqlToExecute + "FROM \"FenixBuilder\".\"TestSuites\" "
	sqlToExecute = sqlToExecute + fmt.Sprintf("WHERE \"TestSuiteUuid\" = '%s' ", testSuiteUuidToLoad)
	sqlToExecute = sqlToExecute + "ORDER BY \"UniqueCounter\" ASC "
	sqlToExecute = sqlToExecute + "LIMIT 1 "
	sqlToExecute = sqlToExecute + "; "

	// Log SQL to be executed if Environment variable is true
	if common_config.LogAllSQLs == true {
		common_config.Logger.WithFields(logrus.Fields{
			"Id":           "e195cf4d-d9fc-444b-bfe8-65c4a666bfd5",
			"sqlToExecute": sqlToExecute,
		}).Debug("SQL to be executed within 'addTestSuiteCreator'")
	}

	// Query DB
	var ctx context.Context
	ctx, timeOutCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer timeOutCancel()

	rows, err := dbTransaction.Query(ctx, sqlToExecute)
	defer rows.Close()

	if err != nil {
		common_config.Logger.WithFields(logrus.Fields{
			"Id":           "95382e00-59e0-4082-be16-fe56bee992d0",
			"Error":        err,
			"sqlToExecute": sqlToExecute,
		}).Error("Something went wrong when executing SQL")

		return err
	}

	var (
		tempInsertTimeStampAsString    string
		InsertedByUserIdOnComputer     string
		InsertedByGCPAuthenticatedUser string

		tempInsertTimeStampAsTimeStamp time.Time

		numberOfRows int
	)

	// Extract data from DB result set
	for rows.Next() {

		err = rows.Scan(
			&tempInsertTimeStampAsTimeStamp,
			&InsertedByUserIdOnComputer,
			&InsertedByGCPAuthenticatedUser,
		)

		if err != nil {

			common_config.Logger.WithFields(logrus.Fields{
				"Id":           "1f5d50c8-d35a-4d87-be33-80f7e64f7a93",
				"Error":        err,
				"sqlToExecute": sqlToExecute,
			}).Error("Something went wrong when processing result from database")

			return err
		}

		// Format Dates
		// Format Insert date
		tempInsertTimeStampAsString = tempInsertTimeStampAsTimeStamp.String()

		// increase number of rows
		numberOfRows = numberOfRows + 1
	}

	if numberOfRows != 1 {
		common_config.Logger.WithFields(logrus.Fields{
			"Id":           "d1987560-3ea1-4081-baba-c9e3fc699553",
			"Error":        err,
			"sqlToExecute": sqlToExecute,
			"numberOfRows": numberOfRows,
		}).Error("Number of rows expected to be one, but was not.")

		err = errors.New("number of rows expected to be one, but was not")

		return err

	}

	// Update Created information
	fullTestSuiteMessage.UpdatedByAndWhen.CreatedByComputerLogin = InsertedByUserIdOnComputer
	fullTestSuiteMessage.UpdatedByAndWhen.CreatedByGcpLogin = InsertedByGCPAuthenticatedUser
	fullTestSuiteMessage.UpdatedByAndWhen.CreatedDate = tempInsertTimeStampAsString

	return err

}

// *************************************************************************************************************
// Extract ExecutionOrder for TestInstructions
func (executionEngine *TestInstructionExecutionEngineStruct) testInstructionExecutionOrderCalculator(
	elementsUuid string,
	testCaseElementModelMapReference *map[string]*fenixTestCaseBuilderServerGrpcApi.MatureTestCaseModelElementMessage,
	testInstructionExecutionOrderMapReference *map[string]*testInstructionsRawExecutionOrderStruct,
	testInstructionContainerMapReference *map[string]*fenixTestCaseBuilderServerGrpcApi.MatureTestInstructionContainersMessage_MatureTestInstructionContainerMessage) (err error) {

	// Extract 'Raw ExecutionOrder' for TestInstructions by recursive process element-model-tree
	err = executionEngine.recursiveTestInstructionExecutionOrderCalculator(
		elementsUuid,
		testCaseElementModelMapReference,
		[]int{0},
		testInstructionExecutionOrderMapReference,
		testInstructionContainerMapReference)

	if err != nil {
		return err
	}

	//*** Convert 'Raw ExecutionOrder'[1,11,403] into 'Processed ExecutionOrder'[001,011,403] ***
	// Loop over Row ExecutionOrderNumbers and find the size of each number
	maxNumberOfDigitsFound := -1
	testInstructionExecutionOrderMap := *testInstructionExecutionOrderMapReference
	for _, testInstructionExecutionOrderRef := range testInstructionExecutionOrderMap {
		testInstructionExecutionOrder := testInstructionExecutionOrderRef
		for _, subpartOfTestInstructionExecutionOrder := range testInstructionExecutionOrder.rawExecutionOrder {
			if len(fmt.Sprint(subpartOfTestInstructionExecutionOrder)) > maxNumberOfDigitsFound {
				maxNumberOfDigitsFound = len(fmt.Sprint(subpartOfTestInstructionExecutionOrder))
			}
		}
	}

	var sortedTestInstructionExecutionOrderSlice testInstructionsRawExecutionOrderSliceType

	// Create the 'Processed ExecutionOrder'[001,011,403] and a then a temporary OrderNumber {1011403} from 'Raw ExecutionOrder'[1,11,403]
	for _, testInstructionExecutionOrderRef := range testInstructionExecutionOrderMap {
		testInstructionExecutionOrder := testInstructionExecutionOrderRef
		var processExecutionOrder []string
		for _, subpartOfTestInstructionExecutionOrder := range testInstructionExecutionOrder.rawExecutionOrder {
			numberOfLeadingZeros := maxNumberOfDigitsFound - len(fmt.Sprint(subpartOfTestInstructionExecutionOrder))

			formatString := "%0" + fmt.Sprint(numberOfLeadingZeros) + "d"
			processExecutionOrderNumber := fmt.Sprintf(formatString, subpartOfTestInstructionExecutionOrder)

			processExecutionOrder = append(processExecutionOrder, processExecutionOrderNumber)

		}
		// Add the 'Processed ExecutionOrder' [001,011,403]
		testInstructionExecutionOrder.processedExecutionOrder = processExecutionOrder

		// Create and add a temporary OrderNumber {1011403} from 'Processed ExecutionOrder' [001,011,403]
		temporaryOrderNumberAsString := strings.Join(processExecutionOrder[:], "")
		temporaryOrderNumber, err := strconv.ParseInt(temporaryOrderNumberAsString, 10, 64)
		if err != nil {
			return err
		}
		testInstructionExecutionOrder.temporaryOrderNumber = temporaryOrderNumber

		// Add to slice that can be sorted
		sortedTestInstructionExecutionOrderSlice = append(sortedTestInstructionExecutionOrderSlice, *testInstructionExecutionOrder)
	}

	//*** Sort on temporary OrderNumber [1011403] and then create the OrderNumber [5] ***
	sort.Sort(testInstructionsRawExecutionOrderSliceType(sortedTestInstructionExecutionOrderSlice))

	for orderNumber, testInstruction := range sortedTestInstructionExecutionOrderSlice {

		// Extract the TestInstruction and add Execution OrderNumber
		testInstructionSorted, existsInMap := testInstructionExecutionOrderMap[testInstruction.testInstructionUuid]
		if existsInMap == false {

			errorId := "9c574bd3-5494-477f-aaf1-26ede3f281ce"

			err = errors.New(fmt.Sprintf("couldn't find TestInstruction %s in 'testInstructionExecutionOrderMap' [ErrorId: %s]",
				testInstruction.testInstructionUuid,
				errorId))
			return err
		}
		// Add order number to TestInstruction
		testInstructionSorted.orderNumber = orderNumber

		// Save the TestInstruction back in Map
		testInstructionExecutionOrderMap[testInstruction.testInstructionUuid] = testInstructionSorted
	}

	return err
}

type testInstructionsRawExecutionOrderStruct struct {
	testInstructionUuid     string
	rawExecutionOrder       []int
	processedExecutionOrder []string
	temporaryOrderNumber    int64
	orderNumber             int
}
type testInstructionsRawExecutionOrderSliceType []testInstructionsRawExecutionOrderStruct

func (e testInstructionsRawExecutionOrderSliceType) Len() int {
	return len(e)
}

func (e testInstructionsRawExecutionOrderSliceType) Less(i, j int) bool {
	return e[i].temporaryOrderNumber < e[j].temporaryOrderNumber
}

func (e testInstructionsRawExecutionOrderSliceType) Swap(i, j int) {
	e[i], e[j] = e[j], e[i]
}

// *************************************************************************************************************
// Extract ExecutionOrder for TestInstructions by recursive process element-model-tree
func (executionEngine *TestInstructionExecutionEngineStruct) recursiveTestInstructionExecutionOrderCalculator(
	elementsUuid string,
	testCaseElementModelMapReference *map[string]*fenixTestCaseBuilderServerGrpcApi.MatureTestCaseModelElementMessage,
	currentExecutionOrder []int,
	testInstructionExecutionOrderMapReference *map[string]*testInstructionsRawExecutionOrderStruct,
	testInstructionContainerMapReference *map[string]*fenixTestCaseBuilderServerGrpcApi.MatureTestInstructionContainersMessage_MatureTestInstructionContainerMessage) (err error) {

	// Extract current element
	testCaseElementModelMap := *testCaseElementModelMapReference
	currentElement, existInMap := testCaseElementModelMap[elementsUuid]

	// If the element doesn't exit then there is something really wrong
	if existInMap == false {
		// This shouldn't happen
		common_config.Logger.WithFields(logrus.Fields{
			"id":           "9f628356-2ea2-48a6-8e6a-546a5f97f05b",
			"elementsUuid": elementsUuid,
		}).Error(elementsUuid + " could not be found in in map 'testCaseElementModelMap'")

		errorId := "1839091b-d76c-4b3c-819e-9c3d517e9698"

		err = errors.New(fmt.Sprintf("%s could not be found in in map 'testCaseElementModelMap' [ErrorId: %s]",
			elementsUuid,
			errorId))

		return err
	}

	// Save TestInstructions ExecutionOrder
	if currentElement.TestCaseModelElementType == fenixTestCaseBuilderServerGrpcApi.TestCaseModelElementTypeEnum_TI_TESTINSTRUCTION ||
		currentElement.TestCaseModelElementType == fenixTestCaseBuilderServerGrpcApi.TestCaseModelElementTypeEnum_TIx_TESTINSTRUCTION_NONE_REMOVABLE {

		testInstructionExecutionOrderMap := *testInstructionExecutionOrderMapReference
		_, existInMap := testInstructionExecutionOrderMap[elementsUuid]

		// If the element does exit then there is something really wrong
		if existInMap == true {
			// This shouldn't happen
			common_config.Logger.WithFields(logrus.Fields{
				"id":           "db8472c1-9383-4a43-b475-ff7218f13ff5",
				"elementsUuid": elementsUuid,
			}).Error(elementsUuid + " testInstruction already exits in could not be found in in map 'testCaseElementModelMap'")

			errorId := "ea7dcef1-659c-414e-9793-6f99152ae2f8"

			err = errors.New(fmt.Sprintf("%s testInstruction can already be found in in map 'testCaseElementModelMap' [ErrorId: %s]",
				elementsUuid,
				errorId))

			return err
		}

		// Create new variable for current Execution
		var tempCurrentExecutionOrder []int
		for _, executionOrder := range currentExecutionOrder {
			tempCurrentExecutionOrder = append(tempCurrentExecutionOrder, executionOrder)
		}

		// Add TestInstruction to execution order map
		testInstructionExecutionOrderMap[elementsUuid] = &testInstructionsRawExecutionOrderStruct{
			testInstructionUuid:     elementsUuid,
			rawExecutionOrder:       tempCurrentExecutionOrder,
			processedExecutionOrder: []string{},
		}

	}

	// Check if parent TestInstructionContainer is executing in parallell or in serial
	var parentTestContainerExecutesInParallell bool

	// When TIC is at the top then set parent as parallell (though it doesn't matter)
	if currentElement.MatureElementUuid == currentElement.ParentElementUuid {
		parentTestContainerExecutesInParallell = true

	} else {
		parentElement, existInMap := testCaseElementModelMap[currentElement.ParentElementUuid]

		// If the element doesn't exit then there is something really wrong
		if existInMap == false {
			// This shouldn't happen
			common_config.Logger.WithFields(logrus.Fields{
				"id":                               "e023bedb-ea12-4e31-9002-711f2babdb4f",
				"currentElement.ParentElementUuid": currentElement.ParentElementUuid,
			}).Error("parent element with uuid: " + currentElement.ParentElementUuid + " could not be found in in map 'testCaseElementModelMap'")

			errorId := "6d133f40-0313-4ceb-9771-580d6d293406"

			err = errors.New(fmt.Sprintf("%s parent element with uuid: %s could not be found in in map 'testCaseElementModelMap' [ErrorId: %s]",
				elementsUuid,
				currentElement.ParentElementUuid,
				errorId))

			return err
		}

		// Extract TestInstructionContainer
		testInstructionContainerMap := *testInstructionContainerMapReference
		parentTestInstructionContainer, existInMap := testInstructionContainerMap[parentElement.MatureElementUuid]

		// If the TIC doesn't exist then there is something really wrong
		if existInMap == false {
			// This shouldn't happen
			common_config.Logger.WithFields(logrus.Fields{
				"executionEngine":     "ecd29086-f3a4-45cf-9e72-45f222d81d99",
				"TestInstructionUUid": parentElement.MatureElementUuid,
			}).Error("TestInstructionContainer: " + parentElement.MatureElementUuid + " could not be found in in map 'testInstructionContainerMap'")

			errorId := "d0757934-9c2a-488f-9dde-5362f408a9f4"

			err = errors.New(fmt.Sprintf("testInstructionContainer: %s could not be found in in map 'testInstructionContainerMap' [ErrorId: %s]",
				parentElement.MatureElementUuid,
				errorId))

			return err
		}

		// Extract if TICs execution parameter is for serial vs parallell
		if parentTestInstructionContainer.BasicTestInstructionContainerInformation.EditableTestInstructionContainerAttributes.TestInstructionContainerExecutionType == fenixTestCaseBuilderServerGrpcApi.TestInstructionContainerExecutionTypeEnum_PARALLELLED_PROCESSED {
			// Parallell
			parentTestContainerExecutesInParallell = true

		} else {
			// Serial
			parentTestContainerExecutesInParallell = false
		}

	}

	// Element has child-element then go that path
	if currentElement.FirstChildElementUuid != elementsUuid {

		// Check if parent TestInstructionContainer executes in Serial or in Parallell
		if currentElement.TestCaseModelElementType == fenixTestCaseBuilderServerGrpcApi.TestCaseModelElementTypeEnum_TI_TESTINSTRUCTION ||
			currentElement.TestCaseModelElementType == fenixTestCaseBuilderServerGrpcApi.TestCaseModelElementTypeEnum_TIx_TESTINSTRUCTION_NONE_REMOVABLE ||
			currentElement.TestCaseModelElementType == fenixTestCaseBuilderServerGrpcApi.TestCaseModelElementTypeEnum_TIC_TESTINSTRUCTIONCONTAINER ||
			currentElement.TestCaseModelElementType == fenixTestCaseBuilderServerGrpcApi.TestCaseModelElementTypeEnum_TICx_TESTINSTRUCTIONCONTAINER_NONE_REMOVABLE {
			if parentTestContainerExecutesInParallell == false {
				// Parent is Serial processed

			} else {
				// Parent is Parallell processed
				currentExecutionOrder = append(currentExecutionOrder, 0)
			}
		}

		// Recursive call to child-element
		err = executionEngine.recursiveTestInstructionExecutionOrderCalculator(
			currentElement.FirstChildElementUuid,
			testCaseElementModelMapReference,
			currentExecutionOrder,
			testInstructionExecutionOrderMapReference,
			testInstructionContainerMapReference)
	}

	// If we got an error back then something wrong happen, so just back out
	if err != nil {
		return err
	}

	// If element has a next-element the go that path
	if currentElement.NextElementUuid != elementsUuid {

		// Check if parent TestInstructionContainer executes in Serial or in Parallell
		if currentElement.TestCaseModelElementType == fenixTestCaseBuilderServerGrpcApi.TestCaseModelElementTypeEnum_TI_TESTINSTRUCTION ||
			currentElement.TestCaseModelElementType == fenixTestCaseBuilderServerGrpcApi.TestCaseModelElementTypeEnum_TIx_TESTINSTRUCTION_NONE_REMOVABLE ||
			currentElement.TestCaseModelElementType == fenixTestCaseBuilderServerGrpcApi.TestCaseModelElementTypeEnum_TIC_TESTINSTRUCTIONCONTAINER ||
			currentElement.TestCaseModelElementType == fenixTestCaseBuilderServerGrpcApi.TestCaseModelElementTypeEnum_TICx_TESTINSTRUCTIONCONTAINER_NONE_REMOVABLE {
			if parentTestContainerExecutesInParallell == false {
				// Parent is Serial processed
				lastPositionValue := currentExecutionOrder[len(currentExecutionOrder)-1]
				lastPositionValue = lastPositionValue + 1
				currentExecutionOrder[len(currentExecutionOrder)-1] = lastPositionValue
			} else {
				// Parent is Parallell processed

			}
		}

		// Recursive call to next-element
		err = executionEngine.recursiveTestInstructionExecutionOrderCalculator(
			currentElement.NextElementUuid,
			testCaseElementModelMapReference,
			currentExecutionOrder,
			testInstructionExecutionOrderMapReference,
			testInstructionContainerMapReference)
	}

	// If we got an error back then something wrong happen, so just back out
	if err != nil {
		return err
	}

	return nil
}

// See https://www.alexedwards.net/blog/using-postgresql-jsonb
// Make the Attrs struct implement the driver.Valuer interface. This method
// simply returns the JSON-encoded representation of the struct.
func (a myAttrStruct) Value() (driver.Value, error) {

	return json.Marshal(a)
}

// Make the Attrs struct implement the sql.Scanner interface. This method
// simply decodes a JSON-encoded value into the struct fields.
func (a *myAttrStruct) Scan(value interface{}) error {
	b, ok := value.([]byte)
	if !ok {

		errorId := "5ee90fed-02fc-4f9e-9927-dd0498ae3e6c"

		return errors.New(fmt.Sprintf("type assertion to []byte failed [ErrorId: %s]", errorId))
	}

	return json.Unmarshal(b, &a)
}

type myAttrStruct struct {
	fenixTestCaseBuilderServerGrpcApi.BasicTestCaseInformationMessage
}
