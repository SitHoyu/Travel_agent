import { reactive } from "vue";

const TOKEN_KEY = "travel_agent_token";
const USER_KEY = "travel_agent_user";

function readStoredUser() {
  const raw = localStorage.getItem(USER_KEY);
  if (!raw) {
    return null;
  }

  try {
    return JSON.parse(raw);
  } catch {
    localStorage.removeItem(USER_KEY);
    return null;
  }
}

const state = reactive({
  token: localStorage.getItem(TOKEN_KEY) || "",
  user: readStoredUser()
});

export const authStore = {
  state,
  getToken() {
    return state.token;
  },
  setToken(token) {
    state.token = token || "";
    if (state.token) {
      localStorage.setItem(TOKEN_KEY, state.token);
      return;
    }
    localStorage.removeItem(TOKEN_KEY);
  },
  clearToken() {
    state.token = "";
    localStorage.removeItem(TOKEN_KEY);
  },
  getUser() {
    return state.user;
  },
  setUser(user) {
    state.user = user || null;
    if (state.user) {
      localStorage.setItem(USER_KEY, JSON.stringify(state.user));
      return;
    }
    localStorage.removeItem(USER_KEY);
  },
  clearUser() {
    state.user = null;
    localStorage.removeItem(USER_KEY);
  },
  saveSession(payload) {
    this.setToken(payload?.access_token || "");
    this.setUser(payload?.user || null);
  },
  isAuthenticated() {
    return Boolean(state.token);
  },
  logout() {
    this.clearToken();
    this.clearUser();
  }
};
