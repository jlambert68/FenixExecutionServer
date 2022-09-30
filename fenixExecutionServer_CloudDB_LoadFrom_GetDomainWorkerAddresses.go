package main

import (
	"FenixExecutionServer/common_config"
	"context"
	fenixSyncShared "github.com/jlambert68/FenixSyncShared"
	"github.com/sirupsen/logrus"
)

// Prepare for Saving the ongoing Execution of a new TestCaseExecution in the CloudDB
func (fenixExecutionServerObject *fenixExecutionServerObjectStruct) prepareGetDomainWorkerAddresses() (err error) {

	// Load all Domain Worker addresses from Cloud-DB
	domainWorkersParameters, err := fenixExecutionServerObject.loadDomainWorkerAddresses()
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
func (fenixExecutionServerObject *fenixExecutionServerObjectStruct) loadDomainWorkerAddresses() (domainWorkersParameters []domainWorkerParametersStruct, err error) {

	usedDBSchema := "FenixExecution" // TODO should this env variable be used? fenixSyncShared.GetDBSchemaName()

	sqlToExecute := ""
	sqlToExecute = sqlToExecute + "SELECT DP.* "
	sqlToExecute = sqlToExecute + "FROM \"" + usedDBSchema + "\".\"DomainParameters\" DP "
	sqlToExecute = sqlToExecute + "ORDER BY DP.\"DomainUuid\" ASC; "

	// Query DB
	// Execute Query CloudDB
	//TODO change so we use the dbTransaction instead so rows will be locked ----- comandTag, err := dbTransaction.Exec(context.Background(), sqlToExecute)
	rows, err := fenixSyncShared.DbPool.Query(context.Background(), sqlToExecute)

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

		var newExecutionWorkerVariables *common_config.ExecutionWorkerVariablesStruct
		newExecutionWorkerVariables = &common_config.ExecutionWorkerVariablesStruct{
			HighestExecutionWorkerProtoFileVersion:     -1,
			FenixExecutionWorkerServerAddress:          domainWorkerParameters.executionWorkerAddress,
			RemoteFenixExecutionWorkerServerConnection: nil,
			FenixExecutionServerAddressToDial:          "",
			FenixExecutionWorkerServerGrpcClient:       nil,
			FenixExecutionServerWorkerAddressToUse:     "",
		}

		// Store in Map
		common_config.ExecutionWorkerVariablesMap[domainWorkerParameters.domainUuid] = newExecutionWorkerVariables

	}
}
