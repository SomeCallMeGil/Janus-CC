const WS_URL = import.meta.env.VITE_WS_URL || 'ws://localhost:8080/ws';

export class WebSocketClient {
  constructor(endpoint = '') {
    this.endpoint = endpoint;
    this.ws = null;
    this.listeners = [];
    this.reconnectDelay = 2000;
    this._reconnectTimer = null;
    this._closed = false;
  }

  connect() {
    this._closed = false;
    this.ws = new WebSocket(`${WS_URL}${this.endpoint}`);

    this.ws.onopen = () => {
      this.reconnectDelay = 2000;
    };

    this.ws.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data);
        this.listeners.forEach((fn) => fn(data));
      } catch {
        // non-JSON frame — ignore
      }
    };

    this.ws.onclose = () => {
      if (!this._closed) {
        this._reconnectTimer = setTimeout(() => this.connect(), this.reconnectDelay);
        this.reconnectDelay = Math.min(this.reconnectDelay * 2, 30000);
      }
    };

    this.ws.onerror = () => {
      this.ws?.close();
    };
  }

  onMessage(callback) {
    this.listeners.push(callback);
    return () => {
      this.listeners = this.listeners.filter((fn) => fn !== callback);
    };
  }

  disconnect() {
    this._closed = true;
    clearTimeout(this._reconnectTimer);
    this.ws?.close();
  }

  get readyState() {
    return this.ws?.readyState ?? WebSocket.CLOSED;
  }
}

/** Singleton WebSocket instance shared across the app. */
export const wsClient = new WebSocketClient();
