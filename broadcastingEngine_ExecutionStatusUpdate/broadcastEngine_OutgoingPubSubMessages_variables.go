package broadcastingEngine_ExecutionStatusUpdate

import (
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
)

type PubSubExecutionStatusMessageToGuiExecutionServerObjectStruct struct {
	Logger         *logrus.Logger
	gcpAccessToken *oauth2.Token
}

var PubSubExecutionStatusMessageToGuiExecutionServerObject PubSubExecutionStatusMessageToGuiExecutionServerObjectStruct
