package broadcastingEngine

import (
	"context"
	"encoding/json"
	fenixSyncShared "github.com/jlambert68/FenixSyncShared"
	"log"
	"time"
)

var BroadcastEngineMessageChannel BroadcastEngineMessageChannelType

type BroadcastEngineMessageChannelType chan BroadcastingMessageForExecutionsStruct

type BroadcastingMessageForExecutionsStruct struct {
	BroadcastTimeStamp        time.Time                        `json:"timeStamp"`
	TestCaseExecutions        []TestCaseExecutionStruct        `json:"testCaseExecutions"`
	TestInstructionExecutions []TestInstructionExecutionStruct `json:"testInstructionExecutions"`
}

type TestCaseExecutionStruct struct {
	TestCaseExecutionUuid   string `json:"testCaseExecutionUuid"`
	TestCaseExecutionStatus string `json:"testCaseExecutionStatus"`
}

type TestInstructionExecutionStruct struct {
	TestInstructionExecutionUuid   string `json:"testInstructionUuid"`
	TestInstructionExecutionStatus string `json:"testInstructionStatus"`
}

var err error

func InitiateAndStartBroadcastNotifyEngine() {

	BroadcastEngineMessageChannel = make(chan BroadcastingMessageForExecutionsStruct, 10)
	var broadcastingMessageForExecutions BroadcastingMessageForExecutionsStruct
	var broadcastingMessageForExecutionsAsByteSlice []byte
	var err error

	for {

		broadcastingMessageForExecutions = <-BroadcastEngineMessageChannel

		broadcastingMessageForExecutionsAsByteSlice, err = json.Marshal(broadcastingMessageForExecutions)

		_, err = fenixSyncShared.DbPool.Exec(context.Background(), "SELECT pg_notify('notes2', $1)", broadcastingMessageForExecutionsAsByteSlice)
		if err != nil {
			log.Println("error sending notification:", err)
		}
	}
}
