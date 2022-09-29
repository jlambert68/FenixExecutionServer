package testInstructionExecutionEngine

import (
	"context"
	fenixSyncShared "github.com/jlambert68/FenixSyncShared"
	"github.com/sirupsen/logrus"
)

// Prepare for Saving the ongoing Execution of a new TestCaseExecution in the CloudDB
func (executionEngine *TestInstructionExecutionEngineStruct) sendNewTestInstructionsThatIsWaitingToBeSentWorker() {

	// After all stuff is done, then Commit or Rollback depending on result
	var doCommitNotRoleBack bool

	// Begin SQL Transaction
	txn, err := fenixSyncShared.DbPool.Begin(context.Background())
	if err != nil {
		executionEngine.logger.WithFields(logrus.Fields{
			"id":    "2effe457-d6b4-47d6-989c-5b4107e52077",
			"error": err,
		}).Error("Problem to do 'DbPool.Begin'  in 'sendNewTestInstructionsThatIsWaitingToBeSentWorker'")

		return
	}
	// Standard is to do a Rollback
	doCommitNotRoleBack = false
	defer executionEngine.commitOrRoleBackParallellSave(&txn, &doCommitNotRoleBack)

	// Generate a new TestCaseExecution-UUID
	//testCaseExecutionUuid := uuidGenerator.New().String()

	// Generate TimeStamp
	//placedOnTestExecutionQueueTimeStamp := time.Now()

	// Load all TestInstructions and their attributes to be sent to the Executions Workers over gRPC
	testInstructionToBeSentToExecutionWorkers, err := executionEngine.loadNewTestInstructionToBeSentToExecutionWorkers() //(txn)
	if err != nil {

		return
	}

	// If there are no TestInstructions  the exit
	if testInstructionToBeSentToExecutionWorkers == nil {

		return
	}

	// Update status on TestInstructions that could be sent to workers
	err = executionEngine.clearTestInstructionExecutionQueueSaveToCloudDB(txn, testInstructionExecutionQueueMessages)
	if err != nil {

		executionEngine.logger.WithFields(logrus.Fields{
			"id":    "8008cb96-cc39-4d43-9948-0246ef7d5aee",
			"error": err,
		}).Error("Couldn't clear TestInstructionExecutionQueue in CloudDB")

		return

	}

	// Update status on TestCases that TestInstructions have been sent to workers
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

// Hold one new TestInstruction to be sent to Execution Worker
type newTestInstructionToBeSentToExecutionWorkersStruct struct {
	domainUuid                        string
	domainName                        string
	executionWorkerAddress            string
	testInstructionExecutionUuid      string
	testInstructionOriginalUuid       string
	testInstructionName               string
	testInstructionMajorVersionNumber int
	testInstructionMinorVersionNumber int
	testInstructionAttributeType      int
	testInstructionAttributeUuid      string
	testInstructionAttributeName      string
	attributeValueAsString            string
	attributeValueUuid                string
}

// Load all New TestInstructions and their attributes to be sent to the Executions Workers over gRPC
func (executionEngine *TestInstructionExecutionEngineStruct) loadNewTestInstructionToBeSentToExecutionWorkers() (testInstructionsToBeSentToExecutionWorkers []newTestInstructionToBeSentToExecutionWorkersStruct, err error) {

	usedDBSchema := "FenixExecution" // TODO should this env variable be used? fenixSyncShared.GetDBSchemaName()

	sqlToExecute := ""
	sqlToExecute = sqlToExecute + "SELECT DP.\"DomainUuid\", DP.\"DomainName\", DP.\"ExecutionWorker Address\", " +
		"TIUE.\"TestInstructionExecutionUuid\", TIUE.\"TestInstructionOriginalUuid\", TIUE.\"TestInstructionName\", " +
		"TIUE.\"TestInstructionMajorVersionNumber\", TIUE.\"TestInstructionMinorVersionNumber\", " +
		"TIAUE.\"TestInstructionAttributeType\", TIAUE.\"TestInstructionAttributeUuid\", " +
		"TIAUE.\"TestInstructionAttributeName\", TIAUE.\"AttributeValueAsString\", TIAUE.\"AttributeValueUuid\" "
	sqlToExecute = sqlToExecute + "FROM \"" + usedDBSchema + "\".\"TestInstructionsUnderExecution\" TIUE, " +
		"\"" + usedDBSchema + "\".\"TestInstructionAttributesUnderExecution\" TIAUE, " +
		"\"" + usedDBSchema + "\".\"DomainParameters\" DP "
	sqlToExecute = sqlToExecute + "WHERE TIUE.\"TestInstructionExecutionStatus\" = 0 AND "
	sqlToExecute = sqlToExecute + "DP.\"DomainUuid\" = TIUE.\"DomainUuid\" AND "
	sqlToExecute = sqlToExecute + "TIUE.\"TestInstructionExecutionUuid\" = TIAUE.\"TestInstructionExecutionUuid\" "
	sqlToExecute = sqlToExecute + "ORDER BY DP.\"DomainUuid\" ASC, TIUE.\"TestInstructionExecutionUuid\" ASC, TIAUE.\"TestInstructionAttributeUuid\" ASC "
	sqlToExecute = sqlToExecute + "; "

	// Query DB
	// Execute Query CloudDB
	//TODO change so we use the dbTransaction instead so rows will be locked ----- comandTag, err := dbTransaction.Exec(context.Background(), sqlToExecute)
	rows, err := fenixSyncShared.DbPool.Query(context.Background(), sqlToExecute)

	if err != nil {
		executionEngine.logger.WithFields(logrus.Fields{
			"Id":           "ac262275-5f05-48c8-982a-46ff2392d3f2",
			"Error":        err,
			"sqlToExecute": sqlToExecute,
		}).Error("Something went wrong when executing SQL")

		return nil, err
	}

	// Extract data from DB result set
	for rows.Next() {

		var tempTestInstructionAndAttributeData newTestInstructionToBeSentToExecutionWorkersStruct

		err := rows.Scan(

			&tempTestInstructionAndAttributeData.domainUuid,
			&tempTestInstructionAndAttributeData.domainName,
			&tempTestInstructionAndAttributeData.executionWorkerAddress,
			&tempTestInstructionAndAttributeData.testInstructionExecutionUuid,
			&tempTestInstructionAndAttributeData.testInstructionOriginalUuid,
			&tempTestInstructionAndAttributeData.testInstructionName,
			&tempTestInstructionAndAttributeData.testInstructionMajorVersionNumber,
			&tempTestInstructionAndAttributeData.testInstructionMinorVersionNumber,
			&tempTestInstructionAndAttributeData.testInstructionAttributeType,
			&tempTestInstructionAndAttributeData.testInstructionAttributeUuid,
			&tempTestInstructionAndAttributeData.testInstructionAttributeName,
			&tempTestInstructionAndAttributeData.attributeValueAsString,
			&tempTestInstructionAndAttributeData.attributeValueUuid,
		)

		if err != nil {

			executionEngine.logger.WithFields(logrus.Fields{
				"Id":           "9facd51b-2292-4786-85cd-c2ebc83108e1",
				"Error":        err,
				"sqlToExecute": sqlToExecute,
			}).Error("Something went wrong when processing result from database")

			return nil, err
		}

		// Add Queue-message to slice of messages
		testInstructionsToBeSentToExecutionWorkers = append(testInstructionsToBeSentToExecutionWorkers, tempTestInstructionAndAttributeData)

	}

	return testInstructionsToBeSentToExecutionWorkers, err

}
