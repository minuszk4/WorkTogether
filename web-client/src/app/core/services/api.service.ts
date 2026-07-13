import { Injectable } from '@angular/core';
import { HttpClient, HttpHeaders } from '@angular/common/http';
import { environment } from '../../../environments/environment';
import { Observable } from 'rxjs';
import { map, shareReplay } from 'rxjs/operators';

export interface ApiResponse<T = any> {
  success: boolean;
  data: T;
  error: {
    code: string;
    message: string;
  } | null;
}

@Injectable({
  providedIn: 'root'
})
export class ApiService {
  private apiBase = environment.apiUrl;
  private profileCache = new Map<string, Observable<any>>();

  private jsonHeaders = new HttpHeaders({ 'Content-Type': 'application/json' });
  private httpOptions = { headers: this.jsonHeaders, withCredentials: true };
  private httpOptionsNoBody = { withCredentials: true };

  constructor(private http: HttpClient) {}

  // ─── Core HTTP helpers ────────────────────────────────────────────
  private get<T>(path: string): Observable<T> {
    return this.http
      .get<ApiResponse<T>>(`${this.apiBase}${path}`, this.httpOptionsNoBody)
      .pipe(map(this.unwrap));
  }

  private post<T>(path: string, body: any): Observable<T> {
    return this.http
      .post<ApiResponse<T>>(`${this.apiBase}${path}`, body, this.httpOptions)
      .pipe(map(this.unwrap));
  }

  private postForm<T>(path: string, formData: FormData): Observable<T> {
    // FormData: browser tự set Content-Type + boundary
    return this.http
      .post<ApiResponse<T>>(`${this.apiBase}${path}`, formData, { withCredentials: true })
      .pipe(map(this.unwrap));
  }

  private put<T>(path: string, body: any): Observable<T> {
    return this.http
      .put<ApiResponse<T>>(`${this.apiBase}${path}`, body, this.httpOptions)
      .pipe(map(this.unwrap));
  }

  private delete<T>(path: string): Observable<T> {
    return this.http
      .delete<ApiResponse<T>>(`${this.apiBase}${path}`, this.httpOptionsNoBody)
      .pipe(map(this.unwrap));
  }

  private unwrap<T>(res: ApiResponse<T>): T {
    if (!res.success) {
      throw new Error(res.error?.message || 'Lỗi kết nối máy chủ.');
    }
    return res.data;
  }

  // ─── Auth APIs ────────────────────────────────────────────────────
  public auth = {
    register: (username: string, email: string, password: string): Observable<any> =>
      this.post<any>('/auth/register', { username, email, password }),

    login: (identity: string, password: string): Observable<any> =>
      this.post<any>('/auth/login', { identity, password }),

    logout: (): Observable<any> =>
      this.post<any>('/auth/logout', {}),
  };

  // ─── User & Profile APIs ──────────────────────────────────────────
  public user = {
    getProfile: (userId: string): Observable<any> => {
      if (!this.profileCache.has(userId)) {
        const obs = this.get<any>(`/users/${userId}/profile`).pipe(shareReplay(1));
        this.profileCache.set(userId, obs);
      }
      return this.profileCache.get(userId)!;
    },

    clearProfileCache: (userId: string) => {
      this.profileCache.delete(userId);
    },

    updateProfile: (displayName: string, bio: string, avatarUrl = ''): Observable<any> =>
      this.put<any>('/users/profile', { display_name: displayName, bio, avatar_url: avatarUrl }),

    updateStatus: (status: string, customText = ''): Observable<any> =>
      this.put<any>('/users/status', { status, custom_text: customText }),

    getFriends: (status = 'ACCEPTED'): Observable<any[]> =>
      this.get<any[]>(`/users/friends?status=${status}`),

    getPendingFriends: (): Observable<any[]> =>
      this.get<any[]>('/users/friends?status=PENDING'),

    sendFriendRequest: (friendId: string): Observable<any> =>
      this.post<any>('/users/friends/request', { friend_id: friendId }),

    respondFriendRequest: (friendshipId: string, action: string): Observable<any> =>
      this.put<any>(`/users/friends/request/${friendshipId}`, { action }),

    blockUser: (targetId: string): Observable<any> =>
      this.post<any>('/users/block', { target_id: targetId }),
  };

  // ─── Room APIs ────────────────────────────────────────────────────
  public room = {
    create: (name: string, description: string, privacy: string, password = '', mode = 'chill'): Observable<any> =>
      this.post<any>('/rooms', { name, description, privacy, password, mode }),

    list: (keyword = '', limit = 20): Observable<any[]> =>
      this.get<any[]>(`/rooms?search=${keyword}&limit=${limit}`),

    getByInviteCode: (inviteCode: string): Observable<any> =>
      this.get<any>(`/rooms/invite/${encodeURIComponent(inviteCode)}`),

    get: (roomId: string): Observable<any> =>
      this.get<any>(`/rooms/${roomId}`),

    listMembers: (roomId: string): Observable<any[]> =>
      this.get<any[]>(`/rooms/${roomId}/members`),

    join: (roomId: string, password = ''): Observable<any> =>
      this.post<any>(`/rooms/${roomId}/join`, { password }),

    leave: (roomId: string): Observable<any> =>
      this.post<any>(`/rooms/${roomId}/leave`, {}),

    createRole: (roomId: string, name: string, perms: string[]): Observable<any> =>
      this.post<any>(`/rooms/${roomId}/roles`, { name, permissions: perms }),

    assignRole: (roomId: string, userId: string, roleId: string): Observable<any> =>
      this.put<any>(`/rooms/${roomId}/members/${userId}/role`, { role_id: roleId }),

    kick: (roomId: string, userId: string): Observable<any> =>
      this.post<any>(`/rooms/${roomId}/members/${userId}/kick`, {}),

    ban: (roomId: string, userId: string, reason = ''): Observable<any> =>
      this.post<any>(`/rooms/${roomId}/members/${userId}/ban`, { reason }),

    mute: (roomId: string, userId: string, durationSec = 600): Observable<any> =>
      this.post<any>(`/rooms/${roomId}/members/${userId}/mute`, { duration_seconds: durationSec }),

    updateSettings: (roomId: string, name: string, description: string, addMusicPolicy: string, avatarUrl?: string | null, rules?: string | null, theme?: string): Observable<any> =>
      this.put<any>(`/rooms/${roomId}/settings`, { name, description, add_music_policy: addMusicPolicy, avatar_url: avatarUrl, rules, theme }),

    updateMode: (roomId: string, mode: 'chill' | 'focus' | 'collaborate'): Observable<{ mode: string }> =>
      this.put<{ mode: string }>(`/rooms/${roomId}/mode`, { mode }),

    listSessionTemplates: (): Observable<any[]> =>
      this.get<any[]>('/rooms/session-templates'),

    listMyActionItems: (): Observable<any[]> =>
      this.get<any[]>('/rooms/me/actions'),

    getActiveSession: (roomId: string): Observable<any | null> =>
      this.get<any | null>(`/rooms/${roomId}/session`),

    startSession: (roomId: string, title: string, goal: string, templateKey: string): Observable<any> =>
      this.post<any>(`/rooms/${roomId}/sessions`, { title, goal, template_key: templateKey }),

    getSessionWorkspace: (roomId: string, sessionId: string): Observable<any> =>
      this.get<any>(`/rooms/${roomId}/sessions/${sessionId}`),

    completeSession: (roomId: string, sessionId: string): Observable<any> =>
      this.put<any>(`/rooms/${roomId}/sessions/${sessionId}/complete`, {}),

    addAgendaItem: (roomId: string, sessionId: string, content: string, position: number): Observable<any> =>
      this.post<any>(`/rooms/${roomId}/sessions/${sessionId}/agenda`, { content, position }),

    updateAgendaItem: (roomId: string, sessionId: string, itemId: string, isDone: boolean): Observable<any> =>
      this.put<any>(`/rooms/${roomId}/sessions/${sessionId}/agenda/${itemId}`, { is_done: isDone }),

    addActionItem: (roomId: string, sessionId: string, content: string, assigneeId?: string): Observable<any> =>
      this.post<any>(`/rooms/${roomId}/sessions/${sessionId}/actions`, { content, assignee_id: assigneeId || null }),

    updateActionItem: (roomId: string, sessionId: string, actionId: string, status: 'OPEN' | 'DONE'): Observable<any> =>
      this.put<any>(`/rooms/${roomId}/sessions/${sessionId}/actions/${actionId}`, { status }),
  };

  // ─── Music APIs ───────────────────────────────────────────────────
  public music = {
    extract: (sourceUrl: string): Observable<any> =>
      this.post<any>('/music/extract', { source_url: sourceUrl }),

    upload: (formData: FormData): Observable<any> =>
      this.postForm<any>('/music/upload', formData),

    search: (keyword: string): Observable<any[]> =>
      this.get<any[]>(`/music/search?keyword=${keyword}`),

    getHistory: (roomId: string): Observable<any[]> =>
      this.get<any[]>(`/music/history/${roomId}`),

    getStats: (roomId: string): Observable<any> =>
      this.get<any>(`/music/rooms/${roomId}/stats`),
  };

  // ─── Playlist & Queue APIs ────────────────────────────────────────
  public playlist = {
    create: (name: string, roomId: string | null = null): Observable<any> =>
      this.post<any>('/playlists/', { name, room_id: roomId }),

    getRoomPlaylists: (roomId: string): Observable<any[]> =>
      this.get<any[]>(`/playlists/room/${roomId}`),

    getUserPlaylists: (): Observable<any[]> =>
      this.get<any[]>('/playlists/user'),

    delete: (playlistId: string): Observable<any> =>
      this.delete<any>(`/playlists/${playlistId}`),

    addTrack: (playlistId: string, track: any): Observable<any> =>
      this.post<any>(`/playlists/${playlistId}/tracks`, {
        track_id: track.id || track.track_id,
        title: track.title,
        artist: track.artist || 'Unknown',
        thumbnail_url: track.thumbnail_url || '',
        duration_ms: track.duration_ms,
        source_url: track.source_url,
      }),

    getTracks: (playlistId: string): Observable<any[]> =>
      this.get<any[]>(`/playlists/${playlistId}/tracks`),

    removeTrack: (playlistId: string, itemID: string): Observable<any> =>
      this.delete<any>(`/playlists/${playlistId}/tracks/${itemID}`),

    moveTrack: (playlistId: string, itemID: string, newPos: number): Observable<any> =>
      this.put<any>(`/playlists/${playlistId}/tracks/${itemID}/move`, { new_position: newPos }),

    voteTrack: (itemID: string, voteType: string): Observable<any> =>
      this.post<any>(`/playlists/tracks/${itemID}/vote`, { vote_type: voteType }),
  };

  // ─── Playback APIs ────────────────────────────────────────────────
  public playback = {
    assignGuestDj: (roomId: string, userId: string, durationSeconds = 3600): Observable<any> =>
      this.post<any>(`/rooms/${roomId}/playback/dj`, {
        user_id: userId,
        duration_seconds: durationSeconds,
      }),

    revokeGuestDj: (roomId: string): Observable<any> =>
      this.delete<any>(`/rooms/${roomId}/playback/dj`),
  };

  // ─── Voice APIs ───────────────────────────────────────────────────
  public voice = {
    getToken: (roomId: string, channelId: string | null = null): Observable<any> =>
      this.post<any>(`/voice/rooms/${roomId}/token`, {
        channel_id: channelId,
        publish_sources: ['microphone', 'camera', 'screen_share']
      }),
  };

  // ─── Notification APIs ────────────────────────────────────────────
  public notification = {
    list: (): Observable<any[]> =>
      this.get<any[]>('/notifications/'),

    unreadCount: (): Observable<any> =>
      this.get<any>('/notifications/unread-count'),

    markRead: (id: string): Observable<any> =>
      this.put<any>(`/notifications/${id}/read`, {}),

    markAllRead: (): Observable<any> =>
      this.put<any>('/notifications/read-all', {}),
  };
}
