package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/google/uuid"
)

//const (
//	// DefaultBasePath is the default base path for the API.
//	DefaultBasePath = "/"
//	// DefaultHost is the default host for the API.
//	DefaultHost = "51.44.28.47:30015"
//	// DefaultSchemes is the default scheme for the API.
//	DefaultSchemes = "http"
//)

// WorkloadActionTypeEnum represents the type of workload action.
type WorkloadActionTypeEnum string

const (
	WorkloadActionTypeEnumBind   WorkloadActionTypeEnum = "bind"
	WorkloadActionTypeEnumCreate WorkloadActionTypeEnum = "create"
	WorkloadActionTypeEnumDelete WorkloadActionTypeEnum = "delete"
	WorkloadActionTypeEnumMove   WorkloadActionTypeEnum = "move"
	WorkloadActionTypeEnumSwapX  WorkloadActionTypeEnum = "swap_x"
	WorkloadActionTypeEnumSwapY  WorkloadActionTypeEnum = "swap_y"
)

// WorkloadActionStatusEnum represents the status of a workload action.
type WorkloadActionStatusEnum string

const (
	WorkloadActionStatusEnumPending   WorkloadActionStatusEnum = "pending"
	WorkloadActionStatusEnumSucceeded WorkloadActionStatusEnum = "succeeded"
	WorkloadActionStatusEnumFailed    WorkloadActionStatusEnum = "failed"
)

// PodParentTypeEnum represents the type of a pod's parent resource.
type PodParentTypeEnum string

const (
	PodParentTypeEnumDeployment  PodParentTypeEnum = "deployment"
	PodParentTypeEnumStatefulset PodParentTypeEnum = "statefulset"
	PodParentTypeEnumReplicaset  PodParentTypeEnum = "replicaset"
	PodParentTypeEnumJob         PodParentTypeEnum = "job"
	PodParentTypeEnumDaemonset   PodParentTypeEnum = "daemonset"
	PodParentTypeEnumCronjob     PodParentTypeEnum = "cronjob"
)

// WorkloadAction is a schema for a workload action.
type WorkloadAction struct {
	ID                  uuid.UUID                `json:"id"`
	ActionType          WorkloadActionTypeEnum   `json:"action_type"`
	ActionStatus        WorkloadActionStatusEnum `json:"action_status"`
	ActionStartTime     time.Time                `json:"action_start_time"`
	ActionEndTime       *time.Time               `json:"action_end_time"`
	ActionReason        *string                  `json:"action_reason"`
	PodParentName       *string                  `json:"pod_parent_name"`
	PodParentType       *PodParentTypeEnum       `json:"pod_parent_type"`
	PodParentUID        *uuid.UUID               `json:"pod_parent_uid"`
	CreatedPodName      *string                  `json:"created_pod_name"`
	CreatedPodNamespace *string                  `json:"created_pod_namespace"`
	CreatedNodeName     *string                  `json:"created_node_name"`
	DeletedPodName      *string                  `json:"deleted_pod_name"`
	DeletedPodNamespace *string                  `json:"deleted_pod_namespace"`
	DeletedNodeName     *string                  `json:"deleted_node_name"`
	BoundPodName        *string                  `json:"bound_pod_name"`
	BoundPodNamespace   *string                  `json:"bound_pod_namespace"`
	BoundNodeName       *string                  `json:"bound_node_name"`
	CreatedAt           time.Time                `json:"created_at"`
	UpdatedAt           time.Time                `json:"updated_at"`
}

// WorkloadActionCreate is a schema for creating a workload action.
type WorkloadActionCreate struct {
	ActionType          WorkloadActionTypeEnum   `json:"action_type,"`
	ActionStatus        WorkloadActionStatusEnum `json:"action_status"`
	ActionStartTime     time.Time                `json:"action_start_time"`
	ActionEndTime       *time.Time               `json:"action_end_time,omitempty"`
	ActionReason        *string                  `json:"action_reason,omitempty"`
	PodParentName       *string                  `json:"pod_parent_name,omitempty"`
	PodParentType       *PodParentTypeEnum       `json:"pod_parent_type,omitempty"`
	PodParentUID        *uuid.UUID               `json:"pod_parent_uid,omitempty"`
	CreatedPodName      *string                  `json:"created_pod_name,omitempty"`
	CreatedPodNamespace *string                  `json:"created_pod_namespace,omitempty"`
	CreatedNodeName     *string                  `json:"created_node_name,omitempty"`
	DeletedPodName      *string                  `json:"deleted_pod_name,omitempty"`
	DeletedPodNamespace *string                  `json:"deleted_pod_namespace,omitempty"`
	DeletedNodeName     *string                  `json:"deleted_node_name,omitempty"`
	BoundPodName        *string                  `json:"bound_pod_name,omitempty"`
	BoundPodNamespace   *string                  `json:"bound_pod_namespace,omitempty"`
	BoundNodeName       *string                  `json:"bound_node_name,omitempty"`
	CreatedAt           time.Time                `json:"created_at"`
	UpdatedAt           time.Time                `json:"updated_at"`
}

// WorkloadActionUpdate is a schema for updating a workload action.
type WorkloadActionUpdate struct {
	ActionType          *WorkloadActionTypeEnum   `json:"action_type,omitempty"`
	ActionStatus        *WorkloadActionStatusEnum `json:"action_status,omitempty"`
	ActionStartTime     *time.Time                `json:"action_start_time,omitempty"`
	ActionEndTime       *time.Time                `json:"action_end_time,omitempty"`
	ActionReason        *string                   `json:"action_reason,omitempty"`
	PodParentName       *string                   `json:"pod_parent_name,omitempty"`
	PodParentType       *PodParentTypeEnum        `json:"pod_parent_type,omitempty"`
	PodParentUID        *uuid.UUID                `json:"pod_parent_uid,omitempty"`
	CreatedPodName      *string                   `json:"created_pod_name,omitempty"`
	CreatedPodNamespace *string                   `json:"created_pod_namespace,omitempty"`
	CreatedNodeName     *string                   `json:"created_node_name,omitempty"`
	DeletedPodName      *string                   `json:"deleted_pod_name,omitempty"`
	DeletedPodNamespace *string                   `json:"deleted_pod_namespace,omitempty"`
	DeletedNodeName     *string                   `json:"deleted_node_name,omitempty"`
	BoundPodName        *string                   `json:"bound_pod_name,omitempty"`
	BoundPodNamespace   *string                   `json:"bound_pod_namespace,omitempty"`
	BoundNodeName       *string                   `json:"bound_node_name,omitempty"`
	UpdatedAt           *time.Time                `json:"updated_at,omitempty"`
}

type WorkloadDecisionStatusUpdate struct {
	PodName        string                   `json:"pod_name"`
	Namespace      string                   `json:"namespace"`
	NodeName       string                   `json:"node_name"`
	ActionType     WorkloadActionTypeEnum   `json:"action_type"`
	DecisionStatus WorkloadActionStatusEnum `json:"decision_status"`
}

// ValidationError is a schema for a validation error.
type ValidationError struct {
	Loc  []interface{} `json:"loc"`
	Msg  string        `json:"msg"`
	Type string        `json:"type"`
}

type WALogger struct {
	BasePath   string
	Host       string
	Schemes    []string
	UserAgent  string
	HTTPClient *http.Client
}

func NewWALogger(scheme string, host string) *WALogger {
	return &WALogger{
		BasePath:   "/",
		Host:       host,
		Schemes:    []string{scheme},
		UserAgent:  "wam",
		HTTPClient: http.DefaultClient,
	}
}

// SetHTTPClient sets the HTTP client for the API client.
func (c *WALogger) SetHTTPClient(httpClient *http.Client) {
	c.HTTPClient = httpClient
}

// NewRequest creates a new HTTP request with the provided method, path, and body.
func (c *WALogger) NewRequest(method, path string, body interface{}) (*http.Request, error) {
	u, err := url.Parse(path)
	if err != nil {
		return nil, err
	}

	if u.Host == "" {
		u.Host = c.Host
	}
	if u.Scheme == "" {
		u.Scheme = c.Schemes[0]
	}

	var reqBody io.Reader
	if body != nil {
		buf := new(bytes.Buffer)
		err = json.NewEncoder(buf).Encode(body)
		if err != nil {
			return nil, err
		}
		reqBody = buf
	}

	req, err := http.NewRequest(method, u.String(), reqBody)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.UserAgent)

	return req, nil
}

// Do performs the HTTP request and decodes the response body into the provided v interface.
func (c *WALogger) Do(ctx context.Context, req *http.Request, v interface{}) (*http.Response, error) {
	req = req.WithContext(ctx)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error: %s", resp.Status)
	}

	if v != nil {
		if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
			return nil, err
		}
	}

	return resp, nil
}

func (c *WALogger) CreateWorkloadAction(ctx context.Context, body WorkloadActionCreate) (*WorkloadAction, *http.Response, error) {
	path := "/workload_action/"
	req, err := c.NewRequest(http.MethodPost, path, body)
	if err != nil {
		return nil, nil, err
	}

	var result WorkloadAction
	resp, err := c.Do(ctx, req, &result)
	if err != nil {
		return nil, resp, err
	}

	return &result, resp, nil
}

// GetAllWorkloadActionsRouteWorkloadActionGetParams holds the parameters for GetAllWorkloadActionsRouteWorkloadActionGet.
type GetAllWorkloadActionsRouteWorkloadActionGetParams struct {
	ActionType          *WorkloadActionTypeEnum
	ActionStatus        *WorkloadActionStatusEnum
	ActionStartTime     *time.Time
	ActionEndTime       *time.Time
	ActionReason        *string
	PodParentName       *string
	PodParentType       *PodParentTypeEnum
	PodParentUID        *uuid.UUID
	CreatedPodName      *string
	CreatedPodNamespace *string
	CreatedNodeName     *string
	DeletedPodName      *string
	DeletedPodNamespace *string
	DeletedNodeName     *string
	BoundPodName        *string
	BoundPodNamespace   *string
	BoundNodeName       *string
}

func (c *WALogger) GetAllWorkloadActions(ctx context.Context, params *GetAllWorkloadActionsRouteWorkloadActionGetParams) ([]WorkloadAction, *http.Response, error) {
	path := "/workload_action/"
	u, err := url.Parse(path)
	if err != nil {
		return nil, nil, err
	}

	q := u.Query()
	if params != nil {
		if params.ActionType != nil {
			q.Add("action_type", string(*params.ActionType))
		}
		if params.ActionStatus != nil {
			q.Add("action_status", string(*params.ActionStatus))
		}
		if params.ActionStartTime != nil {
			q.Add("action_start_time", params.ActionStartTime.Format(time.RFC3339))
		}
		if params.ActionEndTime != nil {
			q.Add("action_end_time", params.ActionEndTime.Format(time.RFC3339))
		}
		if params.ActionReason != nil {
			q.Add("action_reason", *params.ActionReason)
		}
		if params.PodParentName != nil {
			q.Add("pod_parent_name", *params.PodParentName)
		}
		if params.PodParentType != nil {
			q.Add("pod_parent_type", string(*params.PodParentType))
		}
		if params.PodParentUID != nil {
			q.Add("pod_parent_uid", params.PodParentUID.String())
		}
		if params.CreatedPodName != nil {
			q.Add("created_pod_name", *params.CreatedPodName)
		}
		if params.CreatedPodNamespace != nil {
			q.Add("created_pod_namespace", *params.CreatedPodNamespace)
		}
		if params.CreatedNodeName != nil {
			q.Add("created_node_name", *params.CreatedNodeName)
		}
		if params.DeletedPodName != nil {
			q.Add("deleted_pod_name", *params.DeletedPodName)
		}
		if params.DeletedPodNamespace != nil {
			q.Add("deleted_pod_namespace", *params.DeletedPodNamespace)
		}
		if params.DeletedNodeName != nil {
			q.Add("deleted_node_name", *params.DeletedNodeName)
		}
		if params.BoundPodName != nil {
			q.Add("bound_pod_name", *params.BoundPodName)
		}
		if params.BoundPodNamespace != nil {
			q.Add("bound_pod_namespace", *params.BoundPodNamespace)
		}
		if params.BoundNodeName != nil {
			q.Add("bound_node_name", *params.BoundNodeName)
		}
	}
	u.RawQuery = q.Encode()

	req, err := c.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, nil, err
	}

	var result []WorkloadAction
	resp, err := c.Do(ctx, req, &result)
	if err != nil {
		return nil, resp, err
	}

	return result, resp, nil
}

func (c *WALogger) GetWorkloadAction(ctx context.Context, actionID uuid.UUID) (*WorkloadAction, *http.Response, error) {
	path := fmt.Sprintf("/workload_action/%s", actionID.String())
	req, err := c.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}

	var result WorkloadAction
	resp, err := c.Do(ctx, req, &result)
	if err != nil {
		return nil, resp, err
	}

	return &result, resp, nil
}

func (c *WALogger) UpdateWorkloadAction(ctx context.Context, actionID uuid.UUID, body WorkloadActionUpdate) (*WorkloadAction, *http.Response, error) {
	path := fmt.Sprintf("/workload_action/%s", actionID.String())
	req, err := c.NewRequest(http.MethodPut, path, body)
	if err != nil {
		return nil, nil, err
	}

	var result WorkloadAction
	resp, err := c.Do(ctx, req, &result)
	if err != nil {
		return nil, resp, err
	}

	return &result, resp, nil
}

func (c *WALogger) DeleteWorkloadAction(ctx context.Context, actionID uuid.UUID) (*http.Response, error) {
	path := fmt.Sprintf("/workload_action/%s", actionID.String())
	req, err := c.NewRequest(http.MethodDelete, path, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.Do(ctx, req, nil)
	if err != nil {
		return resp, err
	}

	return resp, nil
}

func (c *WALogger) UpdateWorkloadDecisionStatus(ctx context.Context, body WorkloadDecisionStatusUpdate) (*http.Response, error) {
	path := "/workload_request_decision/status"
	req, err := c.NewRequest(http.MethodPut, path, body)
	if err != nil {
		return nil, err
	}

	var result WorkloadAction
	resp, err := c.Do(ctx, req, &result)
	if err != nil {
		return resp, err
	}

	return resp, nil
}

//func main() {
//	c := NewWALogger("http", "51.44.28.47:30015")
//	res, _, err := c.GetAllWorkloadActions(context.Background(), nil)
//	if err != nil {
//		fmt.Println(err)
//		return
//	}
//
//	prettyJSON, err := json.MarshalIndent(res, "", "  ")
//	if err != nil {
//		log.Fatalf("Error marshaling to pretty JSON: %v", err)
//	}
//
//	fmt.Println("--- Pretty JSON Output ---")
//	fmt.Println(string(prettyJSON))
//
//	now := time.Now()
//
//	res3, _, err := c.CreateWorkloadAction(context.Background(), WorkloadActionCreate{
//		ActionType:          WorkloadActionTypeEnumDelete,
//		ActionStatus:        WorkloadActionStatusEnumPending,
//		ActionStartTime:     now,
//		ActionEndTime:       nil,
//		ActionReason:        nil,
//		PodParentName:       nil,
//		PodParentType:       nil,
//		PodParentUID:        nil,
//		CreatedPodName:      nil,
//		CreatedPodNamespace: nil,
//		CreatedNodeName:     nil,
//		DeletedPodName:      nil,
//		DeletedPodNamespace: nil,
//		DeletedNodeName:     nil,
//		BoundPodName:        nil,
//		BoundPodNamespace:   nil,
//		BoundNodeName:       nil,
//		CreatedAt:           now,
//		UpdatedAt:           now,
//	})
//	if err != nil {
//		fmt.Println(err)
//		return
//	}
//
//	prettyJSON3, err := json.MarshalIndent(res3, "", "  ")
//	if err != nil {
//		log.Fatalf("Error marshaling to pretty JSON: %v", err)
//	}
//
//	fmt.Println("--- Pretty JSON Output ---")
//	fmt.Println(string(prettyJSON3))
//
//	res2, _, err := c.GetWorkloadAction(context.Background(), res3.ID)
//	if err != nil {
//		fmt.Println(err)
//		return
//	}
//
//	prettyJSON2, err := json.MarshalIndent(res2, "", "  ")
//	if err != nil {
//		log.Fatalf("Error marshaling to pretty JSON: %v", err)
//	}
//
//	fmt.Println("--- Pretty JSON Output ---")
//	fmt.Println(string(prettyJSON2))
//
//	now2 := time.Now()
//	x := WorkloadActionStatusEnumFailed
//	k := uuid.MustParse("7f347b06-4d7b-4f64-8524-a8941a3ab3d1")
//	res4, _, err := c.UpdateWorkloadAction(context.Background(), res2.ID, WorkloadActionUpdate{
//		ActionType:          nil,
//		ActionStatus:        &x,
//		ActionStartTime:     &now2,
//		ActionEndTime:       &now2,
//		ActionReason:        nil,
//		PodParentName:       nil,
//		PodParentType:       nil,
//		PodParentUID:        &k,
//		CreatedPodName:      nil,
//		CreatedPodNamespace: nil,
//		CreatedNodeName:     nil,
//		DeletedPodName:      nil,
//		DeletedPodNamespace: nil,
//		DeletedNodeName:     nil,
//		BoundPodName:        nil,
//		BoundPodNamespace:   nil,
//		BoundNodeName:       nil,
//		UpdatedAt:           &now2,
//	})
//	if err != nil {
//		fmt.Println(err)
//		return
//	}
//
//	prettyJSON4, err := json.MarshalIndent(res4, "", "  ")
//	if err != nil {
//		log.Fatalf("Error marshaling to pretty JSON: %v", err)
//	}
//
//	fmt.Println("--- Pretty JSON Output ---")
//	fmt.Println(string(prettyJSON4))
//}
