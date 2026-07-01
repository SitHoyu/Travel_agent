<template>
  <div class="layout">
    <aside class="layout__sidebar">
      <div>
        <p class="layout__eyebrow">Travel Agent</p>
        <h1 class="layout__title">行程规划台</h1>
        <p class="layout__text">
          当前先做一个最小可用前端，后续再补完整的计划交互与任务流。
        </p>
      </div>

      <nav class="layout__nav">
        <RouterLink class="layout__nav-link" to="/planner">生成计划</RouterLink>
        <RouterLink class="layout__nav-link" to="/plans">我的计划</RouterLink>
      </nav>

      <div class="layout__profile">
        <p class="layout__profile-name">{{ userLabel }}</p>
        <button class="button button--ghost" type="button" @click="handleLogout">
          退出登录
        </button>
      </div>
    </aside>

    <main class="layout__content">
      <header class="layout__header">
        <div>
          <p class="layout__eyebrow">Frontend MVP</p>
          <h2 class="layout__section-title">{{ title }}</h2>
        </div>
        <slot name="headerActions" />
      </header>

      <section class="layout__panel">
        <slot />
      </section>
    </main>
  </div>
</template>

<script setup>
import { computed } from "vue";
import { useRouter, RouterLink } from "vue-router";
import { authStore } from "../stores/auth";

defineProps({
  title: {
    type: String,
    default: ""
  }
});

const router = useRouter();

const userLabel = computed(() => {
  const user = authStore.state.user;
  if (!user) {
    return "未登录";
  }
  return user.nickname || user.username || "当前用户";
});

function handleLogout() {
  authStore.logout();
  router.push("/login");
}
</script>
