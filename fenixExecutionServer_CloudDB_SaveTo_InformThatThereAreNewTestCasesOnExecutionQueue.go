package main

import (
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
)

// Prepare for Saving the llongoing Execution of a new TestCaseExecution in the CloudDB
func (fenixExecutionServerObject *fenixExecutionServerObjectStruct) prepareInformThatThereAreNewTestCasesOnExecutionQueueSaveToCloudDB(emptyParameter *fenixExecutionServerGrpcApi.EmptyParameter) (ackNackResponse *fenixExecutionServerGrpcApi.AckNackResponse) {

	// Begin SQL Transaction
	txn, err := fenixSyncShared.DbPool.Begin(context.Background())
	if err != nil {
		fenixExecutionServerObject.logger.WithFields(logrus.Fields{
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
			ProtoFileVersionUsedByClient: fenixExecutionServerGrpcApi.CurrentFenixExecutionServerProtoFileVersionEnum(fenixExecutionServerObject.getHighestFenixTestDataProtoFileVersion()),
		}

		return ackNackResponse
	}
	defer txn.Commit(context.Background())

	// Generate a new TestCaseExecution-UUID
	//testCaseExecutionUuid := uuidGenerator.New().String()

	// Generate TimeStamp
	//placedOnTestExecutionQueueTimeStamp := time.Now()

	// Extract TestCaseExecutionQueue-messages to be added to data for ongoing Executions
	testCaseExecutionQueueMessages, err := fenixExecutionServerObject.loadTestCaseExecutionQueueMessages()
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
			ProtoFileVersionUsedByClient: fenixExecutionServerGrpcApi.CurrentFenixExecutionServerProtoFileVersionEnum(fenixExecutionServerObject.getHighestFenixTestDataProtoFileVersion()),
		}

		return ackNackResponse
	}

	//TODO ROLLBACK if error - in all places

	// Save the Initiation of a new TestCaseExecution in the CloudDB
	err = fenixExecutionServerObject.saveTestCasesOnOngoingExecutionsQueueSaveToCloudDB(txn, testCaseExecutionQueueMessages)
	if err != nil {

		fenixExecutionServerObject.logger.WithFields(logrus.Fields{
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
			ProtoFileVersionUsedByClient: fenixExecutionServerGrpcApi.CurrentFenixExecutionServerProtoFileVersionEnum(fenixExecutionServerObject.getHighestFenixTestDataProtoFileVersion()),
		}

		return ackNackResponse

	}

	// Delete messages in ExecutionQueue that has been put to ongoing executions
	err = fenixExecutionServerObject.clearTestCasesExecutionQueueSaveToCloudDB(txn, testCaseExecutionQueueMessages)
	if err != nil {

		fenixExecutionServerObject.logger.WithFields(logrus.Fields{
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
			ProtoFileVersionUsedByClient: fenixExecutionServerGrpcApi.CurrentFenixExecutionServerProtoFileVersionEnum(fenixExecutionServerObject.getHighestFenixTestDataProtoFileVersion()),
		}

		return ackNackResponse

	}

	//Load all data around TestCase to bes used for putting TestInstructions on the TestInstructionExecutionQueue
	allDataAroundAllTestCase, err := fenixExecutionServerObject.loadTestCaseModelAndTestInstructionsAndTestInstructionContainersToBeAddedToExecutionQueueLoadFromCloudDB(testCaseExecutionQueueMessages)
	if err != nil {

		fenixExecutionServerObject.logger.WithFields(logrus.Fields{
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
			ProtoFileVersionUsedByClient: fenixExecutionServerGrpcApi.CurrentFenixExecutionServerProtoFileVersionEnum(fenixExecutionServerObject.getHighestFenixTestDataProtoFileVersion()),
		}

		return ackNackResponse

	}

	// Add TestInstructions to TestInstructionsExecutionQueue
	err = fenixExecutionServerObject.SaveTestInstructionsToExecutionQueueSaveToCloudDB(txn, testCaseExecutionQueueMessages, allDataAroundAllTestCase)
	if err != nil {

		fenixExecutionServerObject.logger.WithFields(logrus.Fields{
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
			ProtoFileVersionUsedByClient: fenixExecutionServerGrpcApi.CurrentFenixExecutionServerProtoFileVersionEnum(fenixExecutionServerObject.getHighestFenixTestDataProtoFileVersion()),
		}

		return ackNackResponse

	}

	// Trigger TestInstructionExecutionEngine by sending

	ackNackResponse = &fenixExecutionServerGrpcApi.AckNackResponse{
		AckNack:                      true,
		Comments:                     "",
		ErrorCodes:                   []fenixExecutionServerGrpcApi.ErrorCodesEnum{},
		ProtoFileVersionUsedByClient: fenixExecutionServerGrpcApi.CurrentFenixExecutionServerProtoFileVersionEnum(fenixExecutionServerObject.getHighestFenixTestDataProtoFileVersion()),
	}

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
	queueTimeStamp            string
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
	domainUuid                      string
	domainName                      string
	testCaseUuid                    string
	testCaseName                    string
	testCaseVersion                 int
	testCaseBasicInformationAsJsonb string
	testInstructionsAsJsonb         string
	testInstructionContainersAsJsonb string
	uniqueCounter                   int
}

// Load TestCaseExecutionQueue-Messages be able to populate the ongoing TestCaseExecution-table
func (fenixExecutionServerObject *fenixExecutionServerObjectStruct) loadTestCaseExecutionQueueMessages() (testCaseExecutionQueueMessages []*tempTestCaseExecutionQueueInformationStruct, err error) {

	usedDBSchema := "FenixExecution" // TODO should this env variable be used? fenixSyncShared.GetDBSchemaName()

	sqlToExecute := ""
	sqlToExecute = sqlToExecute + "SELECT TCEQ.* "
	sqlToExecute = sqlToExecute + "FROM \"" + usedDBSchema + "\".\"TestCaseExecutionQueue\" TCEQ "
	sqlToExecute = sqlToExecute + "ORDER BY TCEQ.\"QueueTimeStamp\" ASC; "

	// Query DB
	rows, err := fenixSyncShared.DbPool.Query(context.Background(), sqlToExecute)

	if err != nil {
		fenixExecutionServerObject.logger.WithFields(logrus.Fields{
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

			fenixExecutionServerObject.logger.WithFields(logrus.Fields{
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
func (fenixExecutionServerObject *fenixExecutionServerObjectStruct) saveTestCasesOnOngoingExecutionsQueueSaveToCloudDB(dbTransaction pgx.Tx, testCaseExecutionQueueMessages []*tempTestCaseExecutionQueueInformationStruct) (err error) {

	fenixExecutionServerObject.logger.WithFields(logrus.Fields{
		"Id": "8e857aa4-3f15-4415-bc08-5ac97bf64446",
	}).Debug("Entering: saveTestCasesOnOngoingExecutionsQueueSaveToCloudDB()")

	defer func() {
		fenixExecutionServerObject.logger.WithFields(logrus.Fields{
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

	// Create Insert Statement for TestCaseExecution that will be put on ExecutionQueue
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

		dataRowsToBeInsertedMultiType = append(dataRowsToBeInsertedMultiType, dataRowToBeInsertedMultiType)

	}

	sqlToExecute = sqlToExecute + "INSERT INTO \"" + usedDBSchema + "\".\"TestCasesUnderExecution\" "
	sqlToExecute = sqlToExecute + "(\"DomainUuid\", \"DomainName\", \"TestSuiteUuid\", \"TestSuiteName\", \"TestSuiteVersion\", " +
		"\"TestSuiteExecutionUuid\", \"TestSuiteExecutionVersion\", \"TestCaseUuid\", \"TestCaseName\", \"TestCaseVersion\"," +
		" \"TestCaseExecutionUuid\", \"TestCaseExecutionVersion\", \"QueueTimeStamp\", \"TestDataSetUuid\", \"ExecutionPriority\", " +
		"\"ExecutionStartTimeStamp\", \"ExecutionStopTimeStamp\", \"TestCaseExecutionStatus\", \"ExecutionHasFinished\") "
	sqlToExecute = sqlToExecute + fenixExecutionServerObject.generateSQLInsertValues(dataRowsToBeInsertedMultiType)
	sqlToExecute = sqlToExecute + ";"

	// Execute Query CloudDB
	comandTag, err := dbTransaction.Exec(context.Background(), sqlToExecute)

	if err != nil {
		fenixExecutionServerObject.logger.WithFields(logrus.Fields{
			"Id":           "7b2447a0-5790-47b5-af28-5f069c80c88a",
			"Error":        err,
			"sqlToExecute": sqlToExecute,
		}).Error("Something went wrong when executing SQL")

		return err
	}

	// Log response from CloudDB
	fenixExecutionServerObject.logger.WithFields(logrus.Fields{
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
func (fenixExecutionServerObject *fenixExecutionServerObjectStruct) clearTestCasesExecutionQueueSaveToCloudDB(dbTransaction pgx.Tx, testCaseExecutionQueueMessages []*tempTestCaseExecutionQueueInformationStruct) (err error) {

	fenixExecutionServerObject.logger.WithFields(logrus.Fields{
		"Id": "7703b634-a46d-4494-897f-1f139b5858c5",
	}).Debug("Entering: clearTestCasesExecutionQueueSaveToCloudDB()")

	defer func() {
		fenixExecutionServerObject.logger.WithFields(logrus.Fields{
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
	sqlToExecute = sqlToExecute + fenixExecutionServerObject.generateSQLINIntegerArray(testCaseExecutionsToBeDeletedFromQueue)
	sqlToExecute = sqlToExecute + ";"

	// Execute Query CloudDB
	comandTag, err := dbTransaction.Exec(context.Background(), sqlToExecute)

	if err != nil {
		fenixExecutionServerObject.logger.WithFields(logrus.Fields{
			"Id":           "38a5ca13-c108-427a-a24a-20c3b6d6c4be",
			"Error":        err,
			"sqlToExecute": sqlToExecute,
		}).Error("Something went wrong when executing SQL")

		return err
	}

	// Log response from CloudDB
	fenixExecutionServerObject.logger.WithFields(logrus.Fields{
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

//Load all data around TestCase to bes used for putting TestInstructions on the TestInstructionExecutionQueue
func (fenixExecutionServerObject *fenixExecutionServerObjectStruct) loadTestCaseModelAndTestInstructionsAndTestInstructionContainersToBeAddedToExecutionQueueLoadFromCloudDB(testCaseExecutionQueueMessages []*tempTestCaseExecutionQueueInformationStruct) (testInstructionsInTestCases []*tempTestInstructionInTestCaseStruct, err error) {

	var testCasesUuidsToBeUsedInSQL []string

	// Loop over TestCaseExecutionQueue-messages and extract  "UniqueCounter"
	for _, testCaseExecutionQueueMessage := range testCaseExecutionQueueMessages {
		testCasesUuidsToBeUsedInSQL = append(testCasesUuidsToBeUsedInSQL, testCaseExecutionQueueMessage.testCaseUuid)
	}

	usedDBSchema := "FenixExecution" // TODO should this env variable be used? fenixSyncShared.GetDBSchemaName()

	sqlToExecute := ""
	sqlToExecute = sqlToExecute + "SELECT DISTINCT ON (TC.\"TestCaseUuid\") "
	sqlToExecute = sqlToExecute + "TC.\"DomainUuid\", TC.\"DomainName\", TC.\"TestCaseUuid\", TC.\"TestCaseName\", TC.\"TestCaseVersion\", \"TestCaseBasicInformationAsJsonb\", \"TestInstructionsAsJsonb\", \"TestInstructionContainersAsJsonb\", TC.\"UniqueCounter\" "
	sqlToExecute = sqlToExecute + "FROM \"" + usedDBSchema + "\".\"TestCases\" TC "
	sqlToExecute = sqlToExecute + "WHERE TC.\"TestCaseUuid\" IN " + fenixExecutionServerObject.generateSQLINArray(testCasesUuidsToBeUsedInSQL) + " "
	sqlToExecute = sqlToExecute + "ORDER BY TC.\"TestCaseUuid\" ASC, TC.\"TestCaseVersion\" DESC; "

	// Query DB
	rows, err := fenixSyncShared.DbPool.Query(context.Background(), sqlToExecute)

	if err != nil {
		fenixExecutionServerObject.logger.WithFields(logrus.Fields{
			"Id":           "e7cef945-e58b-43b9-b8e2-f5d264e0fd21",
			"Error":        err,
			"sqlToExecute": sqlToExecute,
		}).Error("Something went wrong when executing SQL")

		return []*tempTestInstructionInTestCaseStruct{}, err
	}

	// Extract data from DB result set
	for rows.Next() {

		var tempTestCaseModelAndTestInstructionsInTestCases *tempTestInstructionInTestCaseStruct

		err := rows.Scan(
			&tempTestCaseModelAndTestInstructionsInTestCases.domainUuid,
			&tempTestCaseModelAndTestInstructionsInTestCases.domainName,
			&tempTestCaseModelAndTestInstructionsInTestCases.testCaseUuid,
			&tempTestCaseModelAndTestInstructionsInTestCases.testCaseVersion,
			&tempTestCaseModelAndTestInstructionsInTestCases.testCaseBasicInformationAsJsonb,
			&tempTestCaseModelAndTestInstructionsInTestCases.testInstructionsAsJsonb,
			&tempTestCaseModelAndTestInstructionsInTestCases.testInstructionContainersAsJsonb,
			&tempTestCaseModelAndTestInstructionsInTestCases.uniqueCounter,
		)

		if err != nil {

			fenixExecutionServerObject.logger.WithFields(logrus.Fields{
				"Id":           "4573547c-f4a6-46b9-b8c8-6189ebb5f721",
				"Error":        err,
				"sqlToExecute": sqlToExecute,
			}).Error("Something went wrong when processing result from database")

			return []*tempTestInstructionInTestCaseStruct{}, err
		}

		// Add Queue-message to slice of messages
		testInstructionsInTestCases = append(testInstructionsInTestCases, tempTestCaseModelAndTestInstructionsInTestCases)

	}

	return testInstructionsInTestCases, err

}

// Save all TestInstructions in TestInstructionExecutionQueue
func (fenixExecutionServerObject *fenixExecutionServerObjectStruct) SaveTestInstructionsToExecutionQueueSaveToCloudDB(dbTransaction pgx.Tx, testCaseExecutionQueueMessages []*tempTestCaseExecutionQueueInformationStruct, testInstructionsInTestCases []*tempTestInstructionInTestCaseStruct) (err error) {

	// Get a common dateTimeStamp to use
	currentDataTimeStamp := fenixSyncShared.GenerateDatetimeTimeStampForDB()

	var dataRowToBeInsertedMultiType []interface{}
	var dataRowsToBeInsertedMultiType [][]interface{}

	usedDBSchema := "FenixExecution" // TODO should this env variable be used? fenixSyncShared.GetDBSchemaName()

	sqlToExecute := ""



	// Convert TestInstruction slice into map-structure
	testCaseExecutionQueueMessagesMap := make(map[string]*tempTestCaseExecutionQueueInformationStruct)
	for _, testCaseExecutionQueueMessages := range testCaseExecutionQueueMessages {
		testCaseExecutionQueueMessagesMap[testCaseExecutionQueueMessages.testCaseUuid] = testCaseExecutionQueueMessages
	}

	// Create Insert Statement for TestCaseExecution that will be put on ExecutionQueue
	// Data to be inserted in the DB-table
	dataRowsToBeInsertedMultiType = nil

	for _, testInstructionsInTestCase := range testInstructionsInTestCases {

		var testInstructions fenixTestCaseBuilderServerGrpcApi.MatureTestInstructionsMessage
		var testInstructionContainers fenixTestCaseBuilderServerGrpcApi.MatureTestInstructionContainerMessage
		var testCaseBasicInformationMessage fenixTestCaseBuilderServerGrpcApi.TestCaseBasicInformationMessage

		err := protojson.Unmarshal([]byte(testInstructionsInTestCase.testInstructionsAsJsonb), &testInstructions)
		err := protojson.Unmarshal([]byte(testInstructionsInTestCase.testInstructionsAsJsonb), &testInstructionContainers)
		err = protojson.Unmarshal([]byte(testInstructionsInTestCase.testCaseBasicInformationAsJsonb), &testCaseBasicInformationMessage)

		// Generate TestCaseElementModel-map
		testCaseElementModelMap := make(map[string]*fenixTestCaseBuilderServerGrpcApi.MatureTestCaseModelElementMessage)
		for _, testCaseModelElement := range testCaseBasicInformationMessage.TestCaseModel.TestCaseModelElements{
			testCaseElementModelMap[testCaseModelElement.MatureElementUuid] = testCaseModelElement
		}

		// Generate TestCaseTestInstruction-map
		testInstructionContainerMap := make(map[string]*fenixTestCaseBuilderServerGrpcApi.MatureTestInstructionContainerMessage)
		for _, testCaseModelElement := range testCaseBasicInformationMessage.TestCaseModel.TestCaseModelElements{
			testCaseElementModelMap[testCaseModelElement.MatureElementUuid] = testCaseModelElement
		}



		calculateExecutionOrder :=  fenixExecutionServerObject.recursiveExecutionOrderCalculator(testCaseBasicInformationMessage.TestCaseModel.FirstMatureElementUuid, testCaseElementModelMap)


		// Loop all TestInstructions in TestCase and add them
		for _, testInstruction := range testInstructions.MatureTestInstructions {

			calculateExecutionOrder := testCaseBasicInformationMessage.TestCaseModel.

			dataRowToBeInsertedMultiType = nil

			dataRowToBeInsertedMultiType = append(dataRowToBeInsertedMultiType, testInstructionsInTestCase.domainUuid)
			dataRowToBeInsertedMultiType = append(dataRowToBeInsertedMultiType, testInstructionsInTestCase.domainName)
			dataRowToBeInsertedMultiType = append(dataRowToBeInsertedMultiType, uuidGenerator.New().String()) //TestInstructionExecutionUuid
			dataRowToBeInsertedMultiType = append(dataRowToBeInsertedMultiType, testInstruction.MatureTestInstructionInformation.MatureBasicTestInstructionInformation.TestInstructionMatureUuid)
			dataRowToBeInsertedMultiType = append(dataRowToBeInsertedMultiType, testInstruction.BasicTestInstructionInformation.NonEditableInformation.TestInstructionOriginalName)
			dataRowToBeInsertedMultiType = append(dataRowToBeInsertedMultiType, testInstruction.BasicTestInstructionInformation.NonEditableInformation.MajorVersionNumber)
			dataRowToBeInsertedMultiType = append(dataRowToBeInsertedMultiType, testInstruction.BasicTestInstructionInformation.NonEditableInformation.MinorVersionNumber)
			dataRowToBeInsertedMultiType = append(dataRowToBeInsertedMultiType, currentDataTimeStamp)
			dataRowToBeInsertedMultiType = append(dataRowToBeInsertedMultiType, testCaseExecutionQueueMessagesMap[testInstructionsInTestCase.testCaseUuid].executionPriority)
			dataRowToBeInsertedMultiType = append(dataRowToBeInsertedMultiType, testCaseExecutionQueueMessagesMap[testInstructionsInTestCase.testCaseUuid].testCaseExecutionUuid)
			dataRowToBeInsertedMultiType = append(dataRowToBeInsertedMultiType, testCaseExecutionQueueMessagesMap[testInstructionsInTestCase.testCaseUuid].testDataSetUuid)
			dataRowToBeInsertedMultiType = append(dataRowToBeInsertedMultiType, testCaseExecutionQueueMessagesMap[testInstructionsInTestCase.testCaseUuid].testCaseExecutionVersion)
			dataRowToBeInsertedMultiType = append(dataRowToBeInsertedMultiType, 1)
			dataRowToBeInsertedMultiType = append(dataRowToBeInsertedMultiType, executionOrder)
			dataRowToBeInsertedMultiType = append(dataRowToBeInsertedMultiType, testInstruction.BasicTestInstructionInformation.NonEditableInformation.TestInstructionOrignalUuid)

			dataRowsToBeInsertedMultiType = append(dataRowsToBeInsertedMultiType, dataRowToBeInsertedMultiType)
		}
	}

	sqlToExecute = sqlToExecute + "INSERT INTO \"" + usedDBSchema + "\".\"TestCasesUnderExecution\" "
	sqlToExecute = sqlToExecute + "(\"DomainUuid\", \"DomainName\", \"TestSuiteUuid\", \"TestSuiteName\", \"TestSuiteVersion\", " +
		"\"TestSuiteExecutionUuid\", \"TestSuiteExecutionVersion\", \"TestCaseUuid\", \"TestCaseName\", \"TestCaseVersion\"," +
		" \"TestCaseExecutionUuid\", \"TestCaseExecutionVersion\", \"QueueTimeStamp\", \"TestDataSetUuid\", \"ExecutionPriority\", " +
		"\"ExecutionStartTimeStamp\", \"ExecutionStopTimeStamp\", \"TestCaseExecutionStatus\", \"ExecutionHasFinished\") "
	sqlToExecute = sqlToExecute + fenixExecutionServerObject.generateSQLInsertValues(dataRowsToBeInsertedMultiType)
	sqlToExecute = sqlToExecute + ";"

	// Execute Query CloudDB
	comandTag, err := dbTransaction.Exec(context.Background(), sqlToExecute)

	if err != nil {
		fenixExecutionServerObject.logger.WithFields(logrus.Fields{
			"Id":           "7b2447a0-5790-47b5-af28-5f069c80c88a",
			"Error":        err,
			"sqlToExecute": sqlToExecute,
		}).Error("Something went wrong when executing SQL")

		return err
	}

	// Log response from CloudDB
	fenixExecutionServerObject.logger.WithFields(logrus.Fields{
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

// Extract ExecutionOrder for TestInstruction
func (fenixExecutionServerObject *fenixExecutionServerObjectStruct) recursiveExecutionOrderCalculator(elementsUuid string, testCaseElementModelMap map[string]*fenixTestCaseBuilderServerGrpcApi.MatureTestCaseModelElementMessage, currentExecutionOrder []int, elementUuidToFind string) (executionOrder []int, err error) {

	// Extract current element
	currentElement, existInMap := testCaseElementModelMap[elementsUuid]

	// If the element doesn't exit then there is something really wrong
	if existInMap == false {
		// This shouldn't happen
		fenixExecutionServerObject.logger.WithFields(logrus.Fields{
			"id":           "9f628356-2ea2-48a6-8e6a-546a5f97f05b",
			"elementsUuid": elementsUuid,
		}).Error(elementsUuid + " could not be found in in map 'testCaseElementModelMap'")

		err = errors.New(elementsUuid + " could not be found in in map 'testCaseElementModelMap'")

		return []int{}, err
	}

	// Check if parent TestInstructionContainer is executing in parallell or in serial
	var parentTestContainerExecutesInParallell bool //TODO FIX THE CHECK

	// Element has child-element then go that path
	if currentElement.FirstChildElementUuid != elementsUuid {



		executionOrder, err = fenixExecutionServerObject.recursiveExecutionOrderCalculator(currentElement.FirstChildElementUuid, testCaseElementModelMap, currentExecutionOrder , elementUuidToFind)
	}

	// If we got an error back then something wrong happen, so just back out
	if err != nil {
		return []int{}, err
	}

	// If element has a next-element the go that path
	if currentElement.NextElementUuid != elementsUuid {

		// Check if parent TestInstructionContainer executes in Serial or in Parallell
		if (currentElement.TestCaseModelElementType == fenixTestCaseBuilderServerGrpcApi.TestCaseModelElementTypeEnum_TI_TESTINSTRUCTION ||
			currentElement.TestCaseModelElementType == fenixTestCaseBuilderServerGrpcApi.TestCaseModelElementTypeEnum_TIx_TESTINSTRUCTION_NONE_REMOVABLE ||
			currentElement.TestCaseModelElementType == fenixTestCaseBuilderServerGrpcApi.TestCaseModelElementTypeEnum_TIC_TESTINSTRUCTIONCONTAINER ||
			currentElement.TestCaseModelElementType == fenixTestCaseBuilderServerGrpcApi.TestCaseModelElementTypeEnum_TICx_TESTINSTRUCTIONCONTAINER_NONE_REMOVABLE) &&
			parentTestContainerExecutesInParallell == false {
			// Is Serial processed
			lastPositionValue := currentExecutionOrder[len(currentExecutionOrder)-1]
			lastPositionValue = lastPositionValue + 1
			currentExecutionOrder[len(currentExecutionOrder)-1] = lastPositionValue
		}

		executionOrder, err = fenixExecutionServerObject.recursiveExecutionOrderCalculator(currentElement.NextElementUuid, testCaseElementModelMap, currentExecutionOrder , elementUuidToFind)
	}

	// If we got an error back then something wrong happen, so just back out
	if err != nil {
		return []int{}, err
	}


	err = errors.New(fmt.Sprintf("Couldn't find the Element that we were looking for, %s", elementUuidToFind))
	return []int{}, err
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
