class GlobalState {
  constructor() {
    this.state = {
      user: null,                   // User profile data
      accessToken: "",              // In-memory JWT access token
      activeRoom: null,             // Currently joined room
      roomMemberRole: "MEMBER",     // Current member role in room
      roomPermissions: [],          // Permissions flag list
      friends: [],                  // Friends list
      roomList: [],                 // Public room list
      voiceConnected: false,        // Voice channel connection state
      voiceParticipants: new Map(), // Active voice participants list
    };

    this.listeners = new Map();
  }

  get(key) {
    return this.state[key];
  }

  set(key, value) {
    const oldValue = this.state[key];
    this.state[key] = value;
    
    // Trigger listeners if value changed
    if (oldValue !== value && this.listeners.has(key)) {
      this.listeners.get(key).forEach(callback => callback(value, oldValue));
    }
  }

  subscribe(key, callback) {
    if (!this.listeners.has(key)) {
      this.listeners.set(key, []);
    }
    this.listeners.get(key).push(callback);
    
    // Return unsubscribe function
    return () => {
      const list = this.listeners.get(key);
      const index = list.indexOf(callback);
      if (index > -1) {
        list.splice(index, 1);
      }
    };
  }

  clear() {
    this.state.user = null;
    this.state.accessToken = "";
    this.state.activeRoom = null;
    this.state.roomMemberRole = "MEMBER";
    this.state.roomPermissions = [];
    this.state.friends = [];
    this.state.voiceConnected = false;
    this.state.voiceParticipants.clear();
  }
}

export const State = new GlobalState();
