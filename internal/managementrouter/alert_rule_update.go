package managementrouter

type UpdateAlertRuleResponse struct {
	Id         string `json:"id"`
	StatusCode int    `json:"status_code"`
	Message    string `json:"message,omitempty"`
}
