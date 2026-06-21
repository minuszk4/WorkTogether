import { Component, OnDestroy, OnInit, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { PlaybackWsService } from '../../../../core/services/websocket/playback-ws.service';
import { Subscription } from 'rxjs';

@Component({
  selector: 'app-room-poll-widget',
  standalone: true,
  imports: [CommonModule],
  templateUrl: './poll-widget.component.html',
  styleUrl: './poll-widget.component.css'
})
export class PollWidgetComponent implements OnInit, OnDestroy {
  public playbackWs = inject(PlaybackWsService);

  public candidates: any[] = [];
  public votes: Record<string, number> = {};
  public showWidget = false;
  public votedTrackId: string | null = null;
  public winnerName = '';
  public showWinner = false;

  public progressPct = 100;
  private duration = 30000;
  private startTime = 0;
  private timer: any = null;
  private winnerTimer: any = null;

  private subs: Subscription[] = [];

  ngOnInit(): void {
    this.subs.push(
      this.playbackWs.poll$.subscribe(poll => {
        if (poll) {
          this.candidates = poll.candidates || [];
          this.duration = poll.duration || 30000;
          this.showWidget = true;
          this.showWinner = false;
          this.votedTrackId = null;
          this.startTime = Date.now();
          this.startCountdown();
        } else {
          this.showWidget = false;
          if (this.timer) {
            clearInterval(this.timer);
            this.timer = null;
          }
        }
      }),
      this.playbackWs.pollVotes$.subscribe(votes => {
        this.votes = votes || {};
      }),
      this.playbackWs.pollEnd$.subscribe(end => {
        if (end && end.winner) {
          this.winnerName = end.winner.title || 'Unknown Title';
          this.showWinner = true;
          if (this.winnerTimer) clearTimeout(this.winnerTimer);
          this.winnerTimer = setTimeout(() => {
            this.showWinner = false;
          }, 5000);
        } else {
          this.showWinner = false;
        }
      })
    );
  }

  ngOnDestroy(): void {
    this.subs.forEach(s => s.unsubscribe());
    if (this.timer) clearInterval(this.timer);
    if (this.winnerTimer) clearTimeout(this.winnerTimer);
  }

  private startCountdown(): void {
    if (this.timer) clearInterval(this.timer);
    this.timer = setInterval(() => {
      const elapsed = Date.now() - this.startTime;
      const remaining = this.duration - elapsed;
      if (remaining <= 0) {
        this.progressPct = 0;
        clearInterval(this.timer);
      } else {
        this.progressPct = (remaining / this.duration) * 100;
      }
    }, 100);
  }

  public vote(candidateId: string): void {
    if (this.votedTrackId) return;
    this.votedTrackId = candidateId;
    this.playbackWs.voteForTrack(candidateId);
  }

  public getVoteCount(candidateId: string): number {
    return this.votes[candidateId] || 0;
  }

  public getTotalVotes(): number {
    let sum = 0;
    for (const key in this.votes) {
      sum += this.votes[key];
    }
    return sum;
  }

  public getVotePct(candidateId: string): number {
    const total = this.getTotalVotes();
    if (total === 0) return 0;
    const count = this.getVoteCount(candidateId);
    return Math.round((count / total) * 100);
  }
}
