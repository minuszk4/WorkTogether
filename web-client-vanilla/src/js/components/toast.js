export const Toast = {
  container: null,

  init() {
    this.container = document.getElementById('toast-container');
    if (!this.container) {
      this.container = document.createElement('div');
      this.container.id = 'toast-container';
      document.body.appendChild(this.container);
    }
  },

  show(message, type = 'info') {
    if (!this.container) this.init();

    const toast = document.createElement('div');
    toast.className = `toast toast-${type}`;
    
    // Icon mappings using simple SVG circles/marks
    let icon = '';
    if (type === 'success') icon = '✓';
    else if (type === 'error') icon = '✗';
    else icon = 'i';

    toast.innerHTML = `
      <span style="font-weight: 600; background-color: var(--border-color); width: 18px; height: 18px; display: inline-flex; align-items: center; justify-content: center; border-radius: var(--radius-full); font-size: 11px;">${icon}</span>
      <span>${message}</span>
    `;

    this.container.appendChild(toast);

    // Auto-remove after 4 seconds
    setTimeout(() => {
      toast.style.opacity = '0';
      toast.style.transform = 'translateY(12px)';
      setTimeout(() => {
        toast.remove();
      }, 300);
    }, 4000);
  },

  success(message) { this.show(message, 'success'); },
  error(message) { this.show(message, 'error'); },
  info(message) { this.show(message, 'info'); }
};
