package events

import (
	"time"
)

type EventType string

const (
	// Monitor events
	MonitorCreated       EventType = "monitor.created"
	MonitorUpdated       EventType = "monitor.updated"
	MonitorDeleted       EventType = "monitor.deleted"
	MonitorPaused        EventType = "monitor.paused"
	MonitorResumed       EventType = "monitor.resumed"
	MonitorTested        EventType = "monitor.tested"
	MonitorResultCreated EventType = "monitor.result_created"
	MonitorStatusChanged EventType = "monitor.status_changed"

	// Incident events
	IncidentOpened       EventType = "incident.opened"
	IncidentAcknowledged EventType = "incident.acknowledged"
	IncidentResolved     EventType = "incident.resolved"

	// Notification events
	NotificationSent   EventType = "notification.delivery_sent"
	NotificationFailed EventType = "notification.delivery_failed"

	// Discovery events
	DiscoveryScanStarted   EventType = "discovery.scan_started"
	DiscoveryScanCompleted EventType = "discovery.scan_completed"
	DiscoveryScanFailed    EventType = "discovery.scan_failed"
	DiscoveryResultCreated EventType = "discovery.result_created"
	DiscoveryResultAccepted EventType = "discovery.result_accepted"
	DiscoveryResultIgnored  EventType = "discovery.result_ignored"
	DiscoveryNewHost       EventType = "discovery.new_host"
	DiscoveryConflictDetected EventType = "discovery.conflict_detected"
	DiscoveryStaleDetected    EventType = "discovery.stale_detected"

	// IPAM events
	IPAMSiteChanged      EventType = "ipam.site_changed"
	IPAMVlanChanged      EventType = "ipam.vlan_changed"
	IPAMPrefixChanged    EventType = "ipam.prefix_changed"
	IPAMIPAddressChanged EventType = "ipam.ip_address_changed"
	IPAMDeviceChanged    EventType = "ipam.device_changed"

	// Status Page events
	StatusPageChanged        EventType = "status_page.changed"
	StatusPageMonitorChanged EventType = "status_page.monitor_changed"

	// Auth/Audit events
	AuditEntryCreated EventType = "audit.entry_created"
	UserChanged       EventType = "user.changed"
	APITokenChanged   EventType = "api_token.changed"
)

type Event struct {
	ID         string         `json:"id"`
	Type       EventType      `json:"type"`
	CreatedAt  time.Time      `json:"created_at"`
	ActorType  string         `json:"actor_type,omitempty"` // user, api_token, system
	ActorID    string         `json:"actor_id,omitempty"`
	ObjectType string         `json:"object_type,omitempty"`
	ObjectID   string         `json:"object_id,omitempty"`
	Payload    map[string]any `json:"payload"`
}

type EventFilter struct {
	Types       []EventType `json:"types"`
	ObjectTypes []string    `json:"object_types"`
}
