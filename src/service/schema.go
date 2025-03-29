package service

type CommonLabels struct {
	Alertname     string `json:"alertname"`
	GrafanaFolder string `json:"grafana_folder"`
	Phone         string `json:"phone"`
	RefID         string `json:"ref_id"`
}

// GrafanaAlert represents an alert from Grafana.
// It contains information about the alert receiver, status, common labels, state, title, and message.
type GrafanaAlert struct {
	Receiver     string       `json:"receiver"`
	Status       string       `json:"status"`
	CommonLabels CommonLabels `json:"commonLabels"`
	State        string       `json:"state"`
	Title        string       `json:"title"`
	Message      string       `json:"message"`
}
