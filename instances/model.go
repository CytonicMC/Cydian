package instances

type InstanceCreateRequest struct {
	InstanceType string `json:"instanceType"`
	Quantity     int    `json:"quantity"`
}

type InstanceCreateResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}
