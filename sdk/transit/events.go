package transit

import (
	"fmt"
)

// IncidentAlert describes alerts received from cloud services
type IncidentAlert struct {
	IncidentID    string     `json:"incidentId"`
	ResourceName  string     `json:"resourceName"`
	Status        string     `json:"status"`
	StartedAt     *Timestamp `json:"startedAt"`
	EndedAt       *Timestamp `json:"endedAt,omitempty"`
	ConditionName string     `json:"conditionName"`
	URL           string     `json:"url,omitempty"`
	Summary       string     `json:"summary,omitempty"`
}

// String implements Stringer interface
func (incidentAlert IncidentAlert) String() string {
	return fmt.Sprintf("[%s, %s, %s, %s, %s, %s, %s, %s]",
		incidentAlert.IncidentID, incidentAlert.ResourceName, incidentAlert.Status, incidentAlert.StartedAt.String(),
		incidentAlert.EndedAt.String(), incidentAlert.ConditionName, incidentAlert.URL, incidentAlert.Summary,
	)
}

// GroundworkEventsRequest describes request payload
type GroundworkEventsRequest struct {
	Events []GroundworkEvent `json:"events"`
}

// String implements Stringer interface
func (groundworkEventsRequest GroundworkEventsRequest) String() string {
	return fmt.Sprintf("[%s]", groundworkEventsRequest.Events)
}

// GroundworkEvent describes event
type GroundworkEvent struct {
	Device              string     `json:"device,omitempty"`
	Host                string     `json:"host"`
	Service             string     `json:"service,omitempty"`
	OperationStatus     string     `json:"operationStatus,omitempty"`
	MonitorStatus       string     `json:"monitorStatus"`
	Severity            string     `json:"severity,omitempty"`
	ApplicationSeverity string     `json:"applicationSeverity,omitempty"`
	Component           string     `json:"component,omitempty"`
	SubComponent        string     `json:"subComponent,omitempty"`
	Priority            string     `json:"priority,omitempty"`
	TypeRule            string     `json:"typeRule,omitempty"`
	TextMessage         string     `json:"textMessage,omitempty"`
	LastInsertDate      *Timestamp `json:"lastInsertDate,omitempty"`
	ReportDate          *Timestamp `json:"reportDate"`
	AppType             string     `json:"appType"`
	// Update level attributes (update only)
	MonitorServer     string `json:"monitorServer,omitempty"`
	ConsolidationName string `json:"consolidationName,omitempty"`
	LogType           string `json:"logType,omitempty"`
	ErrorType         string `json:"errorType,omitempty"`
	LoggerName        string `json:"loggerName,omitempty"`
	ApplicationName   string `json:"applicationName,omitempty"`
}

// String implements Stringer interface
func (groundworkEvent GroundworkEvent) String() string {
	return fmt.Sprintf("[%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s]",
		groundworkEvent.Device, groundworkEvent.Host, groundworkEvent.Service, groundworkEvent.OperationStatus,
		groundworkEvent.MonitorStatus, groundworkEvent.Severity, groundworkEvent.ApplicationSeverity,
		groundworkEvent.Component, groundworkEvent.SubComponent, groundworkEvent.Priority, groundworkEvent.TypeRule,
		groundworkEvent.TextMessage, groundworkEvent.LastInsertDate.String(), groundworkEvent.ReportDate.String(),
		groundworkEvent.AppType, groundworkEvent.MonitorServer, groundworkEvent.ConsolidationName,
		groundworkEvent.LogType, groundworkEvent.ErrorType, groundworkEvent.LoggerName, groundworkEvent.ApplicationName,
	)
}

// GroundworkEventsAckRequest describes request payload
type GroundworkEventsAckRequest struct {
	Acks []GroundworkEventAck `json:"acks"`
}

// GroundworkEventAck describes event ack
type GroundworkEventAck struct {
	AppType            string `json:"appType"`
	Host               string `json:"host"`
	Service            string `json:"service,omitempty"`
	AcknowledgedBy     string `json:"acknowledgedBy,omitempty"`
	AcknowledgeComment string `json:"acknowledgeComment,omitempty"`
}

// String implements Stringer interface
func (groundworkEventAck GroundworkEventAck) String() string {
	return fmt.Sprintf("[%s, %s, %s, %s, %s]",
		groundworkEventAck.AppType, groundworkEventAck.Host, groundworkEventAck.Service,
		groundworkEventAck.AcknowledgedBy, groundworkEventAck.AcknowledgeComment,
	)
}

// GroundworkEventsUnackRequest describes request payload
type GroundworkEventsUnackRequest struct {
	Unacks []GroundworkEventUnack `json:"unacks"`
}

// GroundworkEventUnack describes event ack
type GroundworkEventUnack struct {
	AppType string `json:"appType"`
	Host    string `json:"host"`
	Service string `json:"service,omitempty"`
}

// String implements Stringer interface
func (groundworkEventUnack GroundworkEventUnack) String() string {
	return fmt.Sprintf("[%s, %s, %s]",
		groundworkEventUnack.AppType, groundworkEventUnack.Host, groundworkEventUnack.Service)
}
