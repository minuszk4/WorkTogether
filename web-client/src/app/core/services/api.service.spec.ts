import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { TestBed } from '@angular/core/testing';
import { ApiService } from './api.service';

describe('ApiService user presence', () => {
  let api: ApiService;
  let http: HttpTestingController;

  beforeEach(() => {
    TestBed.configureTestingModule({
      providers: [ApiService, provideHttpClient(), provideHttpClientTesting()]
    });
    api = TestBed.inject(ApiService);
    http = TestBed.inject(HttpTestingController);
  });

  afterEach(() => http.verify());

  it('caches profiles separately for each room', () => {
    api.user.getProfile('user-1', 'room-a').subscribe();
    api.user.getProfile('user-1', 'room-b').subscribe();

    http.expectOne(request => request.urlWithParams.endsWith('/users/user-1/profile?room_id=room-a'))
      .flush({ success: true, data: { id: 'user-1' }, error: null });
    http.expectOne(request => request.urlWithParams.endsWith('/users/user-1/profile?room_id=room-b'))
      .flush({ success: true, data: { id: 'user-1' }, error: null });
  });

  it('heartbeats without custom text', () => {
    api.user.heartbeatPresence('away').subscribe();

    const request = http.expectOne(request => request.url.endsWith('/users/status/heartbeat'));
    expect(request.request.method).toBe('PUT');
    expect(request.request.body).toEqual({ status: 'away' });
    request.flush({ success: true, data: {}, error: null });
  });
});
