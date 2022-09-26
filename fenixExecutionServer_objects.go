package main

import (
	"FenixExecutionServer/testInstructionExecutionEngine"
	fenixExecutionServerGrpcApi "github.com/jlambert68/FenixGrpcApi/FenixExecutionServer/fenixExecutionServerGrpcApi/go_grpc_api"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
	"google.golang.org/grpc"
	"net"
)

type fenixExecutionServerObjectStruct struct {
	logger                    *logrus.Logger
	gcpAccessToken            *oauth2.Token
	executionEngineChannelRef *testInstructionExecutionEngine.ExecutionEngineChannelType
	executionEngine           *testInstructionExecutionEngine.TestInstructionExecutionEngineStruct
}

// Variable holding everything together
var fenixExecutionServerObject *fenixExecutionServerObjectStruct

// gRPC variables
var (
	registerFenixExecutionServerGrpcServicesServer *grpc.Server
	lis                                            net.Listener
)

// gRPC Server used for register clients Name, Ip and Por and Clients Test Enviroments and Clients Test Commandst
type fenixExecutionServerGrpcServicesServer struct {
	fenixExecutionServerGrpcApi.UnimplementedFenixExecutionServerGrpcServicesServer
}

//TODO FIXA DENNA PATH, HMMM borde köra i DB framöver
// For now hardcoded MerklePath
//var merkleFilterPath string = //"AccountEnvironment/ClientJuristictionCountryCode/MarketSubType/MarketName/" //SecurityType/"
