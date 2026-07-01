<template>
  <AppLayout title="我的计划">
    <template #headerActions>
      <div class="actions">
        <button class="button button--ghost" type="button" :disabled="loading" @click="fetchPlans">
          {{ loading ? "刷新中..." : "刷新列表" }}
        </button>
      </div>
    </template>

    <div class="stack">
      <div class="card">
        <h3 class="card__title">计划列表</h3>
        <p class="card__text">
          当前展示当前登录用户已保存的计划摘要信息，便于快速回看和进入详情页。
        </p>
      </div>

      <p v-if="errorMessage" class="message message--error">{{ errorMessage }}</p>

      <div v-if="loading" class="card">
        <div class="empty-state">
          <p class="empty-state__title">正在加载计划列表</p>
          <p class="empty-state__text">后端正在查询当前用户的保存记录，请稍候。</p>
        </div>
      </div>

      <div v-else-if="items.length === 0" class="card">
        <div class="empty-state">
          <p class="empty-state__title">还没有保存的计划</p>
          <p class="empty-state__text">
            你可以先去“生成计划”页面生成一个结果，再确认保存，这里就会出现记录。
          </p>
          <div class="actions actions--center">
            <RouterLink class="button button--link" to="/planner">前往生成计划</RouterLink>
          </div>
        </div>
      </div>

      <div v-else class="stack">
        <div class="summary-grid summary-grid--three">
          <div class="mini-card">
            <span class="mini-card__label">总记录数</span>
            <strong>{{ total }}</strong>
          </div>
          <div class="mini-card">
            <span class="mini-card__label">当前页</span>
            <strong>{{ page }}</strong>
          </div>
          <div class="mini-card">
            <span class="mini-card__label">每页条数</span>
            <strong>{{ pageSize }}</strong>
          </div>
        </div>

        <article v-for="plan in items" :key="plan.id" class="plan-list-card">
          <div class="plan-list-card__header">
            <div>
              <div class="plan-list-card__title-row">
                <h3 class="plan-list-card__title">{{ plan.title || "未命名计划" }}</h3>
                <span class="status-badge">{{ plan.status || "saved" }}</span>
              </div>
              <p class="plan-list-card__meta">
                目的地：{{ plan.destination || "未填写" }} · 请求 ID：{{ plan.request_id || "无" }}
              </p>
            </div>

            <RouterLink class="button button--ghost button--link" :to="`/plans/${plan.id}`">
              查看详情
            </RouterLink>
          </div>

          <p class="plan-list-card__summary">
            {{ plan.summary || "当前计划没有摘要信息。" }}
          </p>

          <div class="plan-list-card__footer">
            <span>计划 ID：{{ plan.id }}</span>
            <span>保存时间：{{ formatDate(plan.created_at) }}</span>
          </div>
        </article>
      </div>
    </div>
  </AppLayout>
</template>

<script setup>
import { onMounted, ref } from "vue";
import { RouterLink } from "vue-router";
import AppLayout from "../layouts/AppLayout.vue";
import { listPlans } from "../api/plans";

const items = ref([]);
const total = ref(0);
const page = ref(1);
const pageSize = ref(20);
const loading = ref(false);
const errorMessage = ref("");

onMounted(() => {
  fetchPlans();
});

async function fetchPlans() {
  loading.value = true;
  errorMessage.value = "";

  try {
    const response = await listPlans({
      page: page.value,
      page_size: pageSize.value
    });

    items.value = response.data?.items || [];
    total.value = response.data?.total || 0;
    page.value = response.data?.page || 1;
    pageSize.value = response.data?.page_size || 20;
  } catch (error) {
    errorMessage.value = error?.response?.data?.error || "获取计划列表失败，请稍后重试。";
  } finally {
    loading.value = false;
  }
}

function formatDate(value) {
  if (!value) {
    return "未知时间";
  }

  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }

  return new Intl.DateTimeFormat("zh-CN", {
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit"
  }).format(date);
}
</script>
