//go:build !gcloud

package taskqueue

type PrimindTaskRequest struct {
	Task PrimindTask `json:"task"`
}

type PrimindTask struct {
	HTTPRequest PrimindHTTPRequest `json:"httpRequest"`
}

type PrimindHTTPRequest struct {
	Body    string            `json:"body"`
	Headers map[string]string `json:"headers,omitempty"`
}

type PrimindTaskResponse struct {
	Name       string `json:"name"`
	CreateTime string `json:"createTime"`
}
