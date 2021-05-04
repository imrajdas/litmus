package gql

import (
	"encoding/json"
	"github.com/hasura/go-graphql-client"
	"github.com/litmuschaos/litmus/litmus-portal/cluster-agents/subscriber/pkg/k8s"

	"github.com/gorilla/websocket"
	"github.com/litmuschaos/litmus/litmus-portal/cluster-agents/subscriber/pkg/types"
	"github.com/sirupsen/logrus"

	"log"
	"net/url"
	"strings"
	"time"
)


func ClusterConnect(clusterData map[string]string) {

	var subscription struct {
		ClusterConnect struct {
			ProjectID   graphql.String `graphql:"project_id"`
			Action struct{
				k8sManifest graphql.String `graphql:"k8s_manifest"`
				ExternalData graphql.String `graphql:"external_data"`
				Namespace graphql.String `graphql:"namespace"`
				RequestType graphql.String `graphql:"request_type"`
			}
		}  `graphql:"clusterConnect(cluster_id: $cluster_id, access_key: $access_key)"`
	}

	variables := map[string]interface{}{
		"cluster_id":   clusterData["CLUSTER_ID"],
		"access_key": clusterData["ACCESS_KEY"],
	}

	client := graphql.NewSubscriptionClient("wss://example.com/graphql")
	defer client.Close()

	// Subscribe subscriptions
	// ...
	// finally run the client

	_, err := client.Subscribe(&subscription, variables, func(dataValue *json.RawMessage, errValue error) error {
		if errValue != nil {
			// handle error
			// if returns error, it will failback to `onError` event
			return nil
		}
		log.Println(dataValue)

		return nil
	})

	if err != nil {
		// Handle error.
	}


	client.Run()





	query := `{"query":"subscription {\n    clusterConnect(clusterInfo: {cluster_id: \"` + clusterData["CLUSTER_ID"] + `\", access_key: \"` + clusterData["ACCESS_KEY"] + `\"}) {\n   \t project_id,\n     action{\n      k8s_manifest,\n      external_data,\n      request_type\n     namespace\n     }\n  }\n}\n"}`
	serverURL, err := url.Parse(clusterData["SERVER_ADDR"])
	scheme := "ws"
	if serverURL.Scheme == "https" {
		scheme = "wss"
	}
	u := url.URL{Scheme: scheme, Host: serverURL.Host, Path: serverURL.Path}
	log.Printf("connecting to %s", u.String())

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer c.Close()

	go func() {
		time.Sleep(1 * time.Second)
		payload := types.OperationMessage{
			Type: "connection_init",
		}
		data, err := json.Marshal(payload)
		err = c.WriteMessage(websocket.TextMessage, data)
		if err != nil {
			log.Println("write:", err)
			return
		}
		payload = types.OperationMessage{
			Payload: []byte(query),
			Type:    "start",
		}
		data, err = json.Marshal(payload)
		err = c.WriteMessage(websocket.TextMessage, data)
		if err != nil {
			log.Println("write:", err)
			return
		}
	}()
	for {
		_, message, err := c.ReadMessage()
		if err != nil {
			log.Fatal("SUBSCRIPTION ERROR : ", err)
		}
		var r types.RawData
		err = json.Unmarshal(message, &r)
		if err != nil {
			logrus.WithError(err).Fatal("error un-marshaling request payload")
		}
		if r.Type == "connection_ack" {
			log.Print("Cluster Connect Established, Listening....")
		}
		if r.Type != "data" {
			continue
		}
		if r.Payload.Errors != nil {
			logrus.Fatal("gql error : ", string(message))
		}
		if strings.Index("kubeobject kubeobjects", strings.ToLower(r.Payload.Data.ClusterConnect.Action.RequestType)) >= 0 {
			KubeObjRequest := types.KubeObjRequest{
				RequestID: r.Payload.Data.ClusterConnect.ProjectID,
			}
			err = json.Unmarshal([]byte(r.Payload.Data.ClusterConnect.Action.ExternalData.(string)), &KubeObjRequest)
			err = SendKubeObjects(clusterData, KubeObjRequest)
			if err != nil {
				logrus.WithError(err).Println("error getting kubernetes object data")
				continue
			}
		}
		if strings.ToLower(r.Payload.Data.ClusterConnect.Action.RequestType) == "logs" {
			podRequest := types.PodLogRequest{
				RequestID: r.Payload.Data.ClusterConnect.ProjectID,
			}
			err = json.Unmarshal([]byte(r.Payload.Data.ClusterConnect.Action.ExternalData.(string)), &podRequest)
			if err != nil {
				logrus.WithError(err).Print("error reading cluster-action request [external-data]")
				continue
			}
			// send pod logs
			logrus.Print("LOG REQUEST ", podRequest)
			SendPodLogs(clusterData, podRequest)
		} else if strings.Index("create update delete get", strings.ToLower(r.Payload.Data.ClusterConnect.Action.RequestType)) >= 0 {
			logrus.Print("WORKFLOW REQUEST ", r.Payload.Data.ClusterConnect.Action)
			_, err = k8s.ClusterOperations(r.Payload.Data.ClusterConnect.Action.K8SManifest, r.Payload.Data.ClusterConnect.Action.RequestType, r.Payload.Data.ClusterConnect.Action.Namespace)
			if err != nil {
				logrus.WithError(err).Print("error performing cluster operation")
				continue
			}
		}
	}
}
