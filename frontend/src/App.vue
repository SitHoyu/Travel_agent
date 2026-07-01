<template>
  <div class="app-shell">
    <RouterView />
  </div>
</template>

<script setup>
import { onMounted } from "vue";
import { RouterView } from "vue-router";
import { getCurrentUser } from "./api/auth";
import { authStore } from "./stores/auth";

onMounted(async () => {
  if (!authStore.isAuthenticated()) {
    return;
  }

  try {
    const response = await getCurrentUser();
    authStore.setUser(response.data);
  } catch {
    authStore.logout();
  }
});
</script>
