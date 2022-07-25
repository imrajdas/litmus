package handlers

import (
	"encoding/json"
	"github.com/litmuschaos/litmus/litmus-portal/graphql-server/utils"
	"github.com/sirupsen/logrus"
	"net/http"
	"os"
)

type Version struct {
	WorkflowHelperImageVersion    string `json:"workflow_helper_image_version"`
	ChaosOperatorVersion          string `json:"chaos_operator_version"`
	ChaosExporterVersion          string `json:"chaos_exporter_version"`
	ChaosRunnerVersion            string `json:"chaos_runner_version"`
	SubscriberVersion             string `json:"subscriber_version"`
	EventTrackerVersion           string `json:"event_tracker_version"`
	ArgoWorkflowExecutorVersion   string `json:"argo_workflow_executor_version"`
	ArgoWorkflowControllerVersion string `json:"argo_workflow_controller_version"`
	GraphQLServerVersion          string `json:"graphql_server_version"`
}

//- name: SUBSCRIBER_IMAGE
//value: "litmuschaos/litmusportal-subscriber:ci"
//- name: EVENT_TRACKER_IMAGE
//value: "litmuschaos/litmusportal-event-tracker:ci"
//- name: ARGO_WORKFLOW_CONTROLLER_IMAGE
//value: "litmuschaos/workflow-controller:v3.2.9"
//- name: ARGO_WORKFLOW_EXECUTOR_IMAGE
//value: "litmuschaos/argoexec:v3.2.9"
//- name: LITMUS_CHAOS_OPERATOR_IMAGE
//value: "litmuschaos/chaos-operator:2.8.0"
//- name: LITMUS_CHAOS_RUNNER_IMAGE
//value: "litmuschaos/chaos-runner:2.8.0"
//- name: LITMUS_CHAOS_EXPORTER_IMAGE
//value: "litmuschaos/chaos-exporter:2.8.0"

func VersionHandler(w http.ResponseWriter, r *http.Request) {
	versionDetails := os.Getenv("WORKFLOW_HELPER_IMAGE_VERSION")
	version := WorkflowHelperImageVersion{WorkflowHelperImageVersion: versionDetails}
	versionByte, err := json.Marshal(version)
	if err != nil {
		logrus.Error(err)
		utils.WriteHeaders(&w, http.StatusBadRequest)
		return
	}

	utils.WriteHeaders(&w, http.StatusOK)
	w.Write(versionByte)
}
