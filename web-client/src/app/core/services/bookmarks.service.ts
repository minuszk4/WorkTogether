import { Injectable, inject } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';

@Injectable({
  providedIn: 'root'
})
export class BookmarksService {
  private http = inject(HttpClient);

  public getBookmarks(roomId: string): Observable<any> {
    return this.http.get<any>(`/api/v1/music/rooms/${roomId}/bookmarks`);
  }

  public saveBookmark(roomId: string, trackId: string, positionMs: number, note: string): Observable<any> {
    return this.http.post<any>(`/api/v1/music/rooms/${roomId}/bookmarks`, {
      track_id: trackId,
      position_ms: positionMs,
      note: note
    });
  }

  public deleteBookmark(roomId: string, id: string): Observable<any> {
    return this.http.delete<any>(`/api/v1/music/rooms/${roomId}/bookmarks/${id}`);
  }
}
