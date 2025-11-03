// ===== CONSTANTS & CONFIG =====
const CONFIG = {
  endpoints: {
    direct: '/url',
    youtube: '/urlyt',
    twitch: '/twitch',
    playurl: '/playurl',
    list: '/list',
    vlcCode: '/vlc/code',
    vlcVerify: '/vlc/verify-code',
    vlcPlay: '/vlc/play',
    vlcStatus: '/vlc/status',
    vlcState: '/vlc/state',
    vlcConfig: '/vlc/config',
    vlcWsConnect: '/vlc/ws/connect',
    vlcWsStatus: '/vlc/ws/status',
    vlcWsControl: '/vlc/ws/control',
    vlcWsDisconnect: '/vlc/ws/disconnect',
    websocket: '/ws',
    queueClear: '/queue/clear'
  },
  selectors: {
    backendUrl: '#backendUrl',
    vlcUrl: '#vlcUrl',
    videoUrl: '#videoUrl',
    autoPlay: '#autoPlay',
    vlcStatus: '#vlcStatus',
    ytdlpStatus: '#ytdlpStatus',
    videosGrid: '#videosGrid',
    vlcModal: '#vlcModal',
    vlcCode: '#vlcCode',
    downloadProgressSection: '#downloadProgressSection',
    downloadsList: '#downloadsList',
    vlcRemoteSection: '#vlcRemoteSection',
    wsStatusDot: '#wsStatusDot',
    wsStatusText: '#wsStatusText',
    volumeSlider: '#volumeSlider',
    volumeValue: '#volumeValue',
    seekSlider: '#seekSlider',
    seekValue: '#seekValue',
    currentTitle: '#currentTitle',
    currentDuration: '#currentDuration',
    playbackState: '#playbackState'
  },
  classes: {
    loading: 'loading',
    disabled: 'disabled',
    active: 'active'
  }
};

export { CONFIG };
