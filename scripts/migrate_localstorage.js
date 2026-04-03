/**
 * PRDS localStorage → PostgreSQL Migration Script
 * 
 * STEP 1: Run this in your browser console on the PRDS page
 *         to export your localStorage data.
 * 
 * STEP 2: Copy the output JSON
 * 
 * STEP 3: POST it to /api/v1/migrate/localstorage with your JWT token
 * 
 * Usage:
 *   const data = exportLocalStorage();
 *   console.log(JSON.stringify(data));
 */

function exportLocalStorage() {
  const state = {};

  // Try to read STATE object if still on the page
  if (typeof STATE !== 'undefined') {
    state.xp = STATE.xp || 0;
    state.streak = STATE.streak || 0;
    state.completed = [...(STATE.completedExercises || [])];
    state.inProgress = [...(STATE.inProgressExercises || [])];
    state.concepts = [...(STATE.masteredConcepts || [])];
  } else {
    state.xp = 0;
    state.streak = 0;
    state.completed = [];
    state.inProgress = [];
    state.concepts = [];
  }

  // Extract codes
  state.codes = {};
  state.notes = {};
  state.sr = {};

  for (const key of Object.keys(localStorage)) {
    if (key.startsWith('code_')) {
      state.codes[key] = localStorage.getItem(key);
    } else if (key.startsWith('notes_')) {
      state.notes[key] = localStorage.getItem(key);
    } else if (key.startsWith('sr_')) {
      try {
        state.sr[key] = JSON.parse(localStorage.getItem(key));
      } catch (e) {}
    } else if (key === 'prds_state') {
      try {
        const s = JSON.parse(localStorage.getItem(key));
        if (s) {
          state.xp = s.xp || state.xp;
          state.streak = s.streak || state.streak;
          if (s.completed) state.completed = s.completed;
          if (s.inProgress) state.inProgress = s.inProgress;
          if (s.concepts) state.concepts = s.concepts;
        }
      } catch (e) {}
    }
  }

  return state;
}

/**
 * Migrate data to backend.
 * Call this after logging in and getting a JWT token.
 * 
 * @param {string} apiBase  e.g. "http://localhost:8080"
 * @param {string} token    JWT token from login
 */
async function migrateToBackend(apiBase, token) {
  const data = exportLocalStorage();

  console.log('Exporting:', {
    xp: data.xp,
    completed: data.completed.length,
    inProgress: data.inProgress.length,
    srCards: Object.keys(data.sr).length,
    codeFiles: Object.keys(data.codes).length,
  });

  const response = await fetch(`${apiBase}/api/v1/migrate/localstorage`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${token}`,
    },
    body: JSON.stringify(data),
  });

  if (!response.ok) {
    const err = await response.text();
    throw new Error(`Migration failed: ${err}`);
  }

  const result = await response.json();
  console.log('✓ Migration complete:', result);
  return result;
}

// ─── Quick usage example ───────────────────────────────────────────────────
/*

// 1. Register / login
const login = await fetch('http://localhost:8080/api/v1/auth/login', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({ email: 'you@example.com', password: 'yourpassword' })
}).then(r => r.json());

const token = login.token;
console.log('Logged in, token:', token.slice(0,20) + '...');

// 2. Migrate
await migrateToBackend('http://localhost:8080', token);

*/
