import http from "./http";

export function generatePlan(payload) {
  return http.post("/v1/agent/plan/run", payload);
}

export function savePlan(payload) {
  return http.post("/v1/plans", payload);
}

export function listPlans(params = {}) {
  return http.get("/v1/plans", { params });
}

export function getPlanDetail(id) {
  return http.get(`/v1/plans/${id}`);
}
