/**
 * Cross-tab coordination for token refresh deduplication.
 *
 * Uses BroadcastChannel to elect a single "leader" tab that performs
 * token refreshes on behalf of all tabs. Followers wait for the result.
 *
 * Falls back to independent refresh if BroadcastChannel is not supported.
 */

const CHANNEL_NAME = 'pos-auth-coordination';
const HEARTBEAT_INTERVAL = 5_000;
const LEADER_TIMEOUT = 10_000;
const ELECTION_TIMEOUT = 500;

type Message =
  | { type: 'PING'; tabId: string }
  | { type: 'PONG'; tabId: string }
  | { type: 'HEARTBEAT'; tabId: string }
  | { type: 'REFRESH_REQUEST'; tabId: string }
  | { type: 'REFRESH_RESULT'; token: string }
  | { type: 'REFRESH_FAILED' }
  | { type: 'LOGOUT' };

// --- Tab identity ---
function generateTabId(): string {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return crypto.randomUUID();
  }
  return Math.random().toString(36).slice(2) + Date.now().toString(36);
}

const tabId = generateTabId();

// --- State ---
let channel: BroadcastChannel | null = null;
let isLeader = false;
let leaderTabId: string | null = null;
let heartbeatTimer: ReturnType<typeof setInterval> | null = null;
let leaderCheckTimer: ReturnType<typeof setInterval> | null = null;
let lastHeartbeat = 0;
let electionPending = false;

// --- Callbacks ---
let onLeaderChangeCallbacks: Array<(leader: boolean) => void> = [];
let onLogoutCallbacks: Array<() => void> = [];
let refreshResolvers: {
  resolve: (token: string) => void;
  reject: (err: unknown) => void;
} | null = null;

function notifyLeaderChange() {
  onLeaderChangeCallbacks.forEach((cb) => {
    try { cb(isLeader); } catch { /* ignore callback errors */ }
  });
}

function notifyLogout() {
  onLogoutCallbacks.forEach((cb) => {
    try { cb(); } catch { /* ignore callback errors */ }
  });
}

// --- Message handling ---
function handleMessage(event: MessageEvent<Message>) {
  const msg = event.data;
  if ('tabId' in msg && msg.tabId === tabId) return; // ignore own messages

  switch (msg.type) {
    case 'PING':
      // Someone is electing — respond if we think we're leader
      if (isLeader) {
        send({ type: 'PONG', tabId });
      }
      break;

    case 'PONG':
      // Someone else is leader — we are not
      if (electionPending) {
        electionPending = false;
        setLeader(msg.tabId);
      }
      break;

    case 'HEARTBEAT':
      lastHeartbeat = Date.now();
      if (leaderTabId !== msg.tabId) {
        setLeader(msg.tabId);
      }
      break;

    case 'REFRESH_REQUEST':
      // Follower is asking us to refresh — perform it if we're leader
      if (isLeader) {
        performRefreshForFollowers();
      }
      break;

    case 'REFRESH_RESULT':
      // Leader completed refresh — resolve pending request
      if (refreshResolvers) {
        refreshResolvers.resolve(msg.token);
        refreshResolvers = null;
      }
      break;

    case 'REFRESH_FAILED':
      // Leader failed — reject pending request
      if (refreshResolvers) {
        refreshResolvers.reject(new Error('Refresh failed on leader'));
        refreshResolvers = null;
      }
      break;

    case 'LOGOUT':
      // Another tab logged out — follow suit
      notifyLogout();
      break;
  }
}

function send(msg: Message) {
  try {
    channel?.postMessage(msg);
  } catch {
    // Channel may be closed
  }
}

// --- Leader election ---
function setLeader(newLeaderId: string) {
  if (leaderTabId === newLeaderId && isLeader === (newLeaderId === tabId)) return;

  leaderTabId = newLeaderId;
  const wasLeader = isLeader;
  isLeader = newLeaderId === tabId;

  if (wasLeader !== isLeader) {
    if (isLeader) {
      startHeartbeat();
    } else {
      stopHeartbeat();
    }
    notifyLeaderChange();
  }
}

function startElection() {
  if (electionPending) return;
  electionPending = true;

  send({ type: 'PING', tabId });

  setTimeout(() => {
    if (electionPending) {
      // No one responded — we are the leader
      electionPending = false;
      setLeader(tabId);
    }
  }, ELECTION_TIMEOUT);
}

// --- Heartbeat ---
function startHeartbeat() {
  stopHeartbeat();
  heartbeatTimer = setInterval(() => {
    send({ type: 'HEARTBEAT', tabId });
  }, HEARTBEAT_INTERVAL);
  // Send immediately
  send({ type: 'HEARTBEAT', tabId });
}

function stopHeartbeat() {
  if (heartbeatTimer) {
    clearInterval(heartbeatTimer);
    heartbeatTimer = null;
  }
}

function startLeaderCheck() {
  stopLeaderCheck();
  leaderCheckTimer = setInterval(() => {
    if (!isLeader && leaderTabId && Date.now() - lastHeartbeat > LEADER_TIMEOUT) {
      // Leader is gone — start election
      leaderTabId = null;
      startElection();
    }
  }, LEADER_TIMEOUT);
}

function stopLeaderCheck() {
  if (leaderCheckTimer) {
    clearInterval(leaderCheckTimer);
    leaderCheckTimer = null;
  }
}

// --- Refresh coordination ---
async function performRefreshForFollowers() {
  // Only called on the leader tab
  try {
    const { default: authApi } = await import('axios');
    const client = authApi.create({ baseURL: '/api', withCredentials: true });
    const response = await client.post('/refresh');
    const newToken = response.data.access_token as string;
    send({ type: 'REFRESH_RESULT', token: newToken });
  } catch {
    send({ type: 'REFRESH_FAILED' });
  }
}

// --- Public API ---

export function initTabCoordination(): void {
  if (typeof BroadcastChannel === 'undefined') return;

  channel = new BroadcastChannel(CHANNEL_NAME);
  channel.onmessage = handleMessage;

  // Start election
  startElection();
  startLeaderCheck();
}

export function destroyTabCoordination(): void {
  stopHeartbeat();
  stopLeaderCheck();
  channel?.close();
  channel = null;
  isLeader = false;
  leaderTabId = null;
  electionPending = false;
}

export function isTabLeader(): boolean {
  if (!channel) return true; // no channel = assume leader (fallback mode)
  return isLeader;
}

/**
 * Request a token refresh. If this tab is the leader, refreshes directly.
 * If follower, broadcasts a request and waits for the leader's result.
 */
export function requestRefresh(): Promise<string | null> {
  if (!channel) {
    // Fallback: no BroadcastChannel, cannot coordinate
    return Promise.resolve(null);
  }

  if (isLeader) {
    // Leader refreshes directly — caller should use doRefresh()
    return Promise.resolve(null); // signals "you're the leader, do it yourself"
  }

  // Follower: wait for leader's result
  return new Promise<string>((resolve, reject) => {
    refreshResolvers = { resolve, reject };
    send({ type: 'REFRESH_REQUEST', tabId });
  });
}

export function onLeaderChange(callback: (leader: boolean) => void): () => void {
  onLeaderChangeCallbacks.push(callback);
  return () => {
    onLeaderChangeCallbacks = onLeaderChangeCallbacks.filter((cb) => cb !== callback);
  };
}

export function onCrossTabLogout(callback: () => void): () => void {
  onLogoutCallbacks.push(callback);
  return () => {
    onLogoutCallbacks = onLogoutCallbacks.filter((cb) => cb !== callback);
  };
}

export function broadcastLogout(): void {
  send({ type: 'LOGOUT' });
}
