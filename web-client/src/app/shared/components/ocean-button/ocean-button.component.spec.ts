import { TestBed } from '@angular/core/testing';
import { By } from '@angular/platform-browser';
import { OceanButtonComponent } from './ocean-button.component';

describe('OceanButtonComponent', () => {
  beforeEach(() => TestBed.configureTestingModule({}));

  it('applies variant class and is disabled stateful', () => {
    const fixture = TestBed.createComponent(OceanButtonComponent);
    const cmp = fixture.componentInstance;
    cmp.variant = 'primary';
    cmp.disabled = true;
    fixture.detectChanges();

    const btn = fixture.debugElement.query(By.css('button')).nativeElement as HTMLButtonElement;
    expect(btn.classList.contains('primary')).toBeTrue();
    expect(btn.disabled).toBeTrue();
  });

  it('emits clicked when not disabled', () => {
    const fixture = TestBed.createComponent(OceanButtonComponent);
    const cmp = fixture.componentInstance;
    fixture.detectChanges();
    let fired = false;
    cmp.clicked.subscribe(() => (fired = true));
    fixture.debugElement.query(By.css('button')).nativeElement.click();
    expect(fired).toBeTrue();
  });
});
