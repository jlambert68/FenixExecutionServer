package testInstructionExecutionEngine

import (
	"FenixExecutionServer/common_config"
	"context"
	"github.com/jackc/pgx/v4"
	fenixExecutionServerGrpcApi "github.com/jlambert68/FenixGrpcApi/FenixExecutionServer/fenixExecutionServerGrpcApi/go_grpc_api"
	fenixSyncShared "github.com/jlambert68/FenixSyncShared"
	"github.com/sirupsen/logrus"
	"time"
)

// After all stuff is done, then Commit or Rollback depending on result
var doCommitNotRoleBack bool

func (executionEngine *TestInstructionExecutionEngineStruct) commitOrRoleBack(dbTransaction pgx.Tx) {
	if doCommitNotRoleBack == true {
		dbTransaction.Commit(context.Background())
	} else {
		dbTransaction.Rollback(context.Background())
	}
}

// Prepare for Saving the ongoing Execution of a new TestCaseExecution in the CloudDB
func (executionEngine *TestInstructionExecutionEngineStruct) prepareInitiateExecutionsForTestInstructionsOnExecutionQueueSaveToCloudDB() {

	// Begin SQL Transaction
	txn, err := fenixSyncShared.DbPool.Begin(context.Background())
	if err != nil {
		executionEngine.logger.WithFields(logrus.Fields{
			"id":    "2effe457-d6b4-47d6-989c-5b4107e52077",
			"error": err,
		}).Error("Problem to do 'DbPool.Begin'  in 'prepareInitiateExecutionsForTestInstructionsOnExecutionQueueSaveToCloudDB'")

		return
	}
	// Standard is to do a Rollback
	doCommitNotRoleBack = false
	defer executionEngine.commitOrRoleBack(txn)

	// Generate a new TestCaseExecution-UUID
	//testCaseExecutionUuid := uuidGenerator.New().String()

	// Generate TimeStamp
	//placedOnTestExecutionQueueTimeStamp := time.Now()

	// Extract TestCaseExecutionQueue-messages to be added to data for ongoing Executions
	testInstructionExecutionQueueMessages, err := executionEngine.loadTestInstructionExecutionQueueMessages() //(txn)
	if err != nil {

		return
	}

	// If there are no TestInstructions on Queue the exit
	if testInstructionExecutionQueueMessages == nil {

		return
	}

	// Save the Initiation of a new TestInstructionsExecution in the table for ongoing TestInstruction-executions CloudDB
	err = executionEngine.saveTestInstructionsInOngoingExecutionsSaveToCloudDB(txn, testInstructionExecutionQueueMessages)
	if err != nil {

		executionEngine.logger.WithFields(logrus.Fields{
			"id":    "6c19d0c5-79b3-4d3a-a359-c5e9f3feac12",
			"error": err,
		}).Error("Couldn't Save TestInstructionsExecutionQueueMessages to queue for ongoing executions in CloudDB")

		return

	}

	// Delete messages in TestInstructionsExecutionQueue that has been put to ongoing executions
	err = executionEngine.clearTestInstructionExecutionQueueSaveToCloudDB(txn, testInstructionExecutionQueueMessages)
	if err != nil {

		executionEngine.logger.WithFields(logrus.Fields{
			"id":    "8008cb96-cc39-4d43-9948-0246ef7d5aee",
			"error": err,
		}).Error("Couldn't clear TestInstructionExecutionQueue in CloudDB")

		// Rollback any SQL transactions
		txn.Rollback(context.Background())

		return

	}

	// Commit every database change
	doCommitNotRoleBack = true

	return
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
	queueTimeStamp                    time.Time
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

// Load TestCaseExecutionQueue-Messages be able to populate the ongoing TestCaseExecution-table
func (executionEngine *TestInstructionExecutionEngineStruct) loadTestInstructionExecutionQueueMessages() (testInstructionExecutionQueueInformation []*tempTestInstructionExecutionQueueInformationStruct, err error) {

	usedDBSchema := "FenixExecution" // TODO should this env variable be used? fenixSyncShared.GetDBSchemaName()

	sqlToExecute := ""
	sqlToExecute = sqlToExecute + "SELECT  DISTINCT ON (TIEQ.\"ExecutionPriority\", TIEQ.\"TestCaseExecutionUuid\") "
	sqlToExecute = sqlToExecute + "TIEQ.* "
	sqlToExecute = sqlToExecute + "FROM \"" + usedDBSchema + "\".\"TestInstructionExecutionQueue\" TIEQ "
	sqlToExecute = sqlToExecute + "ORDER BY TIEQ.\"ExecutionPriority\" ASC, TIEQ.\"TestCaseExecutionUuid\" ASC, TIEQ.\"TestInstructionExecutionOrder\" ASC, TIEQ.\"QueueTimeStamp\" ASC; "

	// Query DB
	// Execute Query CloudDB
	//TODO change so we use the dbTransaction instead so rows will be locked ----- comandTag, err := dbTransaction.Exec(context.Background(), sqlToExecute)
	rows, err := fenixSyncShared.DbPool.Query(context.Background(), sqlToExecute)

	if err != nil {
		executionEngine.logger.WithFields(logrus.Fields{
			"Id":           "e293caef-e705-45c6-b0e5-a8916847a502",
			"Error":        err,
			"sqlToExecute": sqlToExecute,
		}).Error("Something went wrong when executing SQL")

		return []*tempTestInstructionExecutionQueueInformationStruct{}, err
	}

	// Extract data from DB result set
	for rows.Next() {

		var tempTestInstructionExecutionQueueMessage tempTestInstructionExecutionQueueInformationStruct

		err := rows.Scan(
			&tempTestInstructionExecutionQueueMessage.domainUuid,
			&tempTestInstructionExecutionQueueMessage.domainName,
			&tempTestInstructionExecutionQueueMessage.testInstructionExecutionUuid,
			&tempTestInstructionExecutionQueueMessage.testInstructionUuid,
			&tempTestInstructionExecutionQueueMessage.testInstructionName,
			&tempTestInstructionExecutionQueueMessage.testInstructionMajorVersionNumber,
			&tempTestInstructionExecutionQueueMessage.testInstructionMinorVersionNumber,
			&tempTestInstructionExecutionQueueMessage.queueTimeStamp,
			&tempTestInstructionExecutionQueueMessage.executionPriority,
			&tempTestInstructionExecutionQueueMessage.testCaseExecutionUuid,
			&tempTestInstructionExecutionQueueMessage.testDataSetUuid,
			&tempTestInstructionExecutionQueueMessage.testCaseExecutionVersion,
			&tempTestInstructionExecutionQueueMessage.testInstructionExecutionVersion,
			&tempTestInstructionExecutionQueueMessage.testInstructionExecutionOrder,
			&tempTestInstructionExecutionQueueMessage.uniqueCounter,
			&tempTestInstructionExecutionQueueMessage.testInstructionOriginalUuid,
		)

		if err != nil {

			executionEngine.logger.WithFields(logrus.Fields{
				"Id":           "a193a3b4-8130-4851-af7a-10242fb310ec",
				"Error":        err,
				"sqlToExecute": sqlToExecute,
			}).Error("Something went wrong when processing result from database")

			return []*tempTestInstructionExecutionQueueInformationStruct{}, err
		}

		// Convert

		// Add Queue-message to slice of messages
		testInstructionExecutionQueueInformation = append(testInstructionExecutionQueueInformation, &tempTestInstructionExecutionQueueMessage)

	}

	return testInstructionExecutionQueueInformation, err

}

// Put all messages found on TestCaseExecutionQueue to the ongoing executions table
func (executionEngine *TestInstructionExecutionEngineStruct) saveTestInstructionsInOngoingExecutionsSaveToCloudDB(dbTransaction pgx.Tx, testInstructionExecutionQueueMessages []*tempTestInstructionExecutionQueueInformationStruct) (err error) {

	executionEngine.logger.WithFields(logrus.Fields{
		"Id": "45351ff1-fd27-47ca-a5fb-91c85f94c535",
	}).Debug("Entering: saveTestInstructionsInOngoingExecutionsSaveToCloudDB()")

	defer func() {
		executionEngine.logger.WithFields(logrus.Fields{
			"Id": "77044809-dc75-49b2-a8e1-65340cdc07eb",
		}).Debug("Exiting: saveTestInstructionsInOngoingExecutionsSaveToCloudDB()")
	}()

	// Get a common dateTimeStamp to use
	currentDataTimeStamp := fenixSyncShared.GenerateDatetimeTimeStampForDB()

	var dataRowToBeInsertedMultiType []interface{}
	var dataRowsToBeInsertedMultiType [][]interface{}

	usedDBSchema := "FenixExecution" // TODO should this env variable be used? fenixSyncShared.GetDBSchemaName()

	sqlToExecute := ""

	// Create Insert Statement for Ongoing TestInstructionExecution
	// Data to be inserted in the DB-table
	dataRowsToBeInsertedMultiType = nil

	for _, testCaseExecutionQueueMessage := range testInstructionExecutionQueueMessages {

		dataRowToBeInsertedMultiType = nil

		dataRowToBeInsertedMultiType = append(dataRowToBeInsertedMultiType, testCaseExecutionQueueMessage.domainUuid)
		dataRowToBeInsertedMultiType = append(dataRowToBeInsertedMultiType, testCaseExecutionQueueMessage.domainName)
		dataRowToBeInsertedMultiType = append(dataRowToBeInsertedMultiType, testCaseExecutionQueueMessage.testInstructionExecutionUuid)
		dataRowToBeInsertedMultiType = append(dataRowToBeInsertedMultiType, testCaseExecutionQueueMessage.testInstructionUuid)
		dataRowToBeInsertedMultiType = append(dataRowToBeInsertedMultiType, testCaseExecutionQueueMessage.testInstructionName)
		dataRowToBeInsertedMultiType = append(dataRowToBeInsertedMultiType, testCaseExecutionQueueMessage.testInstructionMajorVersionNumber)
		dataRowToBeInsertedMultiType = append(dataRowToBeInsertedMultiType, testCaseExecutionQueueMessage.testInstructionMinorVersionNumber)
		dataRowToBeInsertedMultiType = append(dataRowToBeInsertedMultiType, currentDataTimeStamp) //SentTimeStamp
		dataRowToBeInsertedMultiType = append(dataRowToBeInsertedMultiType, int(fenixExecutionServerGrpcApi.TestInstructionExecutionStatusEnum_TIE_INITIATED))
		dataRowToBeInsertedMultiType = append(dataRowToBeInsertedMultiType, currentDataTimeStamp) // ExecutionStatusUpdateTimeStamp
		dataRowToBeInsertedMultiType = append(dataRowToBeInsertedMultiType, testCaseExecutionQueueMessage.testDataSetUuid)
		dataRowToBeInsertedMultiType = append(dataRowToBeInsertedMultiType, testCaseExecutionQueueMessage.testCaseExecutionUuid)
		dataRowToBeInsertedMultiType = append(dataRowToBeInsertedMultiType, testCaseExecutionQueueMessage.testCaseExecutionVersion)
		dataRowToBeInsertedMultiType = append(dataRowToBeInsertedMultiType, testCaseExecutionQueueMessage.testInstructionExecutionVersion)
		dataRowToBeInsertedMultiType = append(dataRowToBeInsertedMultiType, testCaseExecutionQueueMessage.testInstructionExecutionOrder)
		dataRowToBeInsertedMultiType = append(dataRowToBeInsertedMultiType, testCaseExecutionQueueMessage.testInstructionOriginalUuid)
		dataRowToBeInsertedMultiType = append(dataRowToBeInsertedMultiType, false) // TestInstructionExecutionHasFinished

		dataRowsToBeInsertedMultiType = append(dataRowsToBeInsertedMultiType, dataRowToBeInsertedMultiType)

	}

	sqlToExecute = sqlToExecute + "INSERT INTO \"" + usedDBSchema + "\".\"TestInstructionsUnderExecution\" "
	sqlToExecute = sqlToExecute + "(\"DomainUuid\", \"DomainName\", \"TestInstructionExecutionUuid\", \"TestInstructionUuid\", \"TestInstructionName\", " +
		"\"TestInstructionMajorVersionNumber\", \"TestInstructionMinorVersionNumber\", \"SentTimeStamp\", \"TestInstructionExecutionStatus\", \"ExecutionStatusUpdateTimeStamp\", " +
		" \"TestDataSetUuid\", \"TestCaseExecutionUuid\", \"TestCaseExecutionVersion\", \"TestInstructionInstructionExecutionVersion\", \"TestInstructionExecutionOrder\", " +
		"\"TestInstructionOriginalUuid\", \"TestInstructionExecutionHasFinished\") "
	sqlToExecute = sqlToExecute + common_config.GenerateSQLInsertValues(dataRowsToBeInsertedMultiType)
	sqlToExecute = sqlToExecute + ";"

	// Execute Query CloudDB
	comandTag, err := dbTransaction.Exec(context.Background(), sqlToExecute)

	if err != nil {
		executionEngine.logger.WithFields(logrus.Fields{
			"Id":           "7b2447a0-5790-47b5-af28-5f069c80c88a",
			"Error":        err,
			"sqlToExecute": sqlToExecute,
		}).Error("Something went wrong when executing SQL")

		return err
	}

	// Log response from CloudDB
	executionEngine.logger.WithFields(logrus.Fields{
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
func (executionEngine *TestInstructionExecutionEngineStruct) clearTestInstructionExecutionQueueSaveToCloudDB(dbTransaction pgx.Tx, testInstructionExecutionQueueMessages []*tempTestInstructionExecutionQueueInformationStruct) (err error) {

	executionEngine.logger.WithFields(logrus.Fields{
		"Id": "536680e1-c646-4005-abd2-5870ffc9b634",
	}).Debug("Entering: clearTestInstructionExecutionQueueSaveToCloudDB()")

	defer func() {
		executionEngine.logger.WithFields(logrus.Fields{
			"Id": "a33f05c7-588d-4eb2-a893-f7bb146d7b10",
		}).Debug("Exiting: clearTestInstructionExecutionQueueSaveToCloudDB()")
	}()

	var testInstructionExecutionsToBeDeletedFromQueue []int

	// Loop over TestCaseExecutionQueue-messages and extract  "UniqueCounter"
	for _, testInstructionExecutionQueueMessage := range testInstructionExecutionQueueMessages {
		testInstructionExecutionsToBeDeletedFromQueue = append(testInstructionExecutionsToBeDeletedFromQueue, testInstructionExecutionQueueMessage.uniqueCounter)
	}

	usedDBSchema := "FenixExecution" // TODO should this env variable be used? fenixSyncShared.GetDBSchemaName()

	sqlToExecute := ""

	sqlToExecute = sqlToExecute + "DELETE FROM \"" + usedDBSchema + "\".\"TestInstructionExecutionQueue\" TIEQ "
	sqlToExecute = sqlToExecute + "WHERE TIEQ.\"UniqueCounter\" IN "
	sqlToExecute = sqlToExecute + common_config.GenerateSQLINIntegerArray(testInstructionExecutionsToBeDeletedFromQueue)
	sqlToExecute = sqlToExecute + ";"

	// Execute Query CloudDB
	comandTag, err := dbTransaction.Exec(context.Background(), sqlToExecute)

	if err != nil {
		executionEngine.logger.WithFields(logrus.Fields{
			"Id":           "357fc125-1444-4c8b-947c-39af4440cfa9",
			"Error":        err,
			"sqlToExecute": sqlToExecute,
		}).Error("Something went wrong when executing SQL")

		return err
	}

	// Log response from CloudDB
	executionEngine.logger.WithFields(logrus.Fields{
		"Id":                       "a4464842-77c5-444b-997e-41012499f8bc",
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
