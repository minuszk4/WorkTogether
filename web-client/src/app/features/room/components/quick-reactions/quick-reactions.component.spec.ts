import { TestBed, ComponentFixture } from '@angular/core/testing';
import { QuickReactionsComponent } from './quick-reactions.component';
import { ChatWsService } from '../../../../core/services/websocket/chat-ws.service';

describe('QuickReactionsComponent', () => {
  let component: QuickReactionsComponent;
  let fixture: ComponentFixture<QuickReactionsComponent>;
  let mockChatWsService: jasmine.SpyObj<ChatWsService>;

  beforeEach(async () => {
    mockChatWsService = jasmine.createSpyObj('ChatWsService', ['sendLiveReaction']);

    await TestBed.configureTestingModule({
      imports: [QuickReactionsComponent],
      providers: [
        { provide: ChatWsService, useValue: mockChatWsService }
      ]
    }).compileComponents();

    fixture = TestBed.createComponent(QuickReactionsComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should compile and render all emoji buttons', () => {
    const element: HTMLElement = fixture.nativeElement;
    const buttons = element.querySelectorAll('button');
    expect(buttons.length).toBe(5);
    
    const emojis = Array.from(buttons).map(b => b.textContent?.trim());
    expect(emojis).toEqual(['❤️', '🔥', '👏', '😮', '📚']);
  });

  it('should call chatWs.sendLiveReaction when an emoji is clicked', () => {
    const element: HTMLElement = fixture.nativeElement;
    const buttons = element.querySelectorAll('button');
    
    const fireBtn = Array.from(buttons).find(b => b.textContent?.trim() === '🔥');
    expect(fireBtn).toBeDefined();
    fireBtn?.dispatchEvent(new Event('click'));
    
    expect(mockChatWsService.sendLiveReaction).toHaveBeenCalledWith('🔥');
  });
});
