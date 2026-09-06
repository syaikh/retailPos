# Cross-Tab Refresh Coordination — Assessment & Plan

## Current State

| Aspect | Implementation | Gap |
|--------|---------------|-----|
| Access token storage | `sessionStorage` (tab-scoped) | Each tab has its own token copy |
| Refresh token storage | httpOnly cookie (shared across tabs) | ✅ Already shared |
| Single-tab dedup | Queue pattern in `doRefresh()` | ✅ Fixed (commit 6f0462e) |
| Cross-tab communication | None | ❌ No BroadcastChannel, no storage events |
| Tab detection | None | ❌ No visibility, focus, or tab ID tracking |
| Proactive refresh | Each tab runs independent 13-min timer | ❌ N tabs = N refresh requests |
| Reactive refresh (401) | Each tab handles independently | ❌ N tabs × concurrent 401s = N refresh attempts |
| Logout propagation | None | ❌ Logout in Tab A leaves Tab B stale |
| Rate limiting | 10 RPM per IP on `/api/refresh` | ⚠️ Many tabs could hit limit |

---

## Problem Analysis

**With the current single-tab queue fix**, each tab still independently triggers refreshes. The cascading refresh race condition is fixed *within* a tab, but *across* tabs:

- 5 tabs × 13-min timer = 5 refresh requests per cycle (unnecessary)
- Server restart → all 5 tabs hit 401 simultaneously → 5 refresh attempts (rate limit risk)
- Token rotation: each refresh consumes a refresh token → 5 tabs = 5 consumed tokens per cycle (DB bloat)

**The refresh token cookie is already shared** across tabs (httpOnly, path=/). The problem is only the *coordination* of who triggers the refresh.

---

## Proposed Solution

**Use `BroadcastChannel` for leader election and token distribution.**

### Architecture

```
Tab A (leader)                Tab B (follower)           Tab C (follower)
     │                             │                          │
     ├── POST /api/refresh ──────> │                          │
     │                             │                          │
     ├── broadcast("token", new) ──┼──> receives token        │
     │                             │──> setAccessToken()      │
     │                             │                          │
     │                             │      broadcast("token",)──┼──> receives token
```

### Leader Election

- On tab open: each tab generates a random `tabId`
- Broadcasts `PING` on a channel
- If no `PONG` within 500ms → tab becomes leader
- If receives `PONG` → tab is follower
- Leader broadcasts `HEARTBEAT` every 5 seconds
- If follower doesn't hear heartbeat for 10s → triggers new election

### Refresh Coordination

1. **Proactive timer**: Only leader runs the timer
2. **401 trigger**: Follower broadcasts `REFRESH_REQUEST`, leader performs it
3. **Result**: Leader broadcasts `REFRESH_RESULT` with new token, all tabs apply it
4. **Failure**: Leader broadcasts `REFRESH_FAILED`, followers retry individually

---

## Implementation Plan

### Step 1: Create `web/src/shared/utils/tab-coordination.ts`

New module with:
- `tabId` generation (crypto.randomUUID or fallback)
- `BroadcastChannel` wrapper with `PING`/`PONG`/`HEARTBEAT`/`REFRESH_REQUEST`/`REFRESH_RESULT` messages
- Leader election logic
- Heartbeat mechanism
- `isLeader()` getter
- `onLeaderChange(callback)` for consumers

### Step 2: Modify `web/src/modules/auth/services/auth-service.ts`

- Import `tab-coordination`
- `startProactiveRefresh()`: only start timer if `isLeader()`
- `doRefresh()`: if not leader, broadcast `REFRESH_REQUEST` and wait for `REFRESH_RESULT`
- `onLeaderChange`: if tab becomes leader, start proactive timer; if loses leadership, stop timer

### Step 3: Handle leader crash

- Follower detects missing heartbeat (10s timeout)
- Triggers new election
- New leader starts proactive timer
- If a refresh was in-flight when leader crashed, new leader retries

### Step 4: Handle logout propagation

- On logout, broadcast `LOGOUT` event
- All tabs receive and call `clearUser()` + redirect to `/login`

### Step 5: Fallback for older browsers

- `BroadcastChannel` is supported in all modern browsers (Chrome 54+, Firefox 38+, Safari 14.1+)
- Fallback: skip coordination, each tab refreshes independently (current behavior)
- Detection: `typeof BroadcastChannel !== 'undefined'`

---

## Edge Cases

| Scenario | Handling |
|----------|----------|
| Leader tab closes | Heartbeat stops → follower detects → new election |
| Leader tab crashes (unresponsive) | Same as above (heartbeat timeout) |
| Leader navigates to different SPA route | Tab still alive, heartbeat continues |
| All tabs close simultaneously | No coordination needed |
| New tab opens while refresh in-flight | New tab joins as follower, waits for next heartbeat |
| Leader becomes follower mid-refresh | Refresh completes, result broadcast still sent |
| `BroadcastChannel` not supported | Fallback to independent refresh (current behavior) |

---

## Browser Compatibility

| API | Chrome | Firefox | Safari | Edge |
|-----|--------|---------|--------|------|
| `BroadcastChannel` | 54+ | 38+ | 14.1+ | 79+ |
| `crypto.randomUUID` | 92+ | 95+ | 15.4+ | 92+ |

Both are well-supported in modern browsers. Fallback for older browsers is simple (skip coordination).

---

## Impact Assessment

| Metric | Before | After |
|--------|--------|-------|
| Refresh requests per cycle (5 tabs) | 5 | 1 |
| DB refresh token rows per cycle | 5 | 1 |
| Risk of hitting rate limit (10 RPM) | High with many tabs | Low |
| Complexity | Low | Moderate |
| New files | 0 | 1 (`tab-coordination.ts`) |
| Modified files | 0 | 1 (`auth-service.ts`) |

---

## Recommendation

**Implement it.** The benefits are clear:

- Eliminates unnecessary refresh requests across tabs
- Reduces DB load from refresh token rotation
- Prevents rate limit issues in multi-tab scenarios
- Adds logout propagation (bonus feature)
- Moderate complexity, well-contained change

The single-tab queue fix we just did handles intra-tab concurrency. Cross-tab coordination handles inter-tab concurrency. Together they provide complete refresh deduplication.

---

## Implementation Progress

### ✅ Completed

| Step | Status | Files |
|------|--------|-------|
| Step 1: Create `tab-coordination.ts` | ✅ Done | `web/src/shared/utils/tab-coordination.ts` |
| Step 2: Modify `auth-service.ts` for proactive refresh | ✅ Done | `web/src/modules/auth/services/auth-service.ts` |
| Step 3: Modify `auth-service.ts` for reactive refresh (401) | ✅ Done | `web/src/modules/auth/services/auth-service.ts` |
| Step 4: Logout propagation | ✅ Done | `web/src/modules/auth/services/auth-service.ts`, `web/src/modules/auth/index.ts`, `web/src/app/main.svelte` |
| Step 5: Fallback for older browsers | ✅ Done | Built into `tab-coordination.ts` (`isTabLeader()` returns `true` when `BroadcastChannel` unavailable) |
| Tests | ✅ 22/22 passing | `web/src/modules/auth/services/__tests__/auth-service.test.ts` |
| Build | ✅ Passing | `web` package |

### Changes Summary

**New file:**
- `web/src/shared/utils/tab-coordination.ts` — BroadcastChannel-based leader election, heartbeat, refresh coordination, logout propagation

**Modified files:**
- `web/src/modules/auth/services/auth-service.ts` — Integrated tab-coordination for proactive and reactive refresh
- `web/src/modules/auth/index.ts` — Added `handleCrossTabLogout` export
- `web/src/app/main.svelte` — Added `handleCrossTabLogout()` initialization
- `web/src/modules/auth/services/__tests__/auth-service.test.ts` — Added mock for `tab-coordination`
