import { TestBed } from '@angular/core/testing';
import { By } from '@angular/platform-browser';
import { IconButtonComponent } from './icon-button.component';

describe('IconButtonComponent', () => {
  beforeEach(() => TestBed.configureTestingModule({}));

  it('renders an icon-only button with title and aria-label', () => {
    const fixture = TestBed.createComponent(IconButtonComponent);
    const cmp = fixture.componentInstance;
    cmp.label = 'Play music';
    cmp.disabled = false;
    fixture.detectChanges();

    const btn = fixture.debugElement.query(By.css('button')).nativeElement as HTMLButtonElement;
    expect(btn.getAttribute('aria-label')).toBe('Play music');
    expect(btn.title).toBe('Play music');
    expect(btn.disabled).toBeFalse();
  });

  it('emits clicked when pressed and not disabled', () => {
    const fixture = TestBed.createComponent(IconButtonComponent);
    const cmp = fixture.componentInstance;
    cmp.label = 'Mute';
    cmp.disabled = false;
    fixture.detectChanges();

    let fired = false;
    cmp.clicked.subscribe(() => (fired = true));

    const btn = fixture.debugElement.query(By.css('button')).nativeElement as HTMLButtonElement;
    btn.click();
    expect(fired).toBeTrue();
  });

  it('does not emit when disabled', () => {
    const fixture = TestBed.createComponent(IconButtonComponent);
    const cmp = fixture.componentInstance;
    cmp.label = 'Mute';
    cmp.disabled = true;
    fixture.detectChanges();

    let fired = false;
    cmp.clicked.subscribe(() => (fired = true));

    const btn = fixture.debugElement.query(By.css('button')).nativeElement as HTMLButtonElement;
    btn.click();
    expect(fired).toBeFalse();
  });
});
