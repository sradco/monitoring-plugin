package managementrouter

type DeleteUserDefinedAlertRulesResponse struct {
	Id         string `json:"id"`
	StatusCode int    `json:"status_code"`
	Message    string `json:"message,omitempty"`
}
