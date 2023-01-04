package main

import (
	"FenixExecutionServer/common_config"
	"FenixExecutionServer/testInstructionExecutionEngine"
	fenixExecutionServerGrpcApi "github.com/jlambert68/FenixGrpcApi/FenixExecutionServer/fenixExecutionServerGrpcApi/go_grpc_api"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
	"google.golang.org/grpc"
	"net"
	"time"
)

type fenixExecutionServerObjectStruct struct {
	logger                         *logrus.Logger
	gcpAccessToken                 *oauth2.Token
	executionEngineChannelRefSlice *[]testInstructionExecutionEngine.ExecutionEngineChannelType
	executionEngine                *testInstructionExecutionEngine.TestInstructionExecutionEngineStruct
	executionWorkerVariablesMap    *map[string]*common_config.ExecutionWorkerVariablesStruct
}

// Variable holding everything together
var fenixExecutionServerObject *fenixExecutionServerObjectStruct

// gRPC variables
var (
	registerFenixExecutionServerGrpcServicesServer *grpc.Server
	lis                                            net.Listener

	// channel used to deside when there are no more incoming gRPC-calls and to end application
	endApplicationWhenNoIncomingGrpcCalls chan endApplicationWhenNoIncomingGrpcCallsStruct
)

type endApplicationWhenNoIncomingGrpcCallsStruct struct {
	gRPCTimeStamp time.Time
	senderName    string
}

// gRPC Server used for register clients Name, Ip and Por and Clients Test Enviroments and Clients Test Commandst
type fenixExecutionServerGrpcServicesServer struct {
	fenixExecutionServerGrpcApi.UnimplementedFenixExecutionServerGrpcServicesServer
	logger *logrus.Logger
}
