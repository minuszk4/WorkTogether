import { TestBed } from '@angular/core/testing';
import { By } from '@angular/platform-browser';
import { SeekbarComponent } from './seekbar.component';

describe('SeekbarComponent', () => {
  beforeEach(() => TestBed.configureTestingModule({}));

  it('renders progress fill width from pct input', () => {
    const fixture = TestBed.createComponent(SeekbarComponent);
    const cmp = fixture.componentInstance;
    cmp.pct = 40;
    fixture.detectChanges();
    const fill = fixture.debugElement.query(By.css('.seek-fill')).nativeElement as HTMLElement;
    expect(fill.style.width).toBe('40%');
  });

  it('emits seek with fraction on click', () => {
    const fixture = TestBed.createComponent(SeekbarComponent);
    const cmp = fixture.componentInstance;
    cmp.pct = 0;
    fixture.detectChanges();
    let got = -1;
    cmp.seek.subscribe(f => (got = f));

    const bar = fixture.debugElement.query(By.css('.seek-bar')).nativeElement as HTMLElement;
    spyOn(bar, 'getBoundingClientRect').and.returnValue({ left: 0, width: 200, right: 200, top: 0, bottom: 0, height: 0, x: 0, y: 0, toJSON: () => ({}) } as DOMRect);
    bar.dispatchEvent(new MouseEvent('click', { clientX: 100 }));

    expect(got).toBeCloseTo(0.5, 1);
  });
});
