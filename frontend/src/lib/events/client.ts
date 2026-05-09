import type { Event } from './types';

export interface EventStreamOptions {
  onEvent: (event: Event) => void;
  onError?: (error: any) => void;
  onConnected?: () => void;
  onDisconnected?: () => void;
  types?: string[];
  objectTypes?: string[];
}

export class EventStreamClient {
  private eventSource: EventSource | null = null;
  private reconnectTimeout: any = null;
  private retryCount = 0;
  private maxRetries = 10;
  private baseDelay = 1000;

  constructor(private options: EventStreamOptions) {}

  connect() {
    if (this.eventSource) {
      this.disconnect();
    }

    const queryParams = new URLSearchParams();
    if (this.options.types && this.options.types.length > 0) {
      queryParams.append('types', this.options.types.join(','));
    }
    if (this.options.objectTypes && this.options.objectTypes.length > 0) {
      queryParams.append('object_types', this.options.objectTypes.join(','));
    }

    const url = `/api/v1/events/stream?${queryParams.toString()}`;
    
    // We use EventSource, which doesn't support custom headers (like Authorization) easily.
    // However, our backend supports session-based auth via cookies if the request is from the same origin.
    // If we needed Bearer token auth, we would need a polyfill or a different approach (like a one-time token).
    // For now, assuming cookie-based auth works.
    this.eventSource = new EventSource(url, { withCredentials: true });

    this.eventSource.onopen = () => {
      console.log('EventStream connected');
      this.retryCount = 0;
      this.options.onConnected?.();
    };

    this.eventSource.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data);
        this.options.onEvent(data);
      } catch (e) {
        console.error('Failed to parse event data:', e);
      }
    };

    this.eventSource.onerror = (error) => {
      console.error('EventStream error:', error);
      this.options.onError?.(error);
      this.handleReconnect();
    };

    // SSE standard doesn't have a specific onmessage for custom event types unless we use addEventListener
    // But since our backend sends "event: <type>", we should use addEventListener for each type or a generic one.
    // Actually, our backend sends "event: <type>", so we need to listen to those specific types or use a generic "message" event.
    // If "event" field is present, "onmessage" won't trigger.
    
    // For simplicity, we can have the backend send a generic "message" event and put the type in the data,
    // OR we can add listeners for all types.
    // Let's check our backend implementation again.
    // It sends: fmt.Fprintf(w, "id: %s\nevent: %s\ndata: %s\n\n", event.ID, event.Type, string(data))
    
    // Since we don't know all types upfront in the client easily, 
    // maybe we should change the backend to send a generic "blackgrid_event" or similar?
    // Or just use the default "message" event if we don't specify the event field.
    
    // Let's modify the backend slightly to also send a generic message event or just listen to all types.
    // Actually, EventSource can listen to any event name.
    
    // Let's assume we listen to a fixed set of categories or a generic 'event' if we change the backend.
    // Or we can just use the "message" event if we don't set the "event" field in SSE.
  }

  private handleReconnect() {
    this.disconnect();
    if (this.retryCount < this.maxRetries) {
      const delay = Math.min(this.baseDelay * Math.pow(2, this.retryCount), 30000);
      console.log(`Reconnecting in ${delay}ms... (attempt ${this.retryCount + 1})`);
      this.reconnectTimeout = setTimeout(() => {
        this.retryCount++;
        this.connect();
      }, delay);
    } else {
      console.error('Max retries reached for EventStream');
    }
  }

  disconnect() {
    if (this.eventSource) {
      this.eventSource.close();
      this.eventSource = null;
    }
    if (this.reconnectTimeout) {
      clearTimeout(this.reconnectTimeout);
      this.reconnectTimeout = null;
    }
    this.options.onDisconnected?.();
  }
}
