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
	"sync/atomic"
)

func PullPubSubExecutionStatusMessagesFromDeadLettering() {
	projectID := common_config.GCPProjectId
	subID := common_config.ExecutionStatusPubSubDeadLetteringSubscription

	var pubSubClient *pubsub.Client
	var err error
	var opts []grpc.DialOption

	ctx := context.Background()

	//PubSub is running on GCP then use credential

	var creds credentials.TransportCredentials
	creds = credentials.NewTLS(&tls.Config{
		InsecureSkipVerify: true,
	})

	opts = []grpc.DialOption{
		grpc.WithTransportCredentials(creds),
	}

	pubSubClient, err = pubsub.NewClient(ctx, projectID, option.WithGRPCDialOption(opts[0]))

	if err != nil {

		common_config.Logger.WithFields(logrus.Fields{
			"ID":  "6aa00314-fdcc-425b-86d9-52dcc7c610e8",
			"err": err,
		}).Error("Got some problem when creating 'pubsub.NewClient'")

		return
	}
	defer pubSubClient.Close()

	clientSubscription := pubSubClient.Subscription(subID)

	common_config.Logger.WithFields(logrus.Fields{
		"ID":  "77eed5b1-a95e-40c0-9067-d6d353d9eba3",
		"err": err,
	}).Info("Started 'cleaner' for PubSub 'ExecutionStatusMessage-DeadLettering'")

	var received int32
	err = clientSubscription.Receive(ctx, func(_ context.Context, pubSubMessage *pubsub.Message) {

		common_config.Logger.WithFields(logrus.Fields{
			"ID": "c074ce42-b34c-49bd-b3ef-102a83b24535",
		}).Debug(fmt.Printf("Removed message: %q", string(pubSubMessage.Data)))

		atomic.AddInt32(&received, 1)

		// Send 'Ack' back to PubSub-system that message has taken care of
		pubSubMessage.Ack()

	})
	if err != nil {
		common_config.Logger.WithFields(logrus.Fields{
			"ID":  "2410eaa0-dce7-458b-ad9b-28d53680f995",
			"err": err,
		}).Fatalln("PubSub receiver for TestInstructionExecutions ended, which is not intended")
	}

}
