import axios from "axios";
import { authStore } from "../stores/auth";

const http = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL || "http://localhost:8080",
  timeout: 120000
});

http.interceptors.request.use((config) => {
  const token = authStore.getToken();
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

export default http;
