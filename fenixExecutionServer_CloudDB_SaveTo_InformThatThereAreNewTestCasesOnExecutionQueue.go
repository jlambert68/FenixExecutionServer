package main

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"github.com/jackc/pgx/v4"
	fenixExecutionServerGrpcApi "github.com/jlambert68/FenixGrpcApi/FenixExecutionServer/fenixExecutionServerGrpcApi/go_grpc_api"
	fenixTestCaseBuilderServerGrpcApi "github.com/jlambert68/FenixGrpcApi/FenixTestCaseBuilderServer/fenixTestCaseBuilderServerGrpcApi/go_grpc_api"
	fenixSyncShared "github.com/jlambert68/FenixSyncShared"
	"github.com/sirupsen/logrus"
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

	// Load TestInstructions to be added to TestInstructionExecutionQueue
	err = fenixExecutionServerObject.loadTestInstructionsToBeAddedToExecutionQueueLoadFromCloudDB(txn, testCaseExecutionQueueMessages)
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
	err = fenixExecutionServerObject.SaveTestInstructionsToExecutionQueueSaveToCloudDB(txn, testCaseExecutionQueueMessages)
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

	type testCaseExecutionsToBeDeletedFromQueueStruct struct {
		testCaseExecutionUuid    string
		testCaseExecutionVersion int
	}
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

/*
SELECT DISTINCT ON (TC."TestCaseUuid")
TC."DomainUuid", TC."DomainName", TC."TestCaseUuid", TC."TestCaseName", TC."TestCaseVersion", "TestInstructionsAsJsonb", TC."UniqueCounter"
FROM "FenixGuiBuilder"."TestCases" TC
WHERE TC."TestCaseUuid"  IN ('c365a9f7-e236-417b-9236-dd20b728fe64', '8c8cc470-0d16-49b2-9a89-3a4d8f81783f', '3383d2d6-a87f-4425-a26b-539349b8c6cc', '63cecc8d-a53d-4589-9cba-d2650e13b5c6')
ORDER  BY TC."TestCaseUuid" ASC, TC."TestCaseVersion" DESC ;


*/
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
