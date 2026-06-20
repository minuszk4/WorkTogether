import { Component, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { Subscription } from 'rxjs';
import { RoomUiStateService, StageMode } from '../../room-ui-state.service';
import { PlayerEngineService } from '../player-engine/player-engine.service';
import { ArtworkVisualizerComponent } from '../artwork-visualizer/artwork-visualizer.component';
import { NowPlayingInfoComponent } from '../now-playing-info/now-playing-info.component';

@Component({
  selector: 'app-room-stage',
  standalone: true,
  imports: [CommonModule, ArtworkVisualizerComponent, NowPlayingInfoComponent],
  template: `
    <div class="stage">
      @switch (stageMode) {
        @case ('screenshare') {
          <div class="stage-media"><ng-content select="[stage-media]"></ng-content></div>
        }
        @case ('video') {
          <div class="stage-media"><ng-content select="[stage-media]"></ng-content></div>
        }
        @default {
          <div class="stage-music">
            <app-room-artwork-visualizer [track]="engine.currentTrack" [isPlaying]="isPlaying"></app-room-artwork-visualizer>
            <div class="stage-meta">
              <app-room-now-playing [track]="engine.currentTrack"></app-room-now-playing>
            </div>
          </div>
        }
      }
    </div>
  `,
  styleUrl: './stage.component.css'
})
export class StageComponent {
  public engine = inject(PlayerEngineService);
  private uiState = inject(RoomUiStateService);

  public stageMode: StageMode = 'music-only';
  public isPlaying = false;
  private subs: Subscription[] = [];

  constructor() {
    this.subs.push(
      this.uiState.changes$.subscribe(s => (this.stageMode = s.stageMode)),
      this.engine.currentState$.subscribe(s => (this.isPlaying = s === 'playing'))
    );
  }
}
