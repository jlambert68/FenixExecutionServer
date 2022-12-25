package main

import (
	"FenixExecutionServer/common_config"
	"context"
	"github.com/jackc/pgx/v4"
	fenixSyncShared "github.com/jlambert68/FenixSyncShared"
	"github.com/sirupsen/logrus"
	"strconv"
)

// Prepare for Saving the ongoing Execution of a new TestCaseExecutionUuid in the CloudDB
func (fenixExecutionServerObject *fenixExecutionServerObjectStruct) prepareGetDomainWorkerAddresses() (err error) {

	// Begin SQL Transaction
	txn, err := fenixSyncShared.DbPool.Begin(context.Background())
	if err != nil {
		common_config.Logger.WithFields(logrus.Fields{
			"id":    "6aa288c4-1774-4336-8d9c-b2504f4e5225",
			"error": err,
		}).Error("Problem to do 'DbPool.Begin' in 'prepareGetDomainWorkerAddresses'")

		return err
	}

	// Close db-transaction when leaving this function
	defer txn.Commit(context.Background())

	// Load all Domain Worker addresses from Cloud-DB
	domainWorkersParameters, err := fenixExecutionServerObject.loadDomainWorkerAddresses(txn)
	if err != nil {

		return err
	}

	// Store Domain Workers in DomainWorker-map
	fenixExecutionServerObject.storeDomainWorkers(domainWorkersParameters)

	return err
}

// One domain with its Worker Address
type domainWorkerParametersStruct struct {
	domainUuid             string
	domainName             string
	executionWorkerAddress string
}

// Load All Domains and their address information
func (fenixExecutionServerObject *fenixExecutionServerObjectStruct) loadDomainWorkerAddresses(dbTransaction pgx.Tx) (domainWorkersParameters []domainWorkerParametersStruct, err error) {

	usedDBSchema := "FenixExecution" // TODO should this env variable be used? fenixSyncShared.GetDBSchemaName()

	sqlToExecute := ""
	sqlToExecute = sqlToExecute + "SELECT DP.* "
	sqlToExecute = sqlToExecute + "FROM \"" + usedDBSchema + "\".\"DomainParameters\" DP "
	sqlToExecute = sqlToExecute + "ORDER BY DP.\"DomainUuid\" ASC; "

	// Query DB
	// Execute Query CloudDB
	rows, err := dbTransaction.Query(context.Background(), sqlToExecute)

	if err != nil {
		fenixExecutionServerObject.logger.WithFields(logrus.Fields{
			"Id":           "d7d2ab8a-adfc-49b0-91ba-3555b88b9bdd",
			"Error":        err,
			"sqlToExecute": sqlToExecute,
		}).Error("Something went wrong when executing SQL")

		return []domainWorkerParametersStruct{}, err
	}

	var domainWorkerParameters domainWorkerParametersStruct

	// Extract data from DB result set
	for rows.Next() {

		err := rows.Scan(
			&domainWorkerParameters.domainUuid,
			&domainWorkerParameters.domainName,
			&domainWorkerParameters.executionWorkerAddress,
		)

		if err != nil {

			fenixExecutionServerObject.logger.WithFields(logrus.Fields{
				"Id":           "e798f65a-0bf1-4dac-8ce4-631a7f25f90d",
				"Error":        err,
				"sqlToExecute": sqlToExecute,
			}).Error("Something went wrong when processing result from database")

			return []domainWorkerParametersStruct{}, err
		}

		// Add Domain to slice of messages
		domainWorkersParameters = append(domainWorkersParameters, domainWorkerParameters)

	}

	return domainWorkersParameters, err

}

// Initiate DomainWorker-map and store Worker information
func (fenixExecutionServerObject *fenixExecutionServerObjectStruct) storeDomainWorkers(domainWorkersParameters []domainWorkerParametersStruct) {

	// Initiate map
	common_config.ExecutionWorkerVariablesMap = make(map[string]*common_config.ExecutionWorkerVariablesStruct) // map[DomainUuid]*common_config.ExecutionWorkerVariablesStruct

	// Store Reference to Map in 'fenixExecutionServerObject'
	fenixExecutionServerObject.executionWorkerVariablesMap = &common_config.ExecutionWorkerVariablesMap

	// Store Domain Worker info in Map
	for _, domainWorkerParameters := range domainWorkersParameters {

		// Create address for Worker depending on Worker is run in Cloud or locally
		var addressToDial string
		var addressToUse string
		if common_config.ExecutionLocationForWorker == common_config.GCP {
			//GCP
			addressToDial = domainWorkerParameters.executionWorkerAddress + ":" + strconv.Itoa(common_config.FenixExecutionWorkerServerPort)
			addressToUse = domainWorkerParameters.executionWorkerAddress
		} else {
			//Local
			addressToDial = common_config.FenixExecutionWorkerAddress + ":" + strconv.Itoa(common_config.FenixExecutionWorkerServerPort)
			addressToUse = common_config.FenixExecutionWorkerAddress
		}

		var newExecutionWorkerVariables *common_config.ExecutionWorkerVariablesStruct
		newExecutionWorkerVariables = &common_config.ExecutionWorkerVariablesStruct{
			HighestExecutionWorkerProtoFileVersion:     -1,
			FenixExecutionWorkerServerAddress:          domainWorkerParameters.executionWorkerAddress,
			RemoteFenixExecutionWorkerServerConnection: nil,
			FenixExecutionServerWorkerAddressToDial:    addressToDial,
			FenixExecutionWorkerServerGrpcClient:       nil,
			FenixExecutionServerWorkerAddressToUse:     addressToUse,
		}

		// Store in Map
		common_config.ExecutionWorkerVariablesMap[domainWorkerParameters.domainUuid] = newExecutionWorkerVariables

	}
}
