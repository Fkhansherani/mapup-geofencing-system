import { useEffect, useRef, useState } from "react";
import { WS_URL } from "../config";

// Connects to the backend WebSocket and calls onAlert(alert) for every message.
// If the connection drops it reconnects automatically, waiting a bit longer each time.
// Returns the connection status: "connecting" | "connected" | "disconnected".
export function useAlertSocket(onAlert) {
  const [status, setStatus] = useState("connecting");

  // Keep the newest onAlert in a ref, so the socket does not reconnect
  // every time the parent component re-renders.
  const onAlertRef = useRef(onAlert);
  useEffect(() => {
    onAlertRef.current = onAlert;
  }, [onAlert]);

  useEffect(() => {
    let socket;
    let retryTimer;
    let retries = 0;
    let stopped = false; // true after the component unmounts

    function connect() {
      setStatus("connecting");
      socket = new WebSocket(WS_URL);

      socket.onopen = () => {
        retries = 0;
        setStatus("connected");
      };

      socket.onmessage = (event) => {
        try {
          onAlertRef.current(JSON.parse(event.data));
        } catch (error) {
          console.error("Bad alert message", error);
        }
      };

      socket.onclose = () => {
        if (stopped) return;
        setStatus("disconnected");
        // 1s, 2s, 4s, 8s ... but never more than 15s
        const delay = Math.min(1000 * 2 ** retries, 15000);
        retries += 1;
        retryTimer = setTimeout(connect, delay);
      };
    }

    connect();

    // Cleanup: stop reconnecting and close the socket.
    return () => {
      stopped = true;
      clearTimeout(retryTimer);
      if (socket) socket.close();
    };
  }, []);

  return status;
}
