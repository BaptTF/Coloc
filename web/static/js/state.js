// ===== STATE MANAGEMENT =====
const state = {
  vlcChallenge: null,
  vlcAuthenticated: false,
  isLoading: false,
  websocket: null,
  wsConnected: false,
  downloads: new Map(), // downloadId -> download info
  reconnectAttempts: 0,
  maxReconnectAttempts: 5
};

export { state };
