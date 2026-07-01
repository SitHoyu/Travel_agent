<template>
  <div class="auth-page">
    <div class="auth-card">
      <p class="auth-card__eyebrow">Travel Agent</p>
      <h1 class="auth-card__title">注册</h1>
      <p class="auth-card__text">
        当前先完成最小注册闭环，注册成功后会直接进入系统。
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
          <span>昵称</span>
          <input
            v-model.trim="form.nickname"
            class="input"
            type="text"
            placeholder="可选昵称"
            autocomplete="nickname"
          />
        </label>

        <label class="field">
          <span>密码</span>
          <input
            v-model="form.password"
            class="input"
            type="password"
            placeholder="请输入密码"
            autocomplete="new-password"
          />
        </label>

        <p v-if="errorMessage" class="message message--error">{{ errorMessage }}</p>

        <button class="button" type="submit" :disabled="submitting">
          {{ submitting ? "注册中..." : "注册" }}
        </button>
      </form>

      <p class="auth-card__footer">
        已有账号？
        <RouterLink to="/login">返回登录</RouterLink>
      </p>
    </div>
  </div>
</template>

<script setup>
import { reactive, ref } from "vue";
import { useRouter, RouterLink } from "vue-router";
import { register } from "../api/auth";
import { authStore } from "../stores/auth";

const router = useRouter();

const form = reactive({
  username: "",
  nickname: "",
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
    const response = await register({
      username: form.username,
      nickname: form.nickname,
      password: form.password
    });
    authStore.saveSession(response.data);
    router.push("/planner");
  } catch (error) {
    errorMessage.value = error?.response?.data?.error || "注册失败，请稍后重试。";
  } finally {
    submitting.value = false;
  }
}
</script>
