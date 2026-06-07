export const Modal = {
  activeOverlay: null,

  open(overlayId) {
    const overlay = document.getElementById(overlayId);
    if (!overlay) return;

    // Close current if open
    if (this.activeOverlay) {
      this.close(this.activeOverlay.id);
    }

    overlay.classList.add('active');
    this.activeOverlay = overlay;

    // Esc to close
    const handleEsc = (e) => {
      if (e.key === 'Escape') {
        this.close(overlayId);
        document.removeEventListener('keydown', handleEsc);
      }
    };
    document.addEventListener('keydown', handleEsc);

    // Click outside to close
    overlay.onclick = (e) => {
      if (e.target === overlay) {
        this.close(overlayId);
      }
    };

    // Close button inside
    const closeBtn = overlay.querySelector('.modal-close');
    if (closeBtn) {
      closeBtn.onclick = () => this.close(overlayId);
    }
  },

  close(overlayId) {
    const overlay = document.getElementById(overlayId);
    if (!overlay) return;

    overlay.classList.remove('active');
    if (this.activeOverlay === overlay) {
      this.activeOverlay = null;
    }
  }
};
