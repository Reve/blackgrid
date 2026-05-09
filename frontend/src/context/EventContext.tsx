import React, { createContext, useContext, useEffect, useState, useCallback, useRef } from 'react';
import { EventStreamClient } from '../lib/events/client';
import { Event, EventType } from '../lib/events/types';
import { useAuth } from './AuthContext';

interface EventContextType {
  isConnected: boolean;
  lastEvents: Event[];
  subscribe: (callback: (event: Event) => void, types?: EventType[]) => () => void;
}

const EventContext = createContext<EventContextType | undefined>(undefined);

export const EventProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const { user } = useAuth();
  const [isConnected, setIsConnected] = useState(false);
  const [lastEvents, setLastEvents] = useState<Event[]>([]);
  const subscribers = useRef<Array<{ callback: (event: Event) => void; types?: EventType[] }>>([]);
  const clientRef = useRef<EventStreamClient | null>(null);

  const handleEvent = useCallback((event: Event) => {
    // Add to history (keep last 50)
    setLastEvents((prev) => [event, ...prev].slice(0, 50));

    // Notify subscribers
    subscribers.current.forEach(({ callback, types }) => {
      if (!types || types.includes(event.type)) {
        callback(event);
      }
    });
  }, []);

  const subscribe = useCallback((callback: (event: Event) => void, types?: EventType[]) => {
    const sub = { callback, types };
    subscribers.current.push(sub);
    return () => {
      subscribers.current = subscribers.current.filter((s) => s !== sub);
    };
  }, []);

  useEffect(() => {
    if (!user) {
      if (clientRef.current) {
        clientRef.current.disconnect();
        clientRef.current = null;
        setIsConnected(false);
      }
      return;
    }

    if (!clientRef.current) {
      clientRef.current = new EventStreamClient({
        onEvent: handleEvent,
        onConnected: () => setIsConnected(true),
        onDisconnected: () => setIsConnected(false),
        onError: () => setIsConnected(false),
      });
      clientRef.current.connect();
    }

    return () => {
      if (clientRef.current) {
        clientRef.current.disconnect();
        clientRef.current = null;
      }
    };
  }, [user, handleEvent]);

  return (
    <EventContext.Provider value={{ isConnected, lastEvents, subscribe }}>
      {children}
    </EventContext.Provider>
  );
};

export const useEvents = () => {
  const context = useContext(EventContext);
  if (context === undefined) {
    throw new Error('useEvents must be used within an EventProvider');
  }
  return context;
};
