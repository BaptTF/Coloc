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
    vlcConfig: '/vlc/config',
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
    downloadsList: '#downloadsList'
  },
  classes: {
    loading: 'loading',
    disabled: 'disabled',
    active: 'active'
  }
};

export { CONFIG };
