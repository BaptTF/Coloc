// ===== STATE MANAGEMENT =====
const state = {
  vlcChallenge: null,
  vlcAuthenticated: false,
  isLoading: false,
  websocket: null,
  wsConnected: false,
  downloads: new Map(), // downloadId -> download info
  reconnectAttempts: 0,
  maxReconnectAttempts: 5,
  // VLC playback state
  vlcState: {
    status: null,        // Full VLC status object
    queue: null,         // VLC play queue
    volume: null,        // VLC volume
    lastUpdate: null,    // Last state update timestamp
    hasState: false,     // Whether we have valid state
    wsConnected: false   // WebSocket connection status
  }
};

export { state };
