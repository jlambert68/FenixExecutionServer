package testInstructionExecutionEngine

import (
	"FenixExecutionServer/common_config"
	"context"
	fenixSyncShared "github.com/jlambert68/FenixSyncShared"
	"github.com/sirupsen/logrus"
	fenixExecutionWorkerGrpcApi "github.com/jlambert68/FenixGrpcApi/FenixExecutionServer/fenixExecutionWorkerGrpcApi/go_grpc_api"
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
	rawTestInstructionsToBeSentToExecutionWorkers, rawTestInstructionAttributesToBeSentToExecutionWorkers, err := executionEngine.loadNewTestInstructionToBeSentToExecutionWorkers() //(txn)
	if err != nil {

		return
	}

	// If there are no TestInstructions  the exit
	if rawTestInstructionsToBeSentToExecutionWorkers == nil {

		return
	}

	// Transform Raw TestInstructions and Attributes, from DB, into messages ready to be sent over gRPC to Execution Workers
	testInstructionsToBeSentToExecutionWorkers, err := executionEngine.transformRawTestInstructionsAndAttributeIntogRPCMessages(rawTestInstructionsToBeSentToExecutionWorkers, rawTestInstructionAttributesToBeSentToExecutionWorkers) //(txn)
	if err != nil {

		return
	}

	// If there are no TestInstructions  the exit
	if rawTestInstructionsToBeSentToExecutionWorkers == nil {

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
}

	// Hold one new TestInstructionAttribute to be sent to Execution Worker
	type newTestInstructionAttributeToBeSentToExecutionWorkersStruct struct {
	testInstructionExecutionUuid      string
	testInstructionAttributeType      int
	testInstructionAttributeUuid      string
	testInstructionAttributeName      string
	attributeValueAsString            string
	attributeValueUuid                string
}

// Load all New TestInstructions and their attributes to be sent to the Executions Workers over gRPC
func (executionEngine *TestInstructionExecutionEngineStruct) loadNewTestInstructionToBeSentToExecutionWorkers() (rawTestInstructionsToBeSentToExecutionWorkers []newTestInstructionToBeSentToExecutionWorkersStruct, rawTestInstructionAttributesToBeSentToExecutionWorkers []newTestInstructionAttributeToBeSentToExecutionWorkersStruct, err error) {

	usedDBSchema := "FenixExecution" // TODO should this env variable be used? fenixSyncShared.GetDBSchemaName()

	var testInstructionExecutionUuids []string

	// *** Process TestInstructions ***
	sqlToExecute := ""
	sqlToExecute = sqlToExecute + "SELECT DP.\"DomainUuid\", DP.\"DomainName\", DP.\"ExecutionWorker Address\", " +
		"TIUE.\"TestInstructionExecutionUuid\", TIUE.\"TestInstructionOriginalUuid\", TIUE.\"TestInstructionName\", " +
		"TIUE.\"TestInstructionMajorVersionNumber\", TIUE.\"TestInstructionMinorVersionNumber\" "
	sqlToExecute = sqlToExecute + "FROM \"" + usedDBSchema + "\".\"TestInstructionsUnderExecution\" TIUE, " +
		"\"" + usedDBSchema + "\".\"DomainParameters\" DP "
	sqlToExecute = sqlToExecute + "WHERE TIUE.\"TestInstructionExecutionStatus\" = 0 AND "
	sqlToExecute = sqlToExecute + "DP.\"DomainUuid\" = TIUE.\"DomainUuid\" "
	sqlToExecute = sqlToExecute + "ORDER BY DP.\"DomainUuid\" ASC, TIUE.\"TestInstructionExecutionUuid\" ASC "
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

		return nil,nil, err
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

		)

		if err != nil {

			executionEngine.logger.WithFields(logrus.Fields{
				"Id":           "1173f1f3-f9e5-411d-a129-7ee5ca336762",
				"Error":        err,
				"sqlToExecute": sqlToExecute,
			}).Error("Something went wrong when processing result from database")

			return nil,nil, err
		}

		// Add Queue-message to slice of messages
		rawTestInstructionsToBeSentToExecutionWorkers = append(rawTestInstructionsToBeSentToExecutionWorkers, tempTestInstructionAndAttributeData)

		// Add TestInstructionExecutionUuid to array to be used in next SQL for Attributes
		testInstructionExecutionUuids = append(testInstructionExecutionUuids, tempTestInstructionAndAttributeData.testInstructionExecutionUuid)
	}

	// If there were no TestInstructions then it can't be any Attributes, so exit
	if len(rawTestInstructionsToBeSentToExecutionWorkers) == 0 {
		return nil, nil, nil
	}

	// *** Process TestInstructionAttributes ***
	sqlToExecute = ""
	sqlToExecute = sqlToExecute + "SELECT TIAUE.\"TestInstructionExecutionUuid\", TIAUE.\"TestInstructionAttributeType\", TIAUE.\"TestInstructionAttributeUuid\", " +
		"TIAUE.\"TestInstructionAttributeName\", TIAUE.\"AttributeValueAsString\", TIAUE.\"AttributeValueUuid\" "

	sqlToExecute = sqlToExecute + "FROM \"" + usedDBSchema + "\".\"TestInstructionAttributesUnderExecution\" TIAUE, "
	sqlToExecute = sqlToExecute + "WHERE TIAUE.\"TestInstructionExecutionUuid\" IN " + common_config.GenerateSQLINArray(testInstructionExecutionUuids)

	sqlToExecute = sqlToExecute + "ORDER BY TIAUE.\"TestInstructionExecutionUuid\" ASC, TIAUE.\"TestInstructionAttributeUuid\" ASC "
	sqlToExecute = sqlToExecute + "; "

	// Query DB
	// Execute Query CloudDB
	//TODO change so we use the dbTransaction instead so rows will be locked ----- comandTag, err := dbTransaction.Exec(context.Background(), sqlToExecute)
	rows, err = fenixSyncShared.DbPool.Query(context.Background(), sqlToExecute)

	if err != nil {
		executionEngine.logger.WithFields(logrus.Fields{
			"Id":           "ea92876d-7aa0-4052-afd3-60dcfc79e32c",
			"Error":        err,
			"sqlToExecute": sqlToExecute,
		}).Error("Something went wrong when executing SQL")

		return nil,nil, err
	}

	// Extract data from DB result set
	for rows.Next() {

		var tempTestInstructionAttribute newTestInstructionAttributeToBeSentToExecutionWorkersStruct

		err := rows.Scan(

			&tempTestInstructionAttribute.testInstructionExecutionUuid,
			&tempTestInstructionAttribute.testInstructionAttributeType,
			&tempTestInstructionAttribute.testInstructionAttributeUuid,
			&tempTestInstructionAttribute.testInstructionAttributeName,
			&tempTestInstructionAttribute.attributeValueAsString,
			&tempTestInstructionAttribute.attributeValueUuid,
		)

		if err != nil {

			executionEngine.logger.WithFields(logrus.Fields{
				"Id":           "bb073f45-0207-475d-ae8f-255bb8ee0cf6",
				"Error":        err,
				"sqlToExecute": sqlToExecute,
			}).Error("Something went wrong when processing result from database")

			return nil,nil, err
		}

		// Add Queue-message to slice of messages
		rawTestInstructionAttributesToBeSentToExecutionWorkers = append(rawTestInstructionAttributesToBeSentToExecutionWorkers, tempTestInstructionAttribute)

	}

	return rawTestInstructionsToBeSentToExecutionWorkers, rawTestInstructionAttributesToBeSentToExecutionWorkers, err

}


// Holds address, to the execution worker, and a message that will be sent to worker
type processTestInstructionExecutionRequestMessageContainer struct {
	addressToExecutionWorker string
	*fenixExecutionWorkerGrpcApi.ProcessTestInstructionExecutionRequest
}

// Transform Raw TestInstructions from DB into messages ready to be sent over gRPC to Execution Workers
func (executionEngine *TestInstructionExecutionEngineStruct) transformRawTestInstructionsAndAttributeIntogRPCMessages(rawTestInstructionsToBeSentToExecutionWorkers []newTestInstructionToBeSentToExecutionWorkersStruct, rawTestInstructionAttributesToBeSentToExecutionWorkers []newTestInstructionAttributeToBeSentToExecutionWorkersStruct) (testInstructionsToBeSentToExecutionWorkers []processTestInstructionExecutionRequestMessageContainer, err error) {

	attributesMap := make(map[string]*[]newTestInstructionAttributeToBeSentToExecutionWorkersStruct)

		// Create map for attributes to be able to find attributes in an easier way
		for _,  rawTestInstructionAttribute := range rawTestInstructionAttributesToBeSentToExecutionWorkers {
			attributesSliceReference, existsInMap := attributesMap[rawTestInstructionAttribute.testInstructionExecutionUuid]

			if existsInMap == true {
				// Exists, so just append value to slice
				*attributesSliceReference = append(*attributesSliceReference, rawTestInstructionAttribute)
			} else {
				// Doesn't exists, so create new slice and add reference to Map
				var newAttributesSlice []newTestInstructionAttributeToBeSentToExecutionWorkersStruct
				newAttributesSlice = append(newAttributesSlice, rawTestInstructionAttribute)
				attributesMap[rawTestInstructionAttribute.testInstructionExecutionUuid] = &newAttributesSlice
			}
		}

	// Loop TestInstruction build message ready to be sent over gRPC to workers
	for _, rawTestInstructionData := range rawTestInstructionsToBeSentToExecutionWorkers {



		[]*ProcessTestInstructionExecutionRequest_TestInstructionAttributeMessage

		var newTestInstructionToBeSentToExecutionWorkers *fenixExecutionWorkerGrpcApi.ProcessTestInstructionExecutionRequest
		newTestInstructionToBeSentToExecutionWorkers = & fenixExecutionWorkerGrpcApi.ProcessTestInstructionExecutionRequest{
			TestInstructionExecutionUuid: "",
			TestInstructionUuid:          "",
			TestInstructionName:          "",
			MajorVersionNumber:           0,
			MinorVersionNumber:           0,
			TestInstructionAttributes:    nil,
			TestData: &fenixExecutionWorkerGrpcApi.ProcessTestInstructionExecutionRequest_TestDataMessage{
				TestDataSetUuid:           "",
				ManualOverrideForTestData: nil,
			},
		}
	}


return testInstructionsToBeSentToExecutionWorkers, err
}