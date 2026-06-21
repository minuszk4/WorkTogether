import { Injectable, inject } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';
import { map } from 'rxjs/operators';

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
export class SubroomService {
  private http = inject(HttpClient);

  public getSubrooms(parentId: string): Observable<any[]> {
    return this.http.get<ApiResponse<any[]>>(`/api/v1/rooms/${parentId}/subrooms`).pipe(
      map(res => {
        if (!res.success) throw new Error(res.error?.message || 'Lỗi lấy danh sách phòng con.');
        return res.data;
      })
    );
  }

  public createSubroom(parentId: string, name: string, description: string): Observable<any> {
    return this.http.post<ApiResponse<any>>(`/api/v1/rooms/${parentId}/subrooms`, { name, description }).pipe(
      map(res => {
        if (!res.success) throw new Error(res.error?.message || 'Lỗi tạo phòng con.');
        return res.data;
      })
    );
  }

  public moveMember(parentId: string, userId: string, subRoomId: string | null): Observable<any> {
    return this.http.put<ApiResponse<any>>(`/api/v1/rooms/${parentId}/members/${userId}/move`, { sub_room_id: subRoomId }).pipe(
      map(res => {
        if (!res.success) throw new Error(res.error?.message || 'Lỗi di chuyển thành viên.');
        return res.data;
      })
    );
  }
}
