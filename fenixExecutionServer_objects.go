package main

import (
	fenixExecutionServerGuiGrpcApi "github.com/jlambert68/FenixGrpcApi/FenixExecutionServer/fenixExecutionServerGuiGrpcApi/go_grpc_api"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
	"google.golang.org/grpc"
	"net"
)

type fenixExecutionServerObjectStruct struct {
	logger         *logrus.Logger
	gcpAccessToken *oauth2.Token
}

// Variable holding everything together
var fenixExecutionServerObject *fenixExecutionServerObjectStruct

// gRPC variables
var (
	registerFenixExecutionServerGuiGrpcServicesServer *grpc.Server // registerFenixExecutionServerGuiGrpcServicesServer *grpc.Server
	lis                                               net.Listener
)

// gRPC Server used for register clients Name, Ip and Por and Clients Test Enviroments and Clients Test Commandst
type fenixExecutionServerGuiGrpcServicesServer struct {
	fenixExecutionServerGuiGrpcApi.UnimplementedFenixExecutionServerGuiGrpcServicesServer
}

//TODO FIXA DENNA PATH, HMMM borde köra i DB framöver
// For now hardcoded MerklePath
//var merkleFilterPath string = //"AccountEnvironment/ClientJuristictionCountryCode/MarketSubType/MarketName/" //SecurityType/"

var highestFenixProtoFileVersion int32 = -1
var highestClientProtoFileVersion int32 = -1
