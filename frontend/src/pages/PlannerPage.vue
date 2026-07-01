<template>
  <AppLayout title="生成计划">
    <template #headerActions>
      <div class="actions">
        <RouterLink class="button button--ghost button--link" to="/plans">
          查看我的计划
        </RouterLink>
      </div>
    </template>

    <div class="stack">
      <div class="card">
        <h3 class="card__title">行程需求</h3>
        <p class="card__text">
          当前页面已经接入真实后端。你可以先生成计划，再决定是否确认保存。
        </p>
      </div>

      <div class="planner-grid">
        <div class="card">
          <form class="form" @submit.prevent="handleGenerate">
            <label class="field">
              <span>请求 ID</span>
              <input
                v-model.trim="form.request_id"
                class="input"
                type="text"
                placeholder="例如 agent-web-001"
              />
            </label>

            <label class="field">
              <span>目的地</span>
              <input
                v-model.trim="form.destination"
                class="input"
                type="text"
                placeholder="例如 揭阳"
              />
            </label>

            <div class="field-row">
              <label class="field">
                <span>开始日期</span>
                <input v-model="form.start_date" class="input" type="date" />
              </label>

              <label class="field">
                <span>结束日期</span>
                <input v-model="form.end_date" class="input" type="date" />
              </label>
            </div>

            <div class="field-row">
              <label class="field">
                <span>预算</span>
                <input
                  v-model.trim="form.budget"
                  class="input"
                  type="text"
                  placeholder="例如 3000 RMB"
                />
              </label>

              <label class="field">
                <span>人数</span>
                <input
                  v-model.number="form.travelers"
                  class="input"
                  type="number"
                  min="1"
                  placeholder="2"
                />
              </label>
            </div>

            <label class="field">
              <span>偏好</span>
              <textarea
                v-model="preferencesText"
                class="input textarea"
                rows="3"
                placeholder="每行一个偏好，例如：&#10;慢节奏&#10;美食&#10;古城"
              />
            </label>

            <label class="field">
              <span>约束</span>
              <textarea
                v-model="constraintsText"
                class="input textarea"
                rows="3"
                placeholder="每行一个约束，例如：&#10;不赶行程&#10;每天最多两个大景点"
              />
            </label>

            <p v-if="errorMessage" class="message message--error">{{ errorMessage }}</p>
            <p v-if="saveErrorMessage" class="message message--error">{{ saveErrorMessage }}</p>
            <p v-if="saveSuccessMessage" class="message message--success">{{ saveSuccessMessage }}</p>

            <div class="actions">
              <button class="button" type="submit" :disabled="submitting">
                {{ submitting ? "生成中..." : "生成计划" }}
              </button>

              <button class="button button--ghost" type="button" @click="fillDemo">
                填充示例
              </button>

              <button
                v-if="result"
                class="button button--secondary"
                type="button"
                :disabled="saving"
                @click="handleSave"
              >
                {{ saving ? "保存中..." : saveButtonLabel }}
              </button>
            </div>

            <div v-if="savedPlan" class="actions">
              <RouterLink class="button button--ghost button--link" :to="`/plans/${savedPlan.id}`">
                查看刚保存的详情
              </RouterLink>
              <RouterLink class="button button--ghost button--link" to="/plans">
                前往我的计划列表
              </RouterLink>
            </div>
          </form>
        </div>

        <div class="card">
          <h3 class="card__title">请求预览</h3>
          <p class="card__text">发送前会按后端契约整理为结构化 JSON。</p>
          <pre class="code-block">{{ requestPreview }}</pre>
        </div>
      </div>

      <div class="card">
        <div class="result-header">
          <div>
            <h3 class="card__title">生成结果</h3>
            <p class="card__text">
              先确认生成结果是否符合预期，再点击“确认保存”写入数据库。
            </p>
          </div>
          <span v-if="result" class="status-badge">{{ result.status || "completed" }}</span>
        </div>

        <div v-if="submitting" class="empty-state">
          <p class="empty-state__title">正在生成计划</p>
          <p class="empty-state__text">后端正在调用 agent、天气和推荐链路，请稍候。</p>
        </div>

        <div v-else-if="result" class="stack">
          <div class="summary-grid">
            <div class="mini-card">
              <span class="mini-card__label">会话 ID</span>
              <strong>{{ result.session_id }}</strong>
            </div>
            <div class="mini-card">
              <span class="mini-card__label">请求 ID</span>
              <strong>{{ result.request_id }}</strong>
            </div>
            <div class="mini-card">
              <span class="mini-card__label">工具次数</span>
              <strong>{{ result.tool_runs }}</strong>
            </div>
            <div class="mini-card">
              <span class="mini-card__label">消息数</span>
              <strong>{{ result.message_count }}</strong>
            </div>
          </div>

          <div class="card card--inner">
            <h4 class="card__title">最终答复</h4>
            <p class="result-text">{{ result.final_answer }}</p>
          </div>

          <div class="planner-grid">
            <div class="card card--inner">
              <h4 class="card__title">计划概览</h4>
              <ul class="list">
                <li>标题：{{ result.plan?.title || "未返回" }}</li>
                <li>目的地：{{ result.plan?.destination || "未返回" }}</li>
                <li>天数：{{ result.plan?.days?.length || 0 }}</li>
              </ul>
              <p class="card__text">{{ result.plan?.summary || "暂无摘要" }}</p>
            </div>

            <div class="card card--inner">
              <h4 class="card__title">酒店区域建议</h4>
              <p class="card__text">
                {{ result.hotel_areas?.summary || "当前未返回酒店区域建议。" }}
              </p>
            </div>
          </div>

          <div class="card card--inner">
            <h4 class="card__title">原始返回 JSON</h4>
            <pre class="code-block">{{ resultPreview }}</pre>
          </div>
        </div>

        <div v-else class="empty-state">
          <p class="empty-state__title">还没有生成结果</p>
          <p class="empty-state__text">
            填完左侧表单后点击“生成计划”，这里会展示后端返回的实际内容。
          </p>
        </div>
      </div>
    </div>
  </AppLayout>
</template>

<script setup>
import { computed, reactive, ref } from "vue";
import { RouterLink } from "vue-router";
import AppLayout from "../layouts/AppLayout.vue";
import { generatePlan, savePlan } from "../api/plans";

const form = reactive(createDemoPayload());
const preferencesText = ref(form.preferences.join("\n"));
const constraintsText = ref(form.constraints.join("\n"));
const submitting = ref(false);
const saving = ref(false);
const errorMessage = ref("");
const saveErrorMessage = ref("");
const saveSuccessMessage = ref("");
const result = ref(null);
const savedPlan = ref(null);

const requestPreview = computed(() => JSON.stringify(buildPayload(), null, 2));
const resultPreview = computed(() => (result.value ? JSON.stringify(result.value, null, 2) : ""));
const saveButtonLabel = computed(() => (savedPlan.value ? "再次保存当前结果" : "确认保存"));

function createDemoPayload() {
  return {
    request_id: buildRequestID(),
    destination: "揭阳",
    start_date: "2026-07-01",
    end_date: "2026-07-03",
    budget: "3000 RMB",
    travelers: 2,
    preferences: ["慢节奏", "美食", "古城"],
    constraints: ["不赶行程", "每天最多两个大景点"]
  };
}

function buildRequestID() {
  return `agent-web-${Date.now()}`;
}

function fillDemo() {
  const demo = createDemoPayload();
  Object.assign(form, demo);
  preferencesText.value = demo.preferences.join("\n");
  constraintsText.value = demo.constraints.join("\n");
}

function splitLines(text) {
  return text
    .split(/\r?\n/)
    .map((item) => item.trim())
    .filter(Boolean);
}

function buildPayload() {
  return {
    request_id: form.request_id || buildRequestID(),
    destination: form.destination.trim(),
    start_date: form.start_date,
    end_date: form.end_date,
    budget: form.budget.trim(),
    travelers: Number(form.travelers) || 1,
    preferences: splitLines(preferencesText.value),
    constraints: splitLines(constraintsText.value)
  };
}

async function handleGenerate() {
  errorMessage.value = "";
  saveErrorMessage.value = "";
  saveSuccessMessage.value = "";
  result.value = null;
  savedPlan.value = null;

  const payload = buildPayload();
  if (!payload.destination || !payload.start_date || !payload.end_date || !payload.budget) {
    errorMessage.value = "请至少填写目的地、开始日期、结束日期和预算。";
    return;
  }
  if (payload.preferences.length === 0) {
    errorMessage.value = "请至少填写一个偏好。";
    return;
  }
  if (payload.constraints.length === 0) {
    errorMessage.value = "请至少填写一个约束。";
    return;
  }

  form.request_id = payload.request_id;
  submitting.value = true;
  try {
    const response = await generatePlan(payload);
    result.value = response.data;
  } catch (error) {
    errorMessage.value = error?.response?.data?.error || "生成失败，请稍后重试。";
  } finally {
    submitting.value = false;
  }
}

async function handleSave() {
  saveErrorMessage.value = "";
  saveSuccessMessage.value = "";

  if (!result.value) {
    saveErrorMessage.value = "请先生成计划，再执行保存。";
    return;
  }

  saving.value = true;
  try {
    const response = await savePlan({
      user_id: 0,
      request: buildPayload(),
      result: result.value
    });
    savedPlan.value = response.data;
    saveSuccessMessage.value = `计划已保存成功，记录 ID：${response.data.id}`;
  } catch (error) {
    saveErrorMessage.value = error?.response?.data?.error || "保存失败，请稍后重试。";
  } finally {
    saving.value = false;
  }
}
</script>
