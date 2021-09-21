package transit

// HostServiceInDowntime describes downtime schedule
type HostServiceInDowntime struct {
	HostName               string `json:"hostName"`
	ServiceDescription     string `json:"serviceDescription,omitempty"`
	ScheduledDowntimeDepth int    `json:"scheduledDowntimeDepth"`
	EntityType             string `json:"entityType"`
	EntityName             string `json:"entityName"`
}

// HostServicesInDowntime defines type used for ClearInDowntime API payload
type HostServicesInDowntime struct {
	BizHostServiceInDowntimes []HostServiceInDowntime `json:"bizHostServiceInDowntimes"`
}

// HostsAndServices defines type used for SetInDowntime API payload
type HostsAndServices struct {
	HostNames                 []string `json:"hostNames"`
	ServiceDescriptions       []string `json:"serviceDescriptions"`
	HostGroupNames            []string `json:"hostGroupNames"`
	ServiceGroupCategoryNames []string `json:"serviceGroupCategoryNames"`
	SetHosts                  bool     `json:"setHosts"`
	SetServices               bool     `json:"setServices"`
}
