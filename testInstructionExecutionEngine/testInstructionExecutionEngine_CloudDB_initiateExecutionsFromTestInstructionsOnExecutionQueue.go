package testInstructionExecutionEngine

import (
	"FenixExecutionServer/common_config"
	"context"
	"github.com/jackc/pgx/v4"
	fenixExecutionServerGrpcApi "github.com/jlambert68/FenixGrpcApi/FenixExecutionServer/fenixExecutionServerGrpcApi/go_grpc_api"
	fenixSyncShared "github.com/jlambert68/FenixSyncShared"
	"github.com/sirupsen/logrus"
	"sort"
	"strconv"
	"strings"
	"time"
)

func (executionEngine *TestInstructionExecutionEngineStruct) prepareInitiateExecutionsForTestInstructionsOnExecutionQueueSaveToCloudDBCommitOrRoleBackParallellSave(
	executionTrackNumberReference *int,
	dbTransactionReference *pgx.Tx,
	doCommitNotRoleBackReference *bool,
	testCaseExecutionsToProcessReference *[]ChannelCommandTestCaseExecutionStruct,
	updateTestCaseExecutionWithStatusReference *bool) {

	executionTrackNumber := *executionTrackNumberReference
	dbTransaction := *dbTransactionReference
	doCommitNotRoleBack := *doCommitNotRoleBackReference
	testCaseExecutionsToProcess := *testCaseExecutionsToProcessReference
	updateTestCaseExecutionWithStatus := *updateTestCaseExecutionWithStatusReference

	if doCommitNotRoleBack == true || updateTestCaseExecutionWithStatus == true {

		if doCommitNotRoleBack == true {

			dbTransaction.Commit(context.Background())
		}

		// Only trigger 'To send TestInstructions' when there are were some waiting on TestInstructionExecutionQueue
		if updateTestCaseExecutionWithStatus == false {

			// Trigger TestInstructionEngine to check if there are TestInstructionExecutions, in under Execution, waiting to be sent to Worker
			channelCommandMessage := ChannelCommandStruct{
				ChannelCommand:                   ChannelCommandCheckForTestInstructionExecutionsWaitingToBeSentToWorker,
				ChannelCommandTestCaseExecutions: testCaseExecutionsToProcess,
			}

			// Send Message on Channel
			*executionEngine.CommandChannelReferenceSlice[executionTrackNumber] <- channelCommandMessage
		} else {
			// No TestInstructionExecutions are waiting in ExecutionQueue, updateTestCaseExecutionWithStatus == true

			dbTransaction.Rollback(context.Background())

			// Trigger TestInstructionEngine to update TestCaseExecutionUuid based on all finished individual TestInstructionExecutions
			channelCommandMessage := ChannelCommandStruct{
				ChannelCommand:                   ChannelCommandUpdateExecutionStatusOnTestCaseExecutionExecutions,
				ChannelCommandTestCaseExecutions: testCaseExecutionsToProcess,
			}

			// Send Message on Channel
			*executionEngine.CommandChannelReferenceSlice[executionTrackNumber] <- channelCommandMessage
		}

	} else {
		dbTransaction.Rollback(context.Background())
	}
}

// Prepare for Saving the ongoing Execution of a new TestCaseExecutionUuid in the CloudDB
func (executionEngine *TestInstructionExecutionEngineStruct) moveTestInstructionExecutionsFromExecutionQueueToOngoingExecutionsSaveToCloudDB(
	executionTrackNumber int,
	testCaseExecutionsToProcess []ChannelCommandTestCaseExecutionStruct) {

	executionEngine.logger.WithFields(logrus.Fields{
		"id":                          "3bd9e5cf-d108-4d99-94fa-8dc673dfcb68",
		"testCaseExecutionsToProcess": testCaseExecutionsToProcess,
	}).Debug("Incoming 'moveTestInstructionExecutionsFromExecutionQueueToOngoingExecutionsSaveToCloudDB'")

	defer executionEngine.logger.WithFields(logrus.Fields{
		"id": "9e9c1445-f805-4e71-bff0-0ea60327a254",
	}).Debug("Outgoing 'moveTestInstructionExecutionsFromExecutionQueueToOngoingExecutionsSaveToCloudDB'")

	// After all stuff is done, then Commit or Rollback depending on result
	var doCommitNotRoleBack bool

	// After all stuff is done, this one is used to decide if the TestCaseExecutionUuid status should be updated or not
	var triggerUpdateTestCaseExecutionWithStatus bool

	// Begin SQL Transaction
	txn, err := fenixSyncShared.DbPool.Begin(context.Background())
	if err != nil {
		executionEngine.logger.WithFields(logrus.Fields{
			"id":    "2effe457-d6b4-47d6-989c-5b4107e52077",
			"error": err,
		}).Error("Problem to do 'DbPool.Begin'  in 'moveTestInstructionExecutionsFromExecutionQueueToOngoingExecutionsSaveToCloudDB'")

		return
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

	// Standard is to do a Rollback
	doCommitNotRoleBack = false

	// Standard is not to update TestCaseExecutionUuid with Execution Status from TestInstructionExecutions
	triggerUpdateTestCaseExecutionWithStatus = false

	defer executionEngine.prepareInitiateExecutionsForTestInstructionsOnExecutionQueueSaveToCloudDBCommitOrRoleBackParallellSave(
		&executionTrackNumber,
		&txn,
		&doCommitNotRoleBack,
		&testCaseExecutionsToProcess,
		&triggerUpdateTestCaseExecutionWithStatus)

	// Generate a new TestCaseExecutionUuid-UUID
	//testCaseExecutionUuid := uuidGenerator.New().String()

	// Generate TimeStamp
	//placedOnTestExecutionQueueTimeStamp := time.Now()

	// Extract TestCaseExecutionQueue-messages to be added to data for ongoing Executions
	testInstructionExecutionQueueMessages, err := executionEngine.loadTestInstructionExecutionQueueMessages(
		txn, testCaseExecutionsToProcess) //(txn)
	if err != nil {

		return
	}

	// If there are no TestInstructions on Queue the exit
	if testInstructionExecutionQueueMessages == nil {

		// Set 'testCaseExecutionsToProcess' to nil to inform Defered function not to trigger SendToWorker
		triggerUpdateTestCaseExecutionWithStatus = true
		//testCaseExecutionsToProcess = []ChannelCommandTestCaseExecutionStruct{}

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

		return

	}

	// Commit every database change
	doCommitNotRoleBack = true

	return
}

/*
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
	matureTestInstructionUuid               string
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
*/
// Load TestCaseExecutionQueue-Messages be able to populate the ongoing TestCaseExecutionUuid-table
func (executionEngine *TestInstructionExecutionEngineStruct) loadTestInstructionExecutionQueueMessages(
	dbTransaction pgx.Tx,
	testCaseExecutionsToProcess []ChannelCommandTestCaseExecutionStruct) (
	testInstructionExecutionQueueInformation []*tempTestInstructionExecutionQueueInformationStruct, err error) {

	usedDBSchema := "FenixExecution" // TODO should this env variable be used? fenixSyncShared.GetDBSchemaName()

	// Generate WHERE-values to only target correct 'TestCaseExecutionUuid' together with 'TestCaseExecutionVersion'
	var correctTestCaseExecutionUuidAndTestCaseExecutionVersionPar string
	var correctTestCaseExecutionUuidAndTestCaseExecutionVersionPars string
	for testCaseExecutionCounter, testCaseExecution := range testCaseExecutionsToProcess {
		correctTestCaseExecutionUuidAndTestCaseExecutionVersionPar =
			"(TIEQ.\"TestCaseExecutionUuid\" = '" + testCaseExecution.TestCaseExecutionUuid + "' AND " +
				"TIEQ.\"TestCaseExecutionVersion\" = " + strconv.Itoa(int(testCaseExecution.TestCaseExecutionVersion)) + ") "

		switch testCaseExecutionCounter {
		case 0:
			// When this is the first then we need to add 'AND before'
			// *NOT NEEDED* in this query
			// correctTestCaseExecutionUuidAndTestCaseExecutionVersionPars = "AND "

		default:
			// When this is not the first then we need to add 'OR' after previous
			correctTestCaseExecutionUuidAndTestCaseExecutionVersionPars =
				correctTestCaseExecutionUuidAndTestCaseExecutionVersionPars + "OR "
		}

		// Add the WHERE-values
		correctTestCaseExecutionUuidAndTestCaseExecutionVersionPars =
			correctTestCaseExecutionUuidAndTestCaseExecutionVersionPars + correctTestCaseExecutionUuidAndTestCaseExecutionVersionPar

	}

	// Create 2nd SQLs WHERE-values from first part
	var correctTestCaseExecutionUuidAndTestCaseExecutionVersionParsSecondPart string
	correctTestCaseExecutionUuidAndTestCaseExecutionVersionParsSecondPart = strings.ReplaceAll(
		correctTestCaseExecutionUuidAndTestCaseExecutionVersionPars, "TIEQ", "TIEQ2")

	/*
		sqlToExecute := ""
		sqlToExecute = sqlToExecute + "SELECT  DISTINCT ON (TIEQ.\"ExecutionPriority\", TIEQ.\"TestCaseExecutionUuid\", TIEQ.\"TestInstructionExecutionOrder\") "
		sqlToExecute = sqlToExecute + "TIEQ.* "
		sqlToExecute = sqlToExecute + "FROM \"" + usedDBSchema + "\".\"TestInstructionExecutionQueue\" TIEQ "
		sqlToExecute = sqlToExecute + "WHERE "
		sqlToExecute = sqlToExecute + correctTestCaseExecutionUuidAndTestCaseExecutionVersionPars
		sqlToExecute = sqlToExecute + "ORDER BY TIEQ.\"ExecutionPriority\" ASC, TIEQ.\"TestCaseExecutionUuid\" ASC, TIEQ.\"TestInstructionExecutionOrder\" ASC, TIEQ.\"QueueTimeStamp\" ASC; "
	*/
	sqlToExecute := ""
	sqlToExecute = sqlToExecute + "SELECT TIEQ.* "
	sqlToExecute = sqlToExecute + "FROM \"" + usedDBSchema + "\".\"TestInstructionExecutionQueue\" TIEQ "
	sqlToExecute = sqlToExecute + "WHERE "
	sqlToExecute = sqlToExecute + correctTestCaseExecutionUuidAndTestCaseExecutionVersionPars
	sqlToExecute = sqlToExecute + "AND "
	sqlToExecute = sqlToExecute + "TIEQ.\"TestInstructionExecutionOrder\" =  (SELECT DISTINCT ON (TIEQ2.\"ExecutionPriority\", TIEQ2.\"TestCaseExecutionUuid\") TIEQ2.\"TestInstructionExecutionOrder\" "
	sqlToExecute = sqlToExecute + "FROM \"FenixExecution\".\"TestInstructionExecutionQueue\" TIEQ2 "
	sqlToExecute = sqlToExecute + "WHERE "
	sqlToExecute = sqlToExecute + correctTestCaseExecutionUuidAndTestCaseExecutionVersionParsSecondPart
	sqlToExecute = sqlToExecute + "ORDER BY TIEQ2.\"ExecutionPriority\" ASC, TIEQ2.\"TestCaseExecutionUuid\" ASC, TIEQ2.\"TestInstructionExecutionOrder\" ASC, TIEQ2.\"QueueTimeStamp\" ASC) "
	sqlToExecute = sqlToExecute + "ORDER BY TIEQ.\"ExecutionPriority\" ASC, TIEQ.\"TestCaseExecutionUuid\" ASC, TIEQ.\"TestInstructionExecutionOrder\" ASC, TIEQ.\"QueueTimeStamp\" ASC; "

	// Log SQL to be executed if Environment variable is true
	if common_config.LogAllSQLs == true {
		common_config.Logger.WithFields(logrus.Fields{
			"Id":           "6ff37f82-f925-48fa-97ac-7ea379b36585",
			"sqlToExecute": sqlToExecute,
		}).Debug("SQL to be executed within 'loadTestInstructionExecutionQueueMessages'")
	}

	// Query DB
	var ctx context.Context
	ctx, timeOutCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer timeOutCancel()

	rows, err := dbTransaction.Query(ctx, sqlToExecute)
	defer rows.Close()

	if err != nil {
		executionEngine.logger.WithFields(logrus.Fields{
			"Id":           "e293caef-e705-45c6-b0e5-a8916847a502",
			"Error":        err,
			"sqlToExecute": sqlToExecute,
		}).Error("Something went wrong when executing SQL")

		return []*tempTestInstructionExecutionQueueInformationStruct{}, err
	}

	// Temporary variables
	var tempQueueTimeStamp time.Time

	// Extract data from DB result set
	for rows.Next() {

		var tempTestInstructionExecutionQueueMessage tempTestInstructionExecutionQueueInformationStruct

		err := rows.Scan(
			&tempTestInstructionExecutionQueueMessage.domainUuid,
			&tempTestInstructionExecutionQueueMessage.domainName,
			&tempTestInstructionExecutionQueueMessage.testInstructionExecutionUuid,
			&tempTestInstructionExecutionQueueMessage.matureTestInstructionUuid,
			&tempTestInstructionExecutionQueueMessage.testInstructionName,
			&tempTestInstructionExecutionQueueMessage.testInstructionMajorVersionNumber,
			&tempTestInstructionExecutionQueueMessage.testInstructionMinorVersionNumber,
			&tempQueueTimeStamp, //&tempTestInstructionExecutionQueueMessage.queueTimeStamp,
			&tempTestInstructionExecutionQueueMessage.executionPriority,
			&tempTestInstructionExecutionQueueMessage.testCaseExecutionUuid,
			&tempTestInstructionExecutionQueueMessage.testDataSetUuid,
			&tempTestInstructionExecutionQueueMessage.testCaseExecutionVersion,
			&tempTestInstructionExecutionQueueMessage.testInstructionExecutionVersion,
			&tempTestInstructionExecutionQueueMessage.testInstructionExecutionOrder,
			&tempTestInstructionExecutionQueueMessage.uniqueCounter,
			&tempTestInstructionExecutionQueueMessage.testInstructionOriginalUuid,
			&tempTestInstructionExecutionQueueMessage.executionStatusReportLevel,
			&tempTestInstructionExecutionQueueMessage.executionDomainUuid,
			&tempTestInstructionExecutionQueueMessage.executionDomainName,
		)

		if err != nil {

			executionEngine.logger.WithFields(logrus.Fields{
				"Id":           "a193a3b4-8130-4851-af7a-10242fb310ec",
				"Error":        err,
				"sqlToExecute": sqlToExecute,
			}).Error("Something went wrong when processing result from database")

			return []*tempTestInstructionExecutionQueueInformationStruct{}, err
		}

		// Convert tempQueueTimeStamp -> tempTestInstructionExecutionQueueMessage.queueTimeStamp
		tempTestInstructionExecutionQueueMessage.queueTimeStamp = common_config.GenerateDatetimeFromTimeInputForDB(tempQueueTimeStamp)

		// Add Queue-message to slice of messages
		testInstructionExecutionQueueInformation = append(testInstructionExecutionQueueInformation, &tempTestInstructionExecutionQueueMessage)

	}

	return testInstructionExecutionQueueInformation, err

}

// Put all messages found on TestCaseExecutionQueue to the ongoing executions table
func (executionEngine *TestInstructionExecutionEngineStruct) saveTestInstructionsInOngoingExecutionsSaveToCloudDB(dbTransaction pgx.Tx, testInstructionExecutionQueueMessages []*tempTestInstructionExecutionQueueInformationStruct) (err error) {

	executionEngine.logger.WithFields(logrus.Fields{
		"Id":                                    "45351ff1-fd27-47ca-a5fb-91c85f94c535",
		"testInstructionExecutionQueueMessages": testInstructionExecutionQueueMessages,
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

	for _, testInstructionExecutionQueueMessage := range testInstructionExecutionQueueMessages {

		dataRowToBeInsertedMultiType = nil

		dataRowToBeInsertedMultiType = append(dataRowToBeInsertedMultiType, testInstructionExecutionQueueMessage.domainUuid)
		dataRowToBeInsertedMultiType = append(dataRowToBeInsertedMultiType, testInstructionExecutionQueueMessage.domainName)
		dataRowToBeInsertedMultiType = append(dataRowToBeInsertedMultiType, testInstructionExecutionQueueMessage.testInstructionExecutionUuid)
		dataRowToBeInsertedMultiType = append(dataRowToBeInsertedMultiType, testInstructionExecutionQueueMessage.matureTestInstructionUuid)
		dataRowToBeInsertedMultiType = append(dataRowToBeInsertedMultiType, testInstructionExecutionQueueMessage.testInstructionName)
		dataRowToBeInsertedMultiType = append(dataRowToBeInsertedMultiType, testInstructionExecutionQueueMessage.testInstructionMajorVersionNumber)
		dataRowToBeInsertedMultiType = append(dataRowToBeInsertedMultiType, testInstructionExecutionQueueMessage.testInstructionMinorVersionNumber)
		dataRowToBeInsertedMultiType = append(dataRowToBeInsertedMultiType, currentDataTimeStamp) //SentTimeStamp
		dataRowToBeInsertedMultiType = append(dataRowToBeInsertedMultiType, int(fenixExecutionServerGrpcApi.TestInstructionExecutionStatusEnum_TIE_INITIATED))
		dataRowToBeInsertedMultiType = append(dataRowToBeInsertedMultiType, currentDataTimeStamp) // ExecutionStatusUpdateTimeStamp
		dataRowToBeInsertedMultiType = append(dataRowToBeInsertedMultiType, testInstructionExecutionQueueMessage.testDataSetUuid)
		dataRowToBeInsertedMultiType = append(dataRowToBeInsertedMultiType, testInstructionExecutionQueueMessage.testCaseExecutionUuid)
		dataRowToBeInsertedMultiType = append(dataRowToBeInsertedMultiType, testInstructionExecutionQueueMessage.testCaseExecutionVersion)
		dataRowToBeInsertedMultiType = append(dataRowToBeInsertedMultiType, testInstructionExecutionQueueMessage.testInstructionExecutionVersion)
		dataRowToBeInsertedMultiType = append(dataRowToBeInsertedMultiType, testInstructionExecutionQueueMessage.testInstructionExecutionOrder)
		dataRowToBeInsertedMultiType = append(dataRowToBeInsertedMultiType, testInstructionExecutionQueueMessage.testInstructionOriginalUuid)
		dataRowToBeInsertedMultiType = append(dataRowToBeInsertedMultiType, false) // TestInstructionExecutionHasFinished
		dataRowToBeInsertedMultiType = append(dataRowToBeInsertedMultiType, testInstructionExecutionQueueMessage.queueTimeStamp)
		dataRowToBeInsertedMultiType = append(dataRowToBeInsertedMultiType, testInstructionExecutionQueueMessage.executionPriority)
		dataRowToBeInsertedMultiType = append(dataRowToBeInsertedMultiType, testInstructionExecutionQueueMessage.executionStatusReportLevel)
		dataRowToBeInsertedMultiType = append(dataRowToBeInsertedMultiType, testInstructionExecutionQueueMessage.executionDomainUuid)
		dataRowToBeInsertedMultiType = append(dataRowToBeInsertedMultiType, testInstructionExecutionQueueMessage.executionDomainName)

		dataRowsToBeInsertedMultiType = append(dataRowsToBeInsertedMultiType, dataRowToBeInsertedMultiType)

	}

	sqlToExecute = sqlToExecute + "INSERT INTO \"" + usedDBSchema + "\".\"TestInstructionsUnderExecution\" "
	sqlToExecute = sqlToExecute + "(\"DomainUuid\", \"DomainName\", \"TestInstructionExecutionUuid\", \"MatureTestInstructionUuid\", \"TestInstructionName\", " +
		"\"TestInstructionMajorVersionNumber\", \"TestInstructionMinorVersionNumber\", \"SentTimeStamp\", \"TestInstructionExecutionStatus\", \"ExecutionStatusUpdateTimeStamp\", " +
		" \"TestDataSetUuid\", \"TestCaseExecutionUuid\", \"TestCaseExecutionVersion\", \"TestInstructionInstructionExecutionVersion\", \"TestInstructionExecutionOrder\", " +
		"\"TestInstructionOriginalUuid\", \"TestInstructionExecutionHasFinished\", \"QueueTimeStamp\"," +
		" \"ExecutionPriority\", \"ExecutionStatusReportLevel\", \"ExecutionDomainUuid\", \"ExecutionDomainName\")  "
	sqlToExecute = sqlToExecute + common_config.GenerateSQLInsertValues(dataRowsToBeInsertedMultiType)
	sqlToExecute = sqlToExecute + ";"

	// Log SQL to be executed if Environment variable is true
	if common_config.LogAllSQLs == true {
		common_config.Logger.WithFields(logrus.Fields{
			"Id":           "09acc12b-f0f2-402c-bde9-206bde49e35e",
			"sqlToExecute": sqlToExecute,
		}).Debug("SQL to be executed within 'saveTestInstructionsInOngoingExecutionsSaveToCloudDB'")
	}

	// Execute Query CloudDB
	comandTag, err := dbTransaction.Exec(context.Background(), sqlToExecute)

	if err != nil {
		executionEngine.logger.WithFields(logrus.Fields{
			"Id":           "d7cd6754-cf4c-43eb-8478-d6558e787dd0",
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
		"Id":                                    "536680e1-c646-4005-abd2-5870ffc9b634",
		"testInstructionExecutionQueueMessages": testInstructionExecutionQueueMessages,
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

	// Log SQL to be executed if Environment variable is true
	if common_config.LogAllSQLs == true {
		common_config.Logger.WithFields(logrus.Fields{
			"Id":           "bcd7ac6b-f0ba-423a-bce4-debc07689104",
			"sqlToExecute": sqlToExecute,
		}).Debug("SQL to be executed within 'clearTestInstructionExecutionQueueSaveToCloudDB'")
	}

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
