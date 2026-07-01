import { createRouter, createWebHistory } from "vue-router";
import { authStore } from "../stores/auth";
import LoginPage from "../pages/LoginPage.vue";
import RegisterPage from "../pages/RegisterPage.vue";
import PlannerPage from "../pages/PlannerPage.vue";
import PlansPage from "../pages/PlansPage.vue";
import PlanDetailPage from "../pages/PlanDetailPage.vue";

const routes = [
  {
    path: "/",
    redirect: "/planner"
  },
  {
    path: "/login",
    name: "login",
    component: LoginPage,
    meta: { guestOnly: true }
  },
  {
    path: "/register",
    name: "register",
    component: RegisterPage,
    meta: { guestOnly: true }
  },
  {
    path: "/planner",
    name: "planner",
    component: PlannerPage,
    meta: { requiresAuth: true }
  },
  {
    path: "/plans",
    name: "plans",
    component: PlansPage,
    meta: { requiresAuth: true }
  },
  {
    path: "/plans/:id",
    name: "plan-detail",
    component: PlanDetailPage,
    meta: { requiresAuth: true }
  }
];

const router = createRouter({
  history: createWebHistory(),
  routes
});

router.beforeEach((to) => {
  const isAuthenticated = authStore.isAuthenticated();

  if (to.meta.requiresAuth && !isAuthenticated) {
    return { name: "login", query: { redirect: to.fullPath } };
  }

  if (to.meta.guestOnly && isAuthenticated) {
    return { name: "planner" };
  }

  return true;
});

export default router;
