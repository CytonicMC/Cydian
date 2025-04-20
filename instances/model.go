package instances

type InstanceCreateRequest struct {
	InstanceType string `json:"instanceType"`
	Quantity     int    `json:"quantity"`
}

type InstanceResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type InstanceDeleteAllRequest struct {
	InstanceType string `json:"instanceType"`
}

type InstanceDeleteRequest struct {
	InstanceType string `json:"instanceType"`
	AllocId      string `json:"allocId"`
}
