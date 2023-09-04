package broadcastingEngine_ExecutionStatusUpdate

import (
	"FenixExecutionServer/common_config"
	"cloud.google.com/go/pubsub"
	"context"
	"crypto/tls"
	"fmt"
	"github.com/sirupsen/logrus"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"strings"
)

func pubSubPublish(msg string) (err error) {
	projectID := common_config.GCPProjectId
	topicID := common_config.ExecutionStatusPubSubTopic

	// Remove any unwanted characters
	// Remove '\n'
	var cleanedMessage string
	cleanedMessage = strings.Replace(msg, "\n", "", -1)

	// Replace '\"' with '"'
	cleanedMessage = strings.ReplaceAll(cleanedMessage, "\\\"", "\"")

	var pubSubClient *pubsub.Client
	var opts []grpc.DialOption

	ctx := context.Background()

	// PubSub is handled within GCP so add TLS

	var creds credentials.TransportCredentials
	creds = credentials.NewTLS(&tls.Config{
		InsecureSkipVerify: true,
	})

	opts = []grpc.DialOption{
		grpc.WithTransportCredentials(creds),
	}

	// Create PubSub-client
	pubSubClient, err = pubsub.NewClient(ctx, projectID, option.WithGRPCDialOption(opts[0]))

	if err != nil {

		common_config.Logger.WithFields(logrus.Fields{
			"ID":  "e389ad9a-5e6b-434a-9a2d-6962db6e1930",
			"err": err,
		}).Error("Got some problem when creating 'pubsub.NewClient'")

		return
	}

	defer pubSubClient.Close()

	var pubSubTopic *pubsub.Topic
	var pubSubResult *pubsub.PublishResult
	pubSubTopic = pubSubClient.Topic(topicID)
	pubSubResult = pubSubTopic.Publish(ctx, &pubsub.Message{
		Data: []byte(cleanedMessage),
	})

	// Block until the pubSubResult is returned and a server-generated
	// ID is returned for the published message.
	var pubSubResultId string
	pubSubResultId, err = pubSubResult.Get(ctx)
	if err != nil {

		common_config.Logger.WithFields(logrus.Fields{
			"ID":  "6c0a900e-1af3-4b53-bb69-f474d256a539",
			"msg": msg,
		}).Error(fmt.Errorf("pubsub: pubSubResult.Get: %w", err))

		return err

	}

	common_config.Logger.WithFields(logrus.Fields{
		"ID": "ce6ae357-2c62-4c53-ad7d-2b818dac58d0",
		//"token": token,
	}).Debug(fmt.Sprintf("Published a message; msg ID: %v", pubSubResultId))

	return err
}
