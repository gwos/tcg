package transit

// Downtime describes downtime schedule.
type Downtime struct {
	EntityType             string `json:"entityType"`
	EntityName             string `json:"entityName"`
	HostName               string `json:"hostName"`
	ServiceDescription     string `json:"serviceDescription,omitempty"`
	ScheduledDowntimeDepth int    `json:"scheduledDowntimeDepth"`
}

// Downtimes defines type used for ClearInDowntime API payload.
type Downtimes struct {
	BizHostServiceInDowntimes []Downtime `json:"bizHostServiceInDowntimes"`
}

// DowntimesRequest defines type used for SetInDowntime API payload.
type DowntimesRequest struct {
	HostNames                 []string `json:"hostNames"`
	HostGroupNames            []string `json:"hostGroupNames"`
	ServiceDescriptions       []string `json:"serviceDescriptions"`
	ServiceGroupCategoryNames []string `json:"serviceGroupCategoryNames"`
	SetHosts                  bool     `json:"setHosts"`
	SetServices               bool     `json:"setServices"`
}
