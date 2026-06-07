import { State } from './state.js';
import { Toast } from './components/toast.js';

const API_BASE = 'http://localhost:8080/api/v1';

async function request(method, path, body = null, isMultipart = false) {
  const url = `${API_BASE}${path}`;
  const headers = {};

  const token = State.get('accessToken');
  if (token) {
    headers['Authorization'] = `Bearer ${token}`;
  }

  const options = {
    method,
    headers,
  };

  // Only set application/json if it's not a multipart file upload
  if (body) {
    if (isMultipart) {
      options.body = body; // Body is a FormData object
    } else {
      headers['Content-Type'] = 'application/json';
      options.body = JSON.stringify(body);
    }
  }

  options.credentials = 'include'; // Bắt buộc để gửi httpOnly Refresh cookie

  let response = await fetch(url, options);

  // Auto Refresh Token on 401 Unauthorized
  if (response.status === 401 && !path.includes('/auth/login') && !path.includes('/auth/register') && !path.includes('/auth/refresh')) {
    logDebug('Token hết hạn. Đang thử làm mới token (Refresh)...');
    
    const refreshed = await rotateToken();
    if (refreshed) {
      // Retry original request with new token
      headers['Authorization'] = `Bearer ${State.get('accessToken')}`;
      response = await fetch(url, options);
    } else {
      // Clear state and force user back to auth view
      State.clear();
      window.location.hash = '#/auth';
      Toast.error('Phiên đăng nhập đã hết hạn, vui lòng đăng nhập lại.');
      return null;
    }
  }

  const data = await response.json();
  if (!response.ok || !data.success) {
    throw new Error(data.error?.message || 'Lỗi kết nối máy chủ.');
  }

  return data.data;
}

async function rotateToken() {
  try {
    const url = `${API_BASE}/auth/refresh`;
    const response = await fetch(url, {
      method: 'POST',
      credentials: 'include',
    });
    
    const data = await response.json();
    if (response.ok && data.success && data.data.access_token) {
      State.set('accessToken', data.data.access_token);
      logDebug('Đã làm mới JWT Access Token thành công.');
      return true;
    }
  } catch (err) {
    logDebug('Lỗi làm mới token: ', err);
  }
  return false;
}

function logDebug(...args) {
  console.log('[API Client]', ...args);
}

export const API = {
  // Auth API
  auth: {
    register: (username, email, password) => request('POST', '/auth/register', { username, email, password }),
    login: (identity, password) => request('POST', '/auth/login', { identity, password }),
    logout: () => request('POST', '/auth/logout', {}),
  },

  // User & Profiles API
  user: {
    getProfile: (userId) => request('GET', `/users/${userId}/profile`),
    updateProfile: (displayName, bio, avatarUrl = '') => request('PUT', '/users/profile', { display_name: displayName, bio, avatar_url: avatarUrl }),
    updateStatus: (status, customText = '') => request('PUT', '/users/status', { status, custom_text: customText }),
    
    // Friends
    getFriends: (status = 'ACCEPTED') => request('GET', `/users/friends?status=${status}`),
    getPendingFriends: () => request('GET', '/users/friends?status=PENDING'),
    sendFriendRequest: (friendId) => request('POST', '/users/friends/request', { friend_id: friendId }),
    respondFriendRequest: (friendshipId, action) => request('PUT', `/users/friends/request/${friendshipId}`, { action }),
    blockUser: (targetId) => request('POST', '/users/block', { target_id: targetId }),
  },

  // Room API
  room: {
    create: (name, description, privacy, password = '') => request('POST', '/rooms', { name, description, privacy, password }),
    list: (keyword = '', limit = 20) => request('GET', `/rooms?search=${keyword}&limit=${limit}`),
    get: (roomId) => request('GET', `/rooms/${roomId}`),
    join: (roomId, password = '') => request('POST', `/rooms/${roomId}/join`, { password }),
    leave: (roomId) => request('POST', `/rooms/${roomId}/leave`, {}),
    
    // Permissions & Roles
    createRole: (roomId, name, perms) => request('POST', `/rooms/${roomId}/roles`, { name, permissions: perms }),
    assignRole: (roomId, userId, roleId) => request('PUT', `/rooms/${roomId}/members/${userId}/role`, { role_id: roleId }),
    
    // Moderation
    kick: (roomId, userId) => request('POST', `/rooms/${roomId}/members/${userId}/kick`, {}),
    ban: (roomId, userId, reason = '') => request('POST', `/rooms/${roomId}/members/${userId}/ban`, { reason }),
    mute: (roomId, userId, durationSec = 600) => request('POST', `/rooms/${roomId}/members/${userId}/mute`, { duration_seconds: durationSec }),
  },

  // Music API
  music: {
    extract: (sourceUrl) => request('POST', '/music/extract', { source_url: sourceUrl }),
    upload: (formData) => request('POST', '/music/upload', formData, true),
    search: (keyword) => request('GET', `/music/search?keyword=${keyword}`),
    getHistory: (roomId) => request('GET', `/music/history/${roomId}`),
  },

  // Playlist & Queue API
  playlist: {
    create: (name, roomId = null) => request('POST', '/playlists/', { name, room_id: roomId }),
    getRoomPlaylists: (roomId) => request('GET', `/playlists/room/${roomId}`),
    getUserPlaylists: () => request('GET', '/playlists/user'),
    delete: (playlistId) => request('DELETE', `/playlists/${playlistId}`),
    
    addTrack: (playlistId, track) => request('POST', `/playlists/${playlistId}/tracks`, {
      track_id: track.id,
      title: track.title,
      artist: track.artist,
      thumbnail_url: track.thumbnail_url || '',
      duration_ms: track.duration_ms,
      source_url: track.source_url,
    }),
    getTracks: (playlistId) => request('GET', `/playlists/${playlistId}/tracks`),
    removeTrack: (playlistId, itemID) => request('DELETE', `/playlists/${playlistId}/tracks/${itemID}`),
    moveTrack: (playlistId, itemID, newPos) => request('PUT', `/playlists/${playlistId}/tracks/${itemID}/move`, { new_position: newPos }),
    voteTrack: (itemID, voteType) => request('POST', `/playlists/tracks/${itemID}/vote`, { vote_type: voteType }),
  },

  // Voice API
  voice: {
    getToken: (roomId) => request('POST', '/voice/token', { room_id: roomId }),
  },

  // Notification API
  notification: {
    list: () => request('GET', '/notifications/'),
    unreadCount: () => request('GET', '/notifications/unread-count'),
    markRead: (id) => request('PUT', `/notifications/${id}/read`, {}),
    markAllRead: () => request('PUT', '/notifications/read-all', {}),
  }
};
