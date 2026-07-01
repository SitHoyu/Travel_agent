import http from "./http";

export function register(payload) {
  return http.post("/v1/auth/register", payload);
}

export function login(payload) {
  return http.post("/v1/auth/login", payload);
}

export function getCurrentUser() {
  return http.get("/v1/users/me");
}
