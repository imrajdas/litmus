package main

import (
	"encoding/json"
	"flag"
	"github.com/litmuschaos/litmus/litmus-portal/cluster-agents/subscriber/pkg/requests"
	"github.com/litmuschaos/litmus/litmus-portal/cluster-agents/subscriber/pkg/types"
	"github.com/streadway/amqp"
	"os"
	//"os/signal"
	"runtime"

	"github.com/litmuschaos/litmus/litmus-portal/cluster-agents/subscriber/pkg/k8s"
	//"github.com/litmuschaos/litmus/litmus-portal/cluster-agents/subscriber/pkg/types"
	"github.com/sirupsen/logrus"
)

var (
	clusterData = map[string]string{
		//"ACCESS_KEY":           os.Getenv("ACCESS_KEY"),
		"CLUSTER_ID":           os.Getenv("CLUSTER_ID"),
		"MQ_ADDR": os.Getenv("MQ_ADDR"),
		"MQ_USER": os.Getenv("MQ_USER"),
		"MQ_PASSWORD": os.Getenv("MQ_PASSWORD"),
		//"SERVER_ADDR":          os.Getenv("SERVER_ADDR"),
		//"IS_CLUSTER_CONFIRMED": os.Getenv("IS_CLUSTER_CONFIRMED"),
		"AGENT_SCOPE":          os.Getenv("AGENT_SCOPE"),
		//"COMPONENTS":           os.Getenv("COMPONENTS"),
		"AGENT_NAMESPACE":      os.Getenv("AGENT_NAMESPACE"),
		"VERSION":              os.Getenv("VERSION"),
	}

	err error
)

func init() {
	logrus.Info("Go Version: ", runtime.Version())
	logrus.Info("Go OS/Arch: ", runtime.GOOS, "/", runtime.GOARCH)

	for key, env := range clusterData {
		if env == "" {
			logrus.Print(key)
			logrus.Fatal("Some environment variable are not setup")
		}
	}

	// Retrieving START_TIME
	clusterData["START_TIME"] = os.Getenv("START_TIME")

	k8s.KubeConfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	flag.Parse()

	// check agent component status
	//err := k8s.CheckComponentStatus(clusterData["COMPONENTS"])
	//if err != nil {
	//	logrus.Fatal(err)
	//}
	//logrus.Info("all components live...starting up subscriber")

	//isConfirmed, newKey, err := k8s.IsClusterConfirmed()
	//if err != nil {
	//	logrus.WithError(err).Fatal("failed to check cluster-bkp confirmed status")
	//}
	//
	//if isConfirmed == true {
	//	clusterData["ACCESS_KEY"] = newKey
	//} else if isConfirmed == false {
	//	clusterConfirmByte, err := k8s.ClusterConfirm(clusterData)
	//	if err != nil {
	//		logrus.WithError(err).WithField("data", string(clusterConfirmByte)).Fatal("failed to confirm cluster-bkp")
	//	}
	//
	//	var clusterConfirmInterface types.Payload
	//	err = json.Unmarshal(clusterConfirmByte, &clusterConfirmInterface)
	//	if err != nil {
	//		logrus.WithError(err).WithField("data", string(clusterConfirmByte)).Fatal("failed to parse cluster-bkp confirm data")
	//	}
	//
	//	if clusterConfirmInterface.Data.ClusterConfirm.IsClusterConfirmed == true {
	//		clusterData["ACCESS_KEY"] = clusterConfirmInterface.Data.ClusterConfirm.NewAccessKey
	//		clusterData["IS_CLUSTER_CONFIRMED"] = "true"
	//		clusterData["START_TIME"] = strconv.FormatInt(time.Now().Unix(), 10)
	//		_, err = k8s.ClusterRegister(clusterData)
	//		if err != nil {
	//			logrus.Fatal(err)
	//		}
	//		logrus.Info(clusterData["CLUSTER_ID"] + " has been confirmed")
	//	} else {
	//		logrus.Info(clusterData["CLUSTER_ID"] + " hasn't been confirmed")
	//	}
	//}

}

func main() {
	//stopCh := make(chan struct{})
	//sigCh := make(chan os.Signal)
	//stream := make(chan types.WorkflowEvent, 10)

	Consumer()

	////start events event watcher
	//events.WorkflowEventWatcher(stopCh, stream, clusterData)
	//
	////start events event watcher
	//events.ChaosEventWatcher(stopCh, stream, clusterData)
	////streams the event data to graphql server
	//go events.WorkflowUpdates(clusterData, stream)
	//
	//// listen for cluster-bkp actions
	//go requests.ClusterConnect(clusterData)

	//signal.Notify(sigCh, os.Kill, os.Interrupt)
	//<-sigCh
	//close(stopCh)
	//close(stream)
}

func Consumer()   {
	conn, err := amqp.Dial("amqp://"+ clusterData["MQ_USER"] +":"+clusterData["MQ_PASSWORD"]+"@"+ clusterData["MQ_ADDR"]+"/"+clusterData["CLUSTER_ID"])
	if err != nil {
		logrus.Println(err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		logrus.Println(err)
	}

	msgs, err := ch.Consume(
		clusterData["CLUSTER_ID"]+"-queue",
		"",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		logrus.Println(err)
	}

	forever := make(chan bool)
	go func() {
		for d := range msgs {
			logrus.Printf("Received Message %s\n", d.Body)

			var r types.RawData
			err = json.Unmarshal(d.Body, &r)
			if err != nil {
				logrus.WithError(err).Fatal("error un-marshaling request payload")
			}

			//logrus.Print(r.Payload.Data)
			err = requests.RequestProcessor(clusterData, r)
			if err != nil {
				logrus.WithError(err).Fatal("error on processing request")
			}

		}
	}()

	logrus.Println("Succesfully connected to our RabbitMQ instance")
	logrus.Println(" [*] - waiting for messages")
	<-forever

}