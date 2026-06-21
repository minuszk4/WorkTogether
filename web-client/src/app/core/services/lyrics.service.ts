import { Injectable, inject } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';

@Injectable({
  providedIn: 'root'
})
export class LyricsService {
  private http = inject(HttpClient);

  public getLyrics(trackId: string): Observable<any> {
    return this.http.get<any>(`/api/v1/music/tracks/${trackId}/lyrics`);
  }

  public saveLyrics(trackId: string, content: string): Observable<any> {
    return this.http.post<any>(`/api/v1/music/tracks/${trackId}/lyrics`, { content });
  }
}
