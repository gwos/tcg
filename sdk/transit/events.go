package transit

import (
	"fmt"
)

// IncidentAlert describes alerts received from cloud services.
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

// String implements Stringer interface.
func (p IncidentAlert) String() string {
	return fmt.Sprintf("[%s, %s, %s, %s, %s, %s, %s, %s]",
		p.IncidentID, p.ResourceName, p.Status, p.StartedAt.String(),
		p.EndedAt.String(), p.ConditionName, p.URL, p.Summary,
	)
}

// GroundworkEventsRequest describes request payload.
type GroundworkEventsRequest struct {
	Events []GroundworkEvent `json:"events"`
}

// String implements Stringer interface.
func (p GroundworkEventsRequest) String() string {
	return fmt.Sprintf("[%s]", p.Events)
}

// GroundworkEvent describes event.
type GroundworkEvent struct {
	AppType             string     `json:"appType"`
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
	// Update level attributes (update only)
	ApplicationName   string `json:"applicationName,omitempty"`
	ConsolidationName string `json:"consolidationName,omitempty"`
	ErrorType         string `json:"errorType,omitempty"`
	LoggerName        string `json:"loggerName,omitempty"`
	LogType           string `json:"logType,omitempty"`
	MonitorServer     string `json:"monitorServer,omitempty"`
}

// String implements Stringer interface.
func (p GroundworkEvent) String() string {
	return fmt.Sprintf("[%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s]",
		p.AppType, p.Device, p.Host, p.Service,
		p.OperationStatus, p.MonitorStatus, p.Severity, p.ApplicationSeverity,
		p.Component, p.SubComponent, p.Priority, p.TypeRule,
		p.TextMessage, p.LastInsertDate.String(), p.ReportDate.String(),
		p.ApplicationName, p.ConsolidationName, p.ErrorType,
		p.LoggerName, p.LogType, p.MonitorServer,
	)
}

// GroundworkEventsAckRequest describes request payload.
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

// String implements Stringer interface.
func (p GroundworkEventAck) String() string {
	return fmt.Sprintf("[%s, %s, %s, %s, %s]",
		p.AppType, p.Host, p.Service,
		p.AcknowledgedBy, p.AcknowledgeComment,
	)
}

// GroundworkEventsUnackRequest describes request payload.
type GroundworkEventsUnackRequest struct {
	Unacks []GroundworkEventUnack `json:"unacks"`
}

// GroundworkEventUnack describes event ack
type GroundworkEventUnack struct {
	AppType string `json:"appType"`
	Host    string `json:"host"`
	Service string `json:"service,omitempty"`
}

// String implements Stringer interface.
func (p GroundworkEventUnack) String() string {
	return fmt.Sprintf("[%s, %s, %s]",
		p.AppType, p.Host, p.Service)
}
