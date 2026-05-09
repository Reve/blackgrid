export type EventType =
  | 'monitor.status_changed'
  | 'monitor.result_created'
  | 'monitor.changed'
  | 'incident.opened'
  | 'incident.acknowledged'
  | 'incident.resolved'
  | 'discovery.scan_started'
  | 'discovery.scan_completed'
  | 'discovery.scan_failed'
  | 'discovery.result_created'
  | 'discovery.new_host'
  | 'discovery.stale_detected'
  | 'discovery.conflict_detected'
  | 'discovery.result_accepted'
  | 'discovery.result_ignored'
  | 'site.changed'
  | 'vlan.changed'
  | 'prefix.changed'
  | 'ip_address.changed'
  | 'device.changed'
  | 'user.changed'
  | 'api_token.changed'
  | 'audit.entry_created'
  | 'notification.sent'
  | 'notification.failed'
  | 'status_page.changed'
  | 'status_page.monitor_changed';

export interface Event {
  id: string;
  type: EventType;
  object_type: string;
  object_id?: string;
  payload: Record<string, any>;
  timestamp: string;
}
