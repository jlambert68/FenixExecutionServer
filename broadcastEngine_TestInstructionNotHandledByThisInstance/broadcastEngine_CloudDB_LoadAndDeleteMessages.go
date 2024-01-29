package broadcastEngine_TestInstructionNotHandledByThisInstance

import (
	"FenixExecutionServer/common_config"
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v4"
	fenixSyncShared "github.com/jlambert68/FenixSyncShared"
	"github.com/sirupsen/logrus"
	"strconv"
	"time"
)

// Commit or Rollback changes 'TestInstructionIsNotHandledByThisExecutionInstanceUpSaveToCloudDB' and send over Broadcast-channel
func commitOrRoleBackLoadTestInstructionExecutionResultMessage(
	dbTransactionReference *pgx.Tx,
	doCommitNotRoleBackReference *bool) {

	dbTransaction := *dbTransactionReference
	doCommitNotRoleBack := *doCommitNotRoleBackReference

	// Should transaction be committed and be broadcast
	if doCommitNotRoleBack == true {
		dbTransaction.Commit(context.Background())

	} else {

		dbTransaction.Rollback(context.Background())
	}
}

// Prepare to Load TestInstructionExecution from database and then remove data in database
func prepareLoadTestInstructionExecutionMessagesReceivedByWrongInstance(
	testInstructionExecutionUuid string,
	testInstructionVersion int32) (
	messagesReceivedByWrongExecutionInstance []*common_config.TestInstructionExecutionMessageReceivedByWrongExecutionStruct,
	err error) {

	// Calculate Execution Track
	var executionTrack int
	executionTrack = common_config.CalculateExecutionTrackNumber(testInstructionExecutionUuid)

	common_config.Logger.WithFields(logrus.Fields{
		"id":                           "4383f104-71b2-407b-8a28-9ce5dd9973de",
		"executionTrack":               executionTrack,
		"testInstructionExecutionUuid": testInstructionExecutionUuid,
		"testInstructionVersion":       testInstructionVersion,
	}).Debug("Incoming 'prepareLoadTestInstructionExecutionMessagesReceivedByWrongInstance'")

	defer common_config.Logger.WithFields(logrus.Fields{
		"id": "693ea38a-27d0-49d8-9d62-a60fa02f027f",
	}).Debug("Outgoing 'prepareLoadTestInstructionExecutionMessagesReceivedByWrongInstance'")

	// Begin SQL Transaction
	var txn pgx.Tx
	txn, err = fenixSyncShared.DbPool.Begin(context.Background())
	if err != nil {
		common_config.Logger.WithFields(logrus.Fields{
			"id":                           "190c9e7c-1351-4e72-b8ab-eb2ff3b97315",
			"error":                        err,
			"executionTrack":               executionTrack,
			"testInstructionExecutionUuid": testInstructionExecutionUuid,
			"testInstructionVersion":       testInstructionVersion,
		}).Error("Problem to do 'DbPool.Begin'  in 'prepareLoadTestInstructionExecutionMessagesReceivedByWrongInstance'")

		return nil, err
	}

	// After all stuff is done, then Commit or Rollback depending on result
	var doCommitNotRoleBack bool

	// Standard is to do a Rollback
	doCommitNotRoleBack = false

	defer commitOrRoleBackLoadTestInstructionExecutionResultMessage(
		&txn,
		&doCommitNotRoleBack)

	// Lock the row
	err = lockRowForTestInstructionExecutionMessageReceivedByWrongInstanceAndThenDeleteFromDatabaseInCloudDB(
		txn, testInstructionExecutionUuid, testInstructionVersion)
	if err != nil {

		common_config.Logger.WithFields(logrus.Fields{
			"id":    "ee92c4fa-999a-47a8-aa64-00a6e00212c9",
			"error": err,
			"messagesReceivedByWrongExecutionInstance": messagesReceivedByWrongExecutionInstance,
		}).Error("Problem when Loading TestInstructionExecution from database in 'prepareLoadTestInstructionExecutionMessagesReceivedByWrongInstance'.")

		return nil, err
	}

	// Extract TestInstructionExecution-messages  and then remove row from database
	messagesReceivedByWrongExecutionInstance, err = loadTestInstructionExecutionMessageReceivedByWrongInstanceAndThenDeleteFromDatabaseInCloudDB(
		txn, testInstructionExecutionUuid, testInstructionVersion)
	if err != nil {

		common_config.Logger.WithFields(logrus.Fields{
			"id":    "ee92c4fa-999a-47a8-aa64-00a6e00212c9",
			"error": err,
			"messagesReceivedByWrongExecutionInstance": messagesReceivedByWrongExecutionInstance,
		}).Error("Problem when Loading TestInstructionExecution from database in 'prepareLoadTestInstructionExecutionMessagesReceivedByWrongInstance'.")

		return nil, err
	}

	// Do the commit and send over Broadcast-system
	doCommitNotRoleBack = true

	return messagesReceivedByWrongExecutionInstance, err
}

// Lock Row before Load TestInstructionExecution from database and then remove data in database
func lockRowForTestInstructionExecutionMessageReceivedByWrongInstanceAndThenDeleteFromDatabaseInCloudDB(
	dbTransaction pgx.Tx,
	testInstructionExecutionUuid string,
	testInstructionExecutionVersion int32) (
	err error) {

	common_config.Logger.WithFields(logrus.Fields{
		"Id":                              "4f91bed6-2c12-4d3b-9961-66654e0340c2",
		"testInstructionExecutionUuid":    testInstructionExecutionUuid,
		"testInstructionExecutionVersion": testInstructionExecutionVersion,
	}).Debug("Entering: lockRowForTestInstructionExecutionMessageReceivedByWrongInstanceAndThenDeleteFromDatabaseInCloudDB()")

	defer func() {
		common_config.Logger.WithFields(logrus.Fields{
			"Id": "c15e0837-f939-4b53-9570-84560abf5092",
		}).Debug("Exiting: lockRowForTestInstructionExecutionMessageReceivedByWrongInstanceAndThenDeleteFromDatabaseInCloudDB()")
	}()

	var testInstructionExecutionVersionAsString string
	testInstructionExecutionVersionAsString = strconv.Itoa(int(testInstructionExecutionVersion))

	sqlToExecute := ""

	sqlToExecute = sqlToExecute + "SELECT * "
	sqlToExecute = sqlToExecute + "" +
		"FROM \"FenixExecution\".\"TestInstructionExecutionMessagesReceivedByWrongExecutionInstanc\" TIERWEI "
	sqlToExecute = sqlToExecute + "" +
		"WHERE TIERWEI.\"ApplicationExecutionRuntimeUuid\" != '" + common_config.ApplicationRuntimeUuid + "' AND "
	sqlToExecute = sqlToExecute + "TIERWEI.\"TestInstructionExecutionUuid\" = '" + testInstructionExecutionUuid + "' AND "
	sqlToExecute = sqlToExecute + "TIERWEI.\"TestInstructionExecutionVersion\" = " + testInstructionExecutionVersionAsString + " "
	sqlToExecute = sqlToExecute + "FOR UPDATE; "

	// Log SQL to be executed if Environment variable is true
	if common_config.LogAllSQLs == true {
		common_config.Logger.WithFields(logrus.Fields{
			"Id":           "0fe8b61a-0821-40ea-8e2c-1ca64564bcec",
			"sqlToExecute": sqlToExecute,
		}).Debug("SQL to be executed within 'lockRowForTestInstructionExecutionMessageReceivedByWrongInstanceAndThenDeleteFromDatabaseInCloudDB'")
	}

	// Query DB
	var ctx context.Context
	ctx, timeOutCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer timeOutCancel()

	rows, err := dbTransaction.Query(ctx, sqlToExecute)
	defer rows.Close()

	if err != nil {
		common_config.Logger.WithFields(logrus.Fields{
			"Id":           "023a0ebe-e0e1-427d-8069-29fea93431e0",
			"Error":        err,
			"sqlToExecute": sqlToExecute,
		}).Error("Something went wrong when executing SQL")

		return err
	}

	return err
}

// Load TestInstructionExecution from database and then remove data in database
func loadTestInstructionExecutionMessageReceivedByWrongInstanceAndThenDeleteFromDatabaseInCloudDB(
	dbTransaction pgx.Tx,
	testInstructionExecutionUuid string,
	testInstructionVersion int32) (
	messagesReceivedByWrongExecutionInstance []*common_config.TestInstructionExecutionMessageReceivedByWrongExecutionStruct,
	err error) {

	common_config.Logger.WithFields(logrus.Fields{
		"Id":                           "cac5fc37-f114-4880-81d4-7d4fdcac35be",
		"testInstructionExecutionUuid": testInstructionExecutionUuid,
		"testInstructionVersion":       testInstructionVersion,
	}).Debug("Entering: loadTestInstructionExecutionResultMessageCloudDBInCloudDB()")

	defer func() {
		common_config.Logger.WithFields(logrus.Fields{
			"Id": "fde0a0d7-8f34-4017-95cc-52c2307b2802",
		}).Debug("Exiting: loadTestInstructionExecutionResultMessageCloudDBInCloudDB()")
	}()

	// Load the TestInstructionExecution
	messagesReceivedByWrongExecutionInstance, err = loadTestInstructionExecutionMessagesReceivedByWrongExecutionInstance(
		dbTransaction,
		testInstructionExecutionUuid,
		testInstructionVersion)

	if err != nil || messagesReceivedByWrongExecutionInstance == nil {
		// There was an Error or the TestInstructionExecution didn't exist in the database for this ExecutionInstance
		return nil, err
	}

	// Remove the row from the database
	err = deleteTestInstructionMessagesReceivedByWrongInstanceFromDatabaseInCloudDB(
		dbTransaction,
		testInstructionExecutionUuid,
		testInstructionVersion,
		int64(len(messagesReceivedByWrongExecutionInstance)))

	if err != nil {
		return nil, err
	}

	// No errors occurred
	return messagesReceivedByWrongExecutionInstance, nil

}

// Load TestInstructionExecution-message from database that didn't belong to ExecutionInstance that received it
func loadTestInstructionExecutionMessagesReceivedByWrongExecutionInstance(
	dbTransaction pgx.Tx,
	testInstructionExecutionUuid string,
	testInstructionExecutionVersion int32) (
	messagesReceivedByWrongExecutionInstance []*common_config.TestInstructionExecutionMessageReceivedByWrongExecutionStruct,
	err error) {

	common_config.Logger.WithFields(logrus.Fields{
		"Id": "7ed9cb4c-a88a-45c1-a77e-2809b4520040",
	}).Debug("Entering: loadTestInstructionExecutionMessagesReceivedByWrongExecutionInstance()")

	defer func() {
		common_config.Logger.WithFields(logrus.Fields{
			"Id": "74fd1c4f-435b-4e1a-b2f0-33f743fd8ec3",
		}).Debug("Exiting: loadTestInstructionExecutionMessagesReceivedByWrongExecutionInstance()")
	}()

	var testInstructionExecutionVersionAsString string
	testInstructionExecutionVersionAsString = strconv.Itoa(int(testInstructionExecutionVersion))

	usedDBSchema := "FenixExecution" // TODO should this env variable be used? fenixSyncShared.GetDBSchemaName()

	sqlToExecute := ""

	sqlToExecute = sqlToExecute + "" +
		"SELECT TIERWEI.\"ApplicationExecutionRuntimeUuid\", TIERWEI.\"TestInstructionExecutionUuid\", " +
		"TIERWEI.\"TestInstructionExecutionVersion\", TIERWEI.\"TimeStamp\", " +
		"TIERWEI.\"MessageType\", TIERWEI.\"MessageAsJsonb\") "
	sqlToExecute = sqlToExecute + "" +
		"FROM \"" + usedDBSchema + "\".\"TestInstructionExecutionMessagesReceivedByWrongExecutionInstanc\" TIERWEI "
	sqlToExecute = sqlToExecute + "" +
		"WHERE TIERWEI.\"ApplicationExecutionRuntimeUuid\" != '" + common_config.ApplicationRuntimeUuid + "' AND "
	sqlToExecute = sqlToExecute + "TIERWEI.\"TestInstructionExecutionUuid\" = '" + testInstructionExecutionUuid + "' AND "
	sqlToExecute = sqlToExecute + "TIERWEI.\"TestInstructionExecutionVersion\" = " + testInstructionExecutionVersionAsString + " "
	sqlToExecute = sqlToExecute + "; "

	// Log SQL to be executed if Environment variable is true
	if common_config.LogAllSQLs == true {
		common_config.Logger.WithFields(logrus.Fields{
			"Id":           "01bd8056-6967-4084-980e-1b37b325e54d",
			"sqlToExecute": sqlToExecute,
		}).Debug("SQL to be executed within 'loadTestInstructionExecutionMessagesReceivedByWrongExecutionInstance'")
	}

	// Query DB
	var ctx context.Context
	ctx, timeOutCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer timeOutCancel()

	rows, err := dbTransaction.Query(ctx, sqlToExecute)
	defer rows.Close()

	if err != nil {
		common_config.Logger.WithFields(logrus.Fields{
			"Id":           "d7202c65-e22a-419e-93c0-c0bbcbeb2c52",
			"Error":        err,
			"sqlToExecute": sqlToExecute,
		}).Error("Something went wrong when executing SQL")

		return nil, err
	}

	// Extract data from DB result set
	for rows.Next() {

		var tempMessagesReceivedByWrongExecutionInstance *common_config.TestInstructionExecutionMessageReceivedByWrongExecutionStruct

		err := rows.Scan(
			&tempMessagesReceivedByWrongExecutionInstance.ApplicatonExecutionRuntimeUuid,
			&tempMessagesReceivedByWrongExecutionInstance.TestInstructionExecutionUuid,
			&tempMessagesReceivedByWrongExecutionInstance.TestInstructionExecutionVersion,
			&tempMessagesReceivedByWrongExecutionInstance.TimeStamp,
			&tempMessagesReceivedByWrongExecutionInstance.MessageType,
			&tempMessagesReceivedByWrongExecutionInstance.MessageAsJsonString,
		)

		if err != nil {
			common_config.Logger.WithFields(logrus.Fields{
				"Id":           "a89b3c1e-b648-4e7c-9316-98f260daedf3",
				"Error":        err,
				"sqlToExecute": sqlToExecute,
			}).Error("Something went wrong when processing result from database")

			return nil, err
		}

		// Add message to slice of messages
		messagesReceivedByWrongExecutionInstance = append(messagesReceivedByWrongExecutionInstance,
			tempMessagesReceivedByWrongExecutionInstance)

	}

	// Not hit in database
	if len(messagesReceivedByWrongExecutionInstance) == 0 {
		return nil, err
	}

	// Return result from database
	return messagesReceivedByWrongExecutionInstance, err

}

// Delete TestInstructionExecution from database if it belongs to this ExecutionInstance,
// but was first revived by another ExecutionInstance
func deleteTestInstructionMessagesReceivedByWrongInstanceFromDatabaseInCloudDB(
	dbTransaction pgx.Tx,
	testInstructionExecutionUuid string,
	testInstructionExecutionVersion int32,
	expectedNumberOfRowsToDelete int64) (
	err error) {

	common_config.Logger.WithFields(logrus.Fields{
		"Id": "772414f3-f07d-4725-a0ae-5fce5209faa0",
	}).Debug("Entering: deleteTestInstructionMessagesReceivedByWrongInstanceFromDatabaseInCloudDB()")

	defer func() {
		common_config.Logger.WithFields(logrus.Fields{
			"Id": "8ac7a62c-d260-44fa-be7b-d6db824b05c0",
		}).Debug("Exiting: deleteTestInstructionMessagesReceivedByWrongInstanceFromDatabaseInCloudDB()")
	}()

	var testInstructionExecutionVersionAsString string
	testInstructionExecutionVersionAsString = strconv.Itoa(int(testInstructionExecutionVersion))

	usedDBSchema := "FenixExecution" // TODO should this env variable be used? fenixSyncShared.GetDBSchemaName()

	sqlToExecute := ""
	sqlToExecute = sqlToExecute + "" +
		"DELETE FROM \"" + usedDBSchema + "\".\"TestInstructionExecutionMessagesReceivedByWrongExecutionInstanc\" TIERWEI "
	sqlToExecute = sqlToExecute + "" +
		"WHERE TIERWEI.\"ApplicationExecutionRuntimeUuid\" != '" + common_config.ApplicationRuntimeUuid + "' AND "
	sqlToExecute = sqlToExecute + "TIERWEI.\"TestInstructionExecutionUuid\" = '" + testInstructionExecutionUuid + "' AND "
	sqlToExecute = sqlToExecute + "TIERWEI.\"TestInstructionExecutionVersion\" = " + testInstructionExecutionVersionAsString + " "
	sqlToExecute = sqlToExecute + "; "

	// Log SQL to be executed if Environment variable is true
	if common_config.LogAllSQLs == true {
		common_config.Logger.WithFields(logrus.Fields{
			"Id":           "dc1801a8-df60-463d-b2be-40ed1c05f018",
			"sqlToExecute": sqlToExecute,
		}).Debug("SQL to be executed within 'deleteTestInstructionMessagesReceivedByWrongInstanceFromDatabaseInCloudDB'")
	}

	// Execute Query CloudDB
	comandTag, err := dbTransaction.Exec(context.Background(), sqlToExecute)

	if err != nil {
		common_config.Logger.WithFields(logrus.Fields{
			"Id":           "0632ac36-7599-415f-8eef-007b44de3b90",
			"Error":        err,
			"sqlToExecute": sqlToExecute,
		}).Error("Something went wrong when executing SQL")

		return err
	}

	// Log response from CloudDB
	common_config.Logger.WithFields(logrus.Fields{
		"Id":                       "899baa48-9253-4e9e-997a-45ae5200e3b8",
		"comandTag.Insert()":       comandTag.Insert(),
		"comandTag.Delete()":       comandTag.Delete(),
		"comandTag.Select()":       comandTag.Select(),
		"comandTag.Update()":       comandTag.Update(),
		"comandTag.RowsAffected()": comandTag.RowsAffected(),
		"comandTag.String()":       comandTag.String(),
	}).Debug("Return data for SQL executed in database")

	// Verify that the expected number of rows was deleted
	if expectedNumberOfRowsToDelete != comandTag.RowsAffected() {

		common_config.Logger.WithFields(logrus.Fields{
			"Id":                           "5aa78598-9d3a-4141-8a9b-55d575324f83",
			"sqlToExecute":                 sqlToExecute,
			"expectedNumberOfRowsToDelete": expectedNumberOfRowsToDelete,
			"comandTag.RowsAffected()":     comandTag.RowsAffected(),
		}).Error("Deleted number of rows is not the same as expected number of rows")

		return errors.New("deleted number of rows is not the same as expected number of rows")

	}

	// No errors occurred
	return nil

}

// Prepare to check if TestInstructionExecution exists in from database for TestInstructionExecution received by
// ExecutionServer not responsible for its execution
func prepareCountTestInstructionExecutionMessagesReceivedByWrongExecutionInstance(
	testInstructionExecutionUuid string,
	testInstructionVersion int32) (
	foundInDatabase bool,
	err error) {

	// Calculate Execution Track
	var executionTrack int
	executionTrack = common_config.CalculateExecutionTrackNumber(testInstructionExecutionUuid)

	common_config.Logger.WithFields(logrus.Fields{
		"id":                           "74acd4ea-420c-4f6e-9b95-444bd1edc1fe",
		"executionTrack":               executionTrack,
		"testInstructionExecutionUuid": testInstructionExecutionUuid,
		"testInstructionVersion":       testInstructionVersion,
	}).Debug("Incoming 'prepareCountTestInstructionExecutionMessagesReceivedByWrongExecutionInstance'")

	defer common_config.Logger.WithFields(logrus.Fields{
		"id": "7278641b-c929-4b1c-9c4e-f99f5e17f3af",
	}).Debug("Outgoing 'prepareCountTestInstructionExecutionMessagesReceivedByWrongExecutionInstance'")

	// Begin SQL Transaction
	var txn pgx.Tx
	txn, err = fenixSyncShared.DbPool.Begin(context.Background())
	if err != nil {
		common_config.Logger.WithFields(logrus.Fields{
			"id":                           "f93bb897-ac1e-481b-8f43-13bb6d949b0d",
			"error":                        err,
			"executionTrack":               executionTrack,
			"testInstructionExecutionUuid": testInstructionExecutionUuid,
			"testInstructionVersion":       testInstructionVersion,
		}).Error("Problem to do 'DbPool.Begin'  in 'prepareCountTestInstructionExecutionMessagesReceivedByWrongExecutionInstance'")

		return false, err
	}

	// After all stuff is done, then Commit or Rollback depending on result
	var doCommitNotRoleBack bool

	// Standard is to do a Rollback
	doCommitNotRoleBack = false

	defer commitOrRoleBackLoadTestInstructionExecutionResultMessage(
		&txn,
		&doCommitNotRoleBack)

	var numberOfRows int
	numberOfRows, err = countTestInstructionExecutionMessagesReceivedByWrongExecutionInstance(
		txn, testInstructionExecutionUuid, testInstructionVersion)
	if err != nil {

		common_config.Logger.WithFields(logrus.Fields{
			"id":                           "90de2bba-b261-4322-bae6-496076ea9a5b",
			"testInstructionExecutionUuid": testInstructionExecutionUuid,
			"testInstructionVersion":       testInstructionVersion,
			"error":                        err,
		}).Error("Problem when checking if TestInstructionExecution exists in database in 'prepareCountTestInstructionExecutionMessagesReceivedByWrongExecutionInstance'.")

		return false, err
	}

	var tempNnumberOfRows int
	if numberOfRows > 2 {
		tempNnumberOfRows = 2
	} else {
		tempNnumberOfRows = numberOfRows
	}

	// Decide how to respond
	switch tempNnumberOfRows {

	// Not found in database
	case 0:
		foundInDatabase = false

	// Found in database
	case 1:
		foundInDatabase = true

	// More than one instance found, not expected
	case 2:

		common_config.Logger.WithFields(logrus.Fields{
			"id":                           "57444d10-19dd-42b0-b8b3-f1c9ddd6235d",
			"ApplicationRuntimeUuid":       common_config.ApplicationRuntimeUuid,
			"testInstructionExecutionUuid": testInstructionExecutionUuid,
			"testInstructionVersion":       testInstructionVersion,
			"error":                        err,
		}).Error("More than one instance found in database in 'prepareCountTestInstructionExecutionMessagesReceivedByWrongExecutionInstance'.")

		foundInDatabase = false
		err = errors.New(fmt.Sprintf("More than one instance found in database. ApplicationRuntimeUuid=%s; "+
			"TestInstructionExecutionUuid=%s; TestInstructionVersion=%d",
			common_config.ApplicationRuntimeUuid,
			testInstructionExecutionUuid,
			testInstructionVersion))

		return false, err

	// Unhandled 'number of rows'
	default:
		common_config.Logger.WithFields(logrus.Fields{
			"id":                           "57444d10-19dd-42b0-b8b3-f1c9ddd6235d",
			"ApplicationRuntimeUuid":       common_config.ApplicationRuntimeUuid,
			"testInstructionExecutionUuid": testInstructionExecutionUuid,
			"testInstructionVersion":       testInstructionVersion,
			"numberOfRows":                 numberOfRows,
			"error":                        err,
		}).Fatalln("Unhandled number of instances found in database in 'prepareCountTestInstructionExecutionMessagesReceivedByWrongExecutionInstance''.")

	}

	// Do the commit and send over Broadcast-system
	doCommitNotRoleBack = true

	return foundInDatabase, err
}

// Check if, TestInstructionExecution-message from database that didn't belong to ExecutionInstance that received it, is still not claimed from database
func countTestInstructionExecutionMessagesReceivedByWrongExecutionInstance(
	dbTransaction pgx.Tx,
	testInstructionExecutionUuid string,
	testInstructionExecutionVersion int32) (
	numberOfRows int,
	err error) {

	common_config.Logger.WithFields(logrus.Fields{
		"Id": "a4be91c9-ab95-4f20-9e10-45c2980e5ca2",
	}).Debug("Entering: countTestInstructionExecutionMessagesReceivedByWrongExecutionInstance()")

	defer func() {
		common_config.Logger.WithFields(logrus.Fields{
			"Id": "ca1ce6d6-23e0-4f38-9f50-66fe48772972",
		}).Debug("Exiting: countTestInstructionExecutionMessagesReceivedByWrongExecutionInstance()")
	}()

	var testInstructionExecutionVersionAsString string
	testInstructionExecutionVersionAsString = strconv.Itoa(int(testInstructionExecutionVersion))

	usedDBSchema := "FenixExecution" // TODO should this env variable be used? fenixSyncShared.GetDBSchemaName()

	sqlToExecute := ""

	sqlToExecute = sqlToExecute + "" +
		"SELECT count(TIERWEI.\"ApplicationExecutionRuntimeUuid\") "
	sqlToExecute = sqlToExecute + "" +
		"FROM \"" + usedDBSchema + "\".\"TestInstructionExecutionMessagesReceivedByWrongExecutionInstanc\" TIERWEI "
	sqlToExecute = sqlToExecute + "" +
		"WHERE TIERWEI.\"ApplicationExecutionRuntimeUuid\" != '" + common_config.ApplicationRuntimeUuid + "' AND "
	sqlToExecute = sqlToExecute + "TIERWEI.\"TestInstructionExecutionUuid\" = '" + testInstructionExecutionUuid + "' AND "
	sqlToExecute = sqlToExecute + "TIERWEI.\"TestInstructionExecutionVersion\" = " + testInstructionExecutionVersionAsString + " "
	sqlToExecute = sqlToExecute + "; "

	// Log SQL to be executed if Environment variable is true
	if common_config.LogAllSQLs == true {
		common_config.Logger.WithFields(logrus.Fields{
			"Id":           "dc23529f-4af1-47c3-b037-de29050cec38",
			"sqlToExecute": sqlToExecute,
		}).Debug("SQL to be executed within 'loadTestInstructionExecutionMessagesReceivedByWrongExecutionInstance'")
	}

	// Query DB
	var ctx context.Context
	ctx, timeOutCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer timeOutCancel()

	rows, err := dbTransaction.Query(ctx, sqlToExecute)
	defer rows.Close()

	if err != nil {
		common_config.Logger.WithFields(logrus.Fields{
			"Id":           "cd39e9b3-0d2e-4853-86e3-f391d426e183",
			"Error":        err,
			"sqlToExecute": sqlToExecute,
		}).Error("Something went wrong when executing SQL")

		return 0, err
	}

	// Extract data from DB result set
	var rowCounter int
	for rows.Next() {

		rowCounter = rowCounter + 1

		err := rows.Scan(
			&numberOfRows,
		)

		if err != nil {
			common_config.Logger.WithFields(logrus.Fields{
				"Id":           "292328c5-bdd1-463a-ae8d-1ba328465af6",
				"Error":        err,
				"sqlToExecute": sqlToExecute,
			}).Error("Something went wrong when processing result from database")

			return 0, err
		}

	}

	// More than 1 row
	if rowCounter > 1 {
		common_config.Logger.WithFields(logrus.Fields{
			"Id":           "9556b0aa-c179-448c-9196-ed7970f2af58",
			"sqlToExecute": sqlToExecute,
		}).Fatalln("Got more than 1 row when processing result from database")

	}

	// Return result from database
	return numberOfRows, err

}
