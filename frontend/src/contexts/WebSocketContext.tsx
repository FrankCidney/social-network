"use client";

import React, { createContext, useContext, useEffect, useState, ReactNode } from "react";
import { profileAPI } from "@/lib/api";

interface WebSocketContextType {
  socket: WebSocket | null;
  isConnected: boolean;
}

const WebSocketContext = createContext<WebSocketContextType | undefined>(undefined);

export function WebSocketProvider({ children }: { children: ReactNode }) {
  const [socket, setSocket] = useState<WebSocket | null>(null);
  const [isConnected, setIsConnected] = useState(false);

  useEffect(() => {
    let ws: WebSocket | null = null;
    let cancelled = false;

    const apiUrl = process.env.NEXT_PUBLIC_API_URL?.replace(/\/$/, "");
    const wsUrl = apiUrl
      ? `${apiUrl.replace(/^http/, "ws")}/api/ws`
      : `ws://${window.location.host}/api/ws`;

    async function connect() {
      try {
        await profileAPI.getMyProfile();
      } catch {
        if (!cancelled) {
          setSocket(null);
          setIsConnected(false);
        }
        return;
      }

      if (cancelled) return;

      ws = new WebSocket(wsUrl);

      ws.onopen = () => {
        setIsConnected(true);
      };

      ws.onclose = (event) => {
        setIsConnected(false);
        setSocket(null);

        if (!cancelled && event.code !== 1000) {
          console.warn("WebSocket closed", {
            code: event.code,
            reason: event.reason || "No reason provided",
          });
        }
      };

      ws.onerror = () => {
        setIsConnected(false);
      };

      setSocket(ws);
    }

    void connect();

    return () => {
      cancelled = true;
      ws?.close();
    };
  }, []);

  return (
    <WebSocketContext.Provider value={{ socket, isConnected }}>
      {children}
    </WebSocketContext.Provider>
  );
}

export function useWebSocket() {
  const context = useContext(WebSocketContext);
  if (context === undefined) {
    throw new Error("useWebSocket must be used within a WebSocketProvider");
  }
  return context;
}
