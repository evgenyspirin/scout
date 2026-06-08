import { createSlice, type PayloadAction } from '@reduxjs/toolkit';

interface AuthState {
  token: string | null;
  login: string | null;
  role: string | null;
}

const STORAGE_KEY = 'scout.auth';

function loadInitial(): AuthState {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (raw) return JSON.parse(raw) as AuthState;
  } catch {
    // ignore malformed storage
  }
  return { token: null, login: null, role: null };
}

interface Credentials {
  token: string;
  login: string;
  role: string;
}

const authSlice = createSlice({
  name: 'auth',
  initialState: loadInitial(),
  reducers: {
    setCredentials(state, action: PayloadAction<Credentials>) {
      state.token = action.payload.token;
      state.login = action.payload.login;
      state.role = action.payload.role;
      localStorage.setItem(STORAGE_KEY, JSON.stringify(state));
    },
    logout(state) {
      state.token = null;
      state.login = null;
      state.role = null;
      localStorage.removeItem(STORAGE_KEY);
    },
  },
});

export const { setCredentials, logout } = authSlice.actions;
export default authSlice.reducer;

/** Decodes the `role` from a JWT payload without verifying the signature. */
export function roleFromToken(token: string): string {
  try {
    const payload = JSON.parse(atob(token.split('.')[1]));
    return typeof payload.role === 'string' ? payload.role : 'user';
  } catch {
    return 'user';
  }
}
