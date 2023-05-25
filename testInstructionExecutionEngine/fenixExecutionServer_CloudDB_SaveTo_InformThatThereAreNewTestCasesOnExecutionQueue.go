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
	testCaseExecutionsToProcess []ChannelCommandTestCaseExecutionStruct) (ackNackResponse *fenixExecutionServerGrpcApi.AckNackResponse) {

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
	testCaseExecutionQueueMessages, err := executionEngine.loadTestCaseExecutionQueueMessages(
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
	err = executionEngine.saveTestCasesOnOngoingExecutionsQueueSaveToCloudDB(txn, testCaseExecutionQueueMessages)
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
			AckNack:                      false,
			Comments:                     "Problem when saving to database",
			ErrorCodes:                   errorCodes,
			ProtoFileVersionUsedByClient: fenixExecutionServerGrpcApi.CurrentFenixExecutionServerProtoFileVersionEnum(common_config.GetHighestFenixExecutionServerProtoFileVersion()),
		}

		return ackNackResponse

	}

	// Delete messages in ExecutionQueue that has been put to ongoing executions
	err = executionEngine.clearTestCasesExecutionQueueSaveToCloudDB(txn, testCaseExecutionQueueMessages)
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
			AckNack:                      false,
			Comments:                     "Problem when saving to database",
			ErrorCodes:                   errorCodes,
			ProtoFileVersionUsedByClient: fenixExecutionServerGrpcApi.CurrentFenixExecutionServerProtoFileVersionEnum(common_config.GetHighestFenixExecutionServerProtoFileVersion()),
		}

		return ackNackResponse

	}

	//Load all data around TestCase to be used for putting TestInstructions on the TestInstructionExecutionQueue
	allDataAroundAllTestCase, err := executionEngine.loadTestCaseModelAndTestInstructionsAndTestInstructionContainersToBeAddedToExecutionQueueLoadFromCloudDB(txn, testCaseExecutionQueueMessages)
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
	testInstructionUuidTotestInstructionExecutionUuidMap, err := executionEngine.SaveTestInstructionsToExecutionQueueSaveToCloudDB(txn, testCaseExecutionQueueMessages, allDataAroundAllTestCase)
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
			AckNack:                      false,
			Comments:                     "Problem when saving to database",
			ErrorCodes:                   errorCodes,
			ProtoFileVersionUsedByClient: fenixExecutionServerGrpcApi.CurrentFenixExecutionServerProtoFileVersionEnum(common_config.GetHighestFenixExecutionServerProtoFileVersion()),
		}

		return ackNackResponse

	}

	// Add attributes to table for 'TestInstructionAttributesUnderExecution'
	err = executionEngine.saveTestInstructionAttributesUnderExecutionSaveToCloudDB(txn, testInstructionUuidTotestInstructionExecutionUuidMap)
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
	domainUuid                string
	domainName                string
	testSuiteUuid             string
	testSuiteName             string
	testSuiteVersion          int
	testSuiteExecutionUuid    string
	testSuiteExecutionVersion int
	testCaseUuid              string
	testCaseName              string
	testCaseVersion           int
	testCaseExecutionUuid     string
	testCaseExecutionVersion  int
	queueTimeStamp            time.Time
	testDataSetUuid           string
	executionPriority         int
	uniqueCounter             int
}

// Struct to use with variable to hold TestInstructionExecutionQueue-messages
type tempTestInstructionExecutionQueueInformationStruct struct {
	domainUuid                        string
	domainName                        string
	testInstructionExecutionUuid      string
	testInstructionUuid               string
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
}

// Struct to be used when extracting TestInstructions from TestCases
type tempTestInstructionInTestCaseStruct struct {
	domainUuid                       string
	domainName                       string
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
	testInstructionExecutionUuid     string
	testInstructionAttributeType     int
	TestInstructionAttributeUuid     string
	TestInstructionAttributeName     string
	AttributeValueAsString           string
	AttributeValueUuid               string
	testInstructionAttributeTypeUuid string
	testInstructionAttributeTypeName string
}

// Load TestCaseExecutionQueue-Messages be able to populate the ongoing TestCaseExecutionUuid-table
func (executionEngine *TestInstructionExecutionEngineStruct) loadTestCaseExecutionQueueMessages(dbTransaction pgx.Tx, testCaseExecutionsToProcess []ChannelCommandTestCaseExecutionStruct) (testCaseExecutionQueueMessages []*tempTestCaseExecutionQueueInformationStruct, err error) {

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
	// Execute Query CloudDB
	rows, err := dbTransaction.Query(context.Background(), sqlToExecute)

	if err != nil {
		common_config.Logger.WithFields(logrus.Fields{
			"Id":           "85459587-0c1e-4db9-b257-742ff3a660fc",
			"Error":        err,
			"sqlToExecute": sqlToExecute,
		}).Error("Something went wrong when executing SQL")

		return []*tempTestCaseExecutionQueueInformationStruct{}, err
	}

	var testCaseExecutionQueueMessage tempTestCaseExecutionQueueInformationStruct

	// USed to secure that exactly one row was found
	numberOfRowFromDB := 0

	// Extract data from DB result set
	for rows.Next() {

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
func (executionEngine *TestInstructionExecutionEngineStruct) saveTestCasesOnOngoingExecutionsQueueSaveToCloudDB(dbTransaction pgx.Tx, testCaseExecutionQueueMessages []*tempTestCaseExecutionQueueInformationStruct) (err error) {

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
	var suiteInformationExists bool

	usedDBSchema := "FenixExecution" // TODO should this env variable be used? fenixSyncShared.GetDBSchemaName()

	sqlToExecute := ""

	// Create Insert Statement for TestCaseExecutionUuid that will be put on ExecutionQueue
	// Data to be inserted in the DB-table
	dataRowsToBeInsertedMultiType = nil

	for _, testCaseExecutionQueueMessage := range testCaseExecutionQueueMessages {

		dataRowToBeInsertedMultiType = nil

		// Check if this is a SingleTestCase-execution. Then use UUIDs from TestCase in Suite-uuid-parts

		if testCaseExecutionQueueMessage.executionPriority == int(fenixExecutionServerGrpcApi.ExecutionPriorityEnum_HIGH_SINGLE_TESTCASE) ||
			testCaseExecutionQueueMessage.executionPriority == int(fenixExecutionServerGrpcApi.ExecutionPriorityEnum_MEDIUM_MULTIPLE_TESTCASES) {

			suiteInformationExists = false
		} else {
			suiteInformationExists = true
		}

		dataRowToBeInsertedMultiType = append(dataRowToBeInsertedMultiType, testCaseExecutionQueueMessage.domainUuid)
		dataRowToBeInsertedMultiType = append(dataRowToBeInsertedMultiType, testCaseExecutionQueueMessage.domainName)

		if suiteInformationExists == true {
			dataRowToBeInsertedMultiType = append(dataRowToBeInsertedMultiType, testCaseExecutionQueueMessage.testSuiteUuid)
		} else {
			dataRowToBeInsertedMultiType = append(dataRowToBeInsertedMultiType, testCaseExecutionQueueMessage.testCaseUuid)
		}

		dataRowToBeInsertedMultiType = append(dataRowToBeInsertedMultiType, testCaseExecutionQueueMessage.testSuiteName)
		dataRowToBeInsertedMultiType = append(dataRowToBeInsertedMultiType, testCaseExecutionQueueMessage.testSuiteVersion)

		if suiteInformationExists == true {
			dataRowToBeInsertedMultiType = append(dataRowToBeInsertedMultiType, testCaseExecutionQueueMessage.testSuiteExecutionUuid)
		} else {
			dataRowToBeInsertedMultiType = append(dataRowToBeInsertedMultiType, testCaseExecutionQueueMessage.testCaseExecutionUuid)
		}

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

		dataRowsToBeInsertedMultiType = append(dataRowsToBeInsertedMultiType, dataRowToBeInsertedMultiType)

	}

	sqlToExecute = sqlToExecute + "INSERT INTO \"" + usedDBSchema + "\".\"TestCasesUnderExecution\" "
	sqlToExecute = sqlToExecute + "(\"DomainUuid\", \"DomainName\", \"TestSuiteUuid\", \"TestSuiteName\", \"TestSuiteVersion\", " +
		"\"TestSuiteExecutionUuid\", \"TestSuiteExecutionVersion\", \"TestCaseUuid\", \"TestCaseName\", \"TestCaseVersion\"," +
		" \"TestCaseExecutionUuid\", \"TestCaseExecutionVersion\", \"QueueTimeStamp\", \"TestDataSetUuid\", \"ExecutionPriority\", " +
		"\"ExecutionStartTimeStamp\", \"ExecutionStopTimeStamp\", \"TestCaseExecutionStatus\", \"ExecutionHasFinished\", " +
		"\"ExecutionStatusUpdateTimeStamp\") "
	sqlToExecute = sqlToExecute + common_config.GenerateSQLInsertValues(dataRowsToBeInsertedMultiType)
	sqlToExecute = sqlToExecute + ";"

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

	// Query DB
	rows, err := dbTransaction.Query(context.Background(), sqlToExecute)

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
func (executionEngine *TestInstructionExecutionEngineStruct) SaveTestInstructionsToExecutionQueueSaveToCloudDB(dbTransaction pgx.Tx, testCaseExecutionQueueMessages []*tempTestCaseExecutionQueueInformationStruct, testInstructionsInTestCases []*tempTestInstructionInTestCaseStruct) (testInstructionAttributesForInstructionExecutionUuidMap map[string]tempAttributesType, err error) {

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
			testInstructionContainerMap[testInstructionContainer.MatureTestInstructionContainerInformation.MatureTestInstructionContainerInformation.TestInstructionContainerMatureUuid] = testInstructionContainer
		}

		// Initiate map for TestInstructionExecution Order
		testInstructionExecutionOrder := make(map[string]*testInstructionsRawExecutionOrderStruct) //map[testInstructionUuid]*testInstructionsRawExecutionOrderStruct

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
			var attributesSliceToStoreInDB tempAttributesType

			for _, attribute := range testInstruction.MatureTestInstructionInformation.TestInstructionAttributesList {

				var attributeToStoreInDB tempAttributeStruct

				// Switch type of attribute, e.g. TextBox, ComboBox and so on
				switch attribute.BaseAttributeInformation.TestInstructionAttributeType {

				case fenixTestCaseBuilderServerGrpcApi.TestInstructionAttributeTypeEnum_TEXTBOX:
					attributeToStoreInDB = tempAttributeStruct{
						testInstructionExecutionUuid:     newTestInstructionExecutionUuid,
						testInstructionAttributeType:     int(attribute.BaseAttributeInformation.TestInstructionAttributeType),
						TestInstructionAttributeUuid:     attribute.AttributeInformation.InputTextBoxProperty.TestInstructionAttributeInputTextBoUuid,
						TestInstructionAttributeName:     attribute.AttributeInformation.InputTextBoxProperty.TestInstructionAttributeInputTextBoxName,
						AttributeValueAsString:           attribute.AttributeInformation.InputTextBoxProperty.TextBoxAttributeValue,
						AttributeValueUuid:               attribute.AttributeInformation.InputTextBoxProperty.TestInstructionAttributeInputTextBoUuid,
						testInstructionAttributeTypeUuid: attribute.BaseAttributeInformation.TestInstructionAttributeTypeUuid,
						testInstructionAttributeTypeName: attribute.BaseAttributeInformation.TestInstructionAttributeTypeName,
					}

				case fenixTestCaseBuilderServerGrpcApi.TestInstructionAttributeTypeEnum_COMBOBOX:
					attributeToStoreInDB = tempAttributeStruct{
						testInstructionExecutionUuid:     newTestInstructionExecutionUuid,
						testInstructionAttributeType:     int(attribute.BaseAttributeInformation.TestInstructionAttributeType),
						TestInstructionAttributeUuid:     attribute.AttributeInformation.InputComboBoxProperty.TestInstructionAttributeComboBoxUuid,
						TestInstructionAttributeName:     attribute.AttributeInformation.InputComboBoxProperty.TestInstructionAttributeComboBoxName,
						AttributeValueAsString:           attribute.AttributeInformation.InputComboBoxProperty.ComboBoxAttributeValue,
						AttributeValueUuid:               attribute.AttributeInformation.InputComboBoxProperty.ComboBoxAttributeValueUuid,
						testInstructionAttributeTypeUuid: attribute.BaseAttributeInformation.TestInstructionAttributeTypeUuid,
						testInstructionAttributeTypeName: attribute.BaseAttributeInformation.TestInstructionAttributeTypeName,
					}

				default:
					common_config.Logger.WithFields(logrus.Fields{
						"Id": "f9c124ba-beb2-40c7-a1b3-52d5b9997b2b",
						"attribute.BaseAttributeInformation.TestInstructionAttributeType": attribute.BaseAttributeInformation.TestInstructionAttributeType,
					}).Fatalln("Unknown attribute type. Exiting")
				}

				// Add attribute to slice of attributes
				attributesSliceToStoreInDB = append(attributesSliceToStoreInDB, attributeToStoreInDB)
			}

			// Store slice of attributes in response-map
			testInstructionAttributesForInstructionExecutionUuidMap[testInstruction.MatureTestInstructionInformation.MatureBasicTestInstructionInformation.TestInstructionMatureUuid] = attributesSliceToStoreInDB

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
			dataRowToBeInsertedMultiType = append(dataRowToBeInsertedMultiType, testInstruction.BasicTestInstructionInformation.NonEditableInformation.TestInstructionOrignalUuid)

			dataRowsToBeInsertedMultiType = append(dataRowsToBeInsertedMultiType, dataRowToBeInsertedMultiType)
		}
	}

	sqlToExecute = sqlToExecute + "INSERT INTO \"" + usedDBSchema + "\".\"TestInstructionExecutionQueue\" "
	sqlToExecute = sqlToExecute + "(\"DomainUuid\", \"DomainName\", \"TestInstructionExecutionUuid\", \"TestInstructionUuid\", \"TestInstructionName\", " +
		"\"TestInstructionMajorVersionNumber\", \"TestInstructionMinorVersionNumber\", \"QueueTimeStamp\", \"ExecutionPriority\", \"TestCaseExecutionUuid\"," +
		" \"TestDataSetUuid\", \"TestCaseExecutionVersion\", \"TestInstructionExecutionVersion\", \"TestInstructionExecutionOrder\", \"TestInstructionOriginalUuid\") "
	sqlToExecute = sqlToExecute + common_config.GenerateSQLInsertValues(dataRowsToBeInsertedMultiType)
	sqlToExecute = sqlToExecute + ";"

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
func (executionEngine *TestInstructionExecutionEngineStruct) saveTestInstructionAttributesUnderExecutionSaveToCloudDB(dbTransaction pgx.Tx, testInstructionAttributesForInstructionExecutionUuidMap map[string]tempAttributesType) (err error) {

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

			dataRowsToBeInsertedMultiType = append(dataRowsToBeInsertedMultiType, dataRowToBeInsertedMultiType)

		}
	}

	sqlToExecute = sqlToExecute + "INSERT INTO \"" + usedDBSchema + "\".\"TestInstructionAttributesUnderExecution\" "
	sqlToExecute = sqlToExecute + "(\"TestInstructionExecutionUuid\", \"TestInstructionAttributeType\", \"TestInstructionAttributeUuid\", " +
		"\"TestInstructionAttributeName\", \"AttributeValueAsString\", \"AttributeValueUuid\", " +
		"\"TestInstructionAttributeTypeUuid\", \"TestInstructionAttributeTypeName\") "
	sqlToExecute = sqlToExecute + common_config.GenerateSQLInsertValues(dataRowsToBeInsertedMultiType)
	sqlToExecute = sqlToExecute + ";"

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

	// No errors occurred
	return nil

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
			err = errors.New(fmt.Sprintf("couldn't find TestInstruction %s in 'testInstructionExecutionOrderMap'", testInstruction.testInstructionUuid))
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

		err = errors.New(elementsUuid + " could not be found in in map 'testCaseElementModelMap'")

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

			err = errors.New(elementsUuid + " testInstruction can already be found in in map 'testCaseElementModelMap'")

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

			err = errors.New(elementsUuid + "parent element with uuid: " + currentElement.ParentElementUuid + " could not be found in in map 'testCaseElementModelMap'")

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

			err = errors.New("testInstructionContainer: " + parentElement.MatureElementUuid + " could not be found in in map 'testInstructionContainerMap'")

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
		return errors.New("type assertion to []byte failed")
	}

	return json.Unmarshal(b, &a)
}

type myAttrStruct struct {
	fenixTestCaseBuilderServerGrpcApi.BasicTestCaseInformationMessage
}
