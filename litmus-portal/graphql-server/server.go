package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/litmuschaos/litmus/litmus-portal/graphql-server/pkg/projects"

	"github.com/litmuschaos/litmus/litmus-portal/graphql-server/pkg/cluster"
	"github.com/litmuschaos/litmus/litmus-portal/graphql-server/utils"

	"github.com/kelseyhightower/envconfig"
	"github.com/litmuschaos/litmus/litmus-portal/graphql-server/pkg/database/mongodb/config"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"

	"github.com/litmuschaos/litmus/litmus-portal/graphql-server/graph"
	"github.com/litmuschaos/litmus/litmus-portal/graphql-server/graph/generated"
	"github.com/litmuschaos/litmus/litmus-portal/graphql-server/pkg/authorization"
	"github.com/litmuschaos/litmus/litmus-portal/graphql-server/pkg/database/mongodb"
	gitOpsHandler "github.com/litmuschaos/litmus/litmus-portal/graphql-server/pkg/gitops/handler"
	"github.com/litmuschaos/litmus/litmus-portal/graphql-server/pkg/handlers"
	"github.com/litmuschaos/litmus/litmus-portal/graphql-server/pkg/myhub"
	pb "github.com/litmuschaos/litmus/litmus-portal/graphql-server/protos"
	"github.com/rs/cors"
	"google.golang.org/grpc"
)

type Config struct {
	Port                        string
	Version                     string `required:"true"`
	AgentDeployments            string `required:"true" split_words:"true"`
	DbServer                    string `required:"true" split_words:"true"`
	JwtSecret                   string `required:"true" split_words:"true"`
	SelfCluster                 string `required:"true" split_words:"true"`
	AgentScope                  string `required:"true" split_words:"true"`
	AgentNamespace              string `required:"true" split_words:"true"`
	LitmusPortalNamespace       string `required:"true" split_words:"true"`
	DbUser                      string `required:"true" split_words:"true"`
	DbPassword                  string `required:"true" split_words:"true"`
	PortalScope                 string `required:"true" split_words:"true"`
	SubscriberImage             string `required:"true" split_words:"true"`
	EventTrackerImage           string `required:"true" split_words:"true"`
	ArgoWorkflowControllerImage string `required:"true" split_words:"true"`
	ArgoWorkflowExecutorImage   string `required:"true" split_words:"true"`
	LitmusChaosOperatorImage    string `required:"true" split_words:"true"`
	LitmusChaosRunnerImage      string `required:"true" split_words:"true"`
	LitmusChaosExporterImage    string `required:"true" split_words:"true"`
	ContainerRuntimeExecutor    string `required:"true" split_words:"true"`
	HubBranchName               string `required:"true" split_words:"true"`
	RMQEndpoint                 string `required:"true" split_words:"true"`
	RMQPort                     string `required:"true" split_words:"true"`
	RMQAdminUser                string `required:"true" split_words:"true"`
	RMQAdminPassword            string `required:"true" split_words:"true"`
}

const defaultPort = "8080"

/*
	req, err := http.NewRequest("PUT", "http://"+RMQENDPOINT+":15672/api/vhosts/"+agentID+"-vhost", nil)
	if err != nil {
		return "", err
	}

	req.SetBasicAuth(RMQAdminUser, RMQAdminPassword)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}

	if resp.StatusCode == 200 {
		logrus.Print("Vhost created")
	}

*/
type PermissionPayload struct {
	Configure string `json:"configure"`
	Write     string `json:"write"`
	Read      string `json:"read"`
}

func rabbitMQInit(config Config) error {
	req, err := http.NewRequest("PUT", "http://"+config.RMQEndpoint+":"+config.RMQPort+"/api/vhosts/admin-vhost", nil)
	if err != nil {
		return err
	}

	req.SetBasicAuth(config.RMQAdminUser, config.RMQAdminPassword)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	logrus.Print(resp)

	if resp.StatusCode == 200 {
		logrus.Print("Admin Vhost created")
	}

	AgentPermissionData := PermissionPayload{
		Configure: ".*",
		Write:     ".*",
		Read:      ".*",
	}

	payloadBytes, err := json.Marshal(AgentPermissionData)
	if err != nil {
		return err
	}
	body := bytes.NewReader(payloadBytes)

	req, err = http.NewRequest("PUT", "http://"+config.RMQEndpoint+":15672/api/permissions/admin-vhost/"+config.RMQAdminUser, body)
	if err != nil {
		return err
	}

	req.SetBasicAuth(config.RMQAdminUser, config.RMQAdminPassword)
	req.Header.Set("Content-Type", "application/json")

	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	logrus.Print(resp)

	if resp.StatusCode == 200 {
		logrus.Print("Agent permission created(1)")
	}

	return nil
}

func init() {
	logrus.Printf("Go Version: %s", runtime.Version())
	logrus.Printf("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH)

	var c Config

	err := envconfig.Process("", &c)
	if err != nil {
		logrus.Fatal(err)
	}

	err = rabbitMQInit(c)
	if err != nil {
		logrus.Fatal(err)
	}
	// confirm version env is valid
	if !strings.Contains(strings.ToLower(c.Version), cluster.CIVersion) {
		splitCPVersion := strings.Split(c.Version, ".")
		if len(splitCPVersion) != 3 {
			logrus.Fatal("version doesn't follow semver semantic")
		}
	}
}

func validateVersion() error {
	currentVersion := os.Getenv("VERSION")
	dbVersion, err := config.GetConfig(context.Background(), "version")
	if err != nil {
		return fmt.Errorf("failed to get version from db, error = %w", err)
	}
	if dbVersion == nil {
		err := config.CreateConfig(context.Background(), &config.ServerConfig{Key: "version", Value: currentVersion})
		if err != nil {
			return fmt.Errorf("failed to insert current version in db, error = %w", err)
		}
		return nil
	}
	if dbVersion.Value.(string) != currentVersion {
		return fmt.Errorf("control plane needs to be upgraded from version %v to %v", dbVersion.Value.(string), currentVersion)
	}
	return nil
}

func main() {
	httpPort := os.Getenv("HTTP_PORT")
	if httpPort == "" {
		httpPort = utils.DefaultHTTPPort
	}

	rpcPort := os.Getenv("RPC_PORT")
	if rpcPort == "" {
		rpcPort = utils.DefaultRPCPort
	}

	// Initialize the mongo client
	mongodb.Client = mongodb.Client.Initialize()

	if err := validateVersion(); err != nil {
		logrus.Fatal(err)
	}

	go startGRPCServer(rpcPort) // start GRPC server

	srv := handler.New(generated.NewExecutableSchema(graph.NewConfig()))
	srv.AddTransport(transport.POST{})
	srv.AddTransport(transport.GET{})
	srv.AddTransport(transport.Websocket{
		KeepAlivePingInterval: 10 * time.Second,
		Upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
	})

	// to be removed in production
	srv.Use(extension.Introspection{})

	router := mux.NewRouter()

	router.Use(cors.New(cors.Options{
		AllowedHeaders:   []string{"*"},
		AllowedOrigins:   []string{"*"},
		AllowCredentials: true,
	}).Handler)

	gitOpsHandler.GitOpsSyncHandler(true) // sync all previous existing repos before start

	go myhub.RecurringHubSync()               // go routine for syncing hubs for all users
	go gitOpsHandler.GitOpsSyncHandler(false) // routine to sync git repos for gitOps

	router.Handle("/", playground.Handler("GraphQL playground", "/query"))
	router.Handle("/query", authorization.Middleware(srv))
	router.HandleFunc("/file/{key}{path:.yaml}", handlers.FileHandler)
	router.HandleFunc("/status", handlers.StatusHandler)

	router.Handle("/icon/{ProjectID}/{HubName}/{ChartName}/{IconName}", authorization.RestMiddlewareWithRole(myhub.GetIconHandler, nil)).Methods("GET")
	logrus.Printf("connect to http://localhost:%s/ for GraphQL playground", httpPort)
	logrus.Fatal(http.ListenAndServe(":"+httpPort, router))
}

// startGRPCServer initializes, registers services to and starts the gRPC server for RPC calls
func startGRPCServer(port string) {
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		logrus.Fatal("failed to listen: %w", err)
	}

	grpcServer := grpc.NewServer()

	// Register services
	pb.RegisterProjectServer(grpcServer, &projects.ProjectServer{})

	logrus.Printf("GRPC server listening on %v", lis.Addr())
	logrus.Fatal(grpcServer.Serve(lis))
}
