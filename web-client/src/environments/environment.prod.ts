export const environment = {
  production: true,
  apiUrl: '/api/v1',
  wsUrl: (window.location.protocol === 'https:' ? 'wss://' : 'ws://') + window.location.host,
  livekitUrl: (window.location.protocol === 'https:' ? 'wss://' : 'ws://') + window.location.host + '/voice'
};
