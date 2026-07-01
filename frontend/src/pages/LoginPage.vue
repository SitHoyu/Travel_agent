<template>
  <div class="auth-page">
    <div class="auth-card">
      <p class="auth-card__eyebrow">Travel Agent</p>
      <h1 class="auth-card__title">登录</h1>
      <p class="auth-card__text">
        先打通登录态，后面我们就可以继续把计划生成、保存和历史记录接起来。
      </p>

      <form class="form" @submit.prevent="handleSubmit">
        <label class="field">
          <span>用户名</span>
          <input
            v-model.trim="form.username"
            class="input"
            type="text"
            placeholder="请输入用户名"
            autocomplete="username"
          />
        </label>

        <label class="field">
          <span>密码</span>
          <input
            v-model="form.password"
            class="input"
            type="password"
            placeholder="请输入密码"
            autocomplete="current-password"
          />
        </label>

        <p v-if="errorMessage" class="message message--error">{{ errorMessage }}</p>

        <button class="button" type="submit" :disabled="submitting">
          {{ submitting ? "登录中..." : "登录" }}
        </button>
      </form>

      <p class="auth-card__footer">
        还没有账号？
        <RouterLink to="/register">前往注册</RouterLink>
      </p>
    </div>
  </div>
</template>

<script setup>
import { reactive, ref } from "vue";
import { useRoute, useRouter, RouterLink } from "vue-router";
import { login } from "../api/auth";
import { authStore } from "../stores/auth";

const route = useRoute();
const router = useRouter();

const form = reactive({
  username: "",
  password: ""
});

const submitting = ref(false);
const errorMessage = ref("");

async function handleSubmit() {
  errorMessage.value = "";

  if (!form.username || !form.password) {
    errorMessage.value = "请输入用户名和密码。";
    return;
  }

  submitting.value = true;
  try {
    const response = await login({
      username: form.username,
      password: form.password
    });
    authStore.saveSession(response.data);

    const redirect = typeof route.query.redirect === "string" ? route.query.redirect : "/planner";
    router.push(redirect);
  } catch (error) {
    errorMessage.value = error?.response?.data?.error || "登录失败，请稍后重试。";
  } finally {
    submitting.value = false;
  }
}
</script>
