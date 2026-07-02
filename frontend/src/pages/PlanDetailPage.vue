<template>
  <AppLayout :title="pageTitle">
    <template #headerActions>
      <div class="actions">
        <RouterLink class="button button--ghost button--link" to="/plans">
          返回计划列表
        </RouterLink>
      </div>
    </template>

    <div class="stack">
      <div v-if="errorMessage" class="message message--error">{{ errorMessage }}</div>

      <div v-if="loading" class="card">
        <div class="empty-state">
          <p class="empty-state__title">正在加载计划详情</p>
          <p class="empty-state__text">后端正在查询当前用户的完整计划记录，请稍候。</p>
        </div>
      </div>

      <template v-else-if="detail">
        <div class="summary-grid">
          <div class="mini-card">
            <span class="mini-card__label">记录 ID</span>
            <strong>{{ detail.id }}</strong>
          </div>
          <div class="mini-card">
            <span class="mini-card__label">请求 ID</span>
            <strong>{{ detail.request_id || "无" }}</strong>
          </div>
          <div class="mini-card">
            <span class="mini-card__label">状态</span>
            <strong>{{ detail.status || "saved" }}</strong>
          </div>
          <div class="mini-card">
            <span class="mini-card__label">保存时间</span>
            <strong>{{ formatDate(detail.created_at) }}</strong>
          </div>
        </div>

        <div v-if="mapMarkers.length > 0" class="card">
          <h3 class="card__title">地图预览</h3>
          <p class="card__text">
            当前已将行程活动点、推荐住宿中心和附近酒店点展示到地图上。后续可以继续区分景点与酒店样式。
          </p>
          <MapContainer :markers="mapMarkers" height="520px" />
        </div>

        <div class="detail-grid">
          <div class="card">
            <h3 class="card__title">计划概览</h3>
            <div class="detail-meta">
              <p><strong>标题：</strong>{{ detail.title || "未命名计划" }}</p>
              <p><strong>目的地：</strong>{{ detail.destination || "未填写" }}</p>
              <p><strong>天数：</strong>{{ detail.plan?.days?.length || 0 }}</p>
              <p><strong>Session：</strong>{{ detail.session_id || "无" }}</p>
            </div>
            <p class="card__text detail-summary">
              {{ detail.summary || detail.plan?.summary || "当前计划暂无摘要。" }}
            </p>
          </div>

          <div class="card">
            <h3 class="card__title">计划说明</h3>
            <p class="result-text">
              {{ detail.final_answer || "当前未返回最终答复。" }}
            </p>
          </div>
        </div>

        <div v-if="detail.validation_summary" class="card">
          <h3 class="card__title">约束校验</h3>
          <p class="result-text">{{ detail.validation_summary }}</p>
        </div>

        <div class="card">
          <h3 class="card__title">酒店区域推荐</h3>
          <p class="card__text">
            {{ detail.hotel_areas?.summary || "当前未返回酒店区域推荐。" }}
          </p>

          <div
            v-if="detail.hotel_areas?.recommendations?.length"
            class="recommendation-list"
          >
            <article
              v-for="item in detail.hotel_areas.recommendations"
              :key="`${item.area}-${item.priority}`"
              class="recommendation-card"
            >
              <div class="recommendation-card__header">
                <div>
                  <h4 class="recommendation-card__title">{{ item.area }}</h4>
                  <p class="recommendation-card__meta">
                    优先级 {{ item.priority }} · {{ item.price_range || "价格待补充" }}
                  </p>
                </div>
              </div>

              <p class="recommendation-card__reason">
                {{ item.fit_reason || "暂无推荐说明。" }}
              </p>

              <div v-if="item.suitable_for?.length" class="tag-list">
                <span v-for="tag in item.suitable_for" :key="tag" class="tag">{{ tag }}</span>
              </div>
            </article>
          </div>

          <p v-if="detail.hotel_areas?.nearby_hotels_error" class="message message--error">
            酒店候选加载说明：{{ detail.hotel_areas.nearby_hotels_error }}
          </p>
        </div>

        <div class="card">
          <div class="result-header">
            <div>
              <h3 class="card__title">附近酒店候选</h3>
              <p class="card__text">
                当前先展示酒店卡片与图片，后续再把酒店点位样式单独优化。
              </p>
            </div>
          </div>

          <div v-if="detail.hotel_areas?.nearby_hotels?.length" class="hotel-grid">
            <article
              v-for="hotel in detail.hotel_areas.nearby_hotels"
              :key="`${hotel.name}-${hotel.location?.longitude}-${hotel.location?.latitude}`"
              class="hotel-card"
            >
              <div class="hotel-card__media">
                <img
                  v-if="hotel.photo_url && !brokenImages[hotel.photo_url]"
                  :src="hotel.photo_url"
                  :alt="hotel.name"
                  class="hotel-card__image"
                  @error="markBrokenImage(hotel.photo_url)"
                />
                <div v-else class="hotel-card__placeholder">暂无图片</div>
              </div>

              <div class="hotel-card__body">
                <h4 class="hotel-card__title">{{ hotel.name || "未命名酒店" }}</h4>
                <p class="hotel-card__text">{{ hotel.address || "暂无地址信息" }}</p>
                <p class="hotel-card__meta">
                  距离推荐中心：{{ formatDistance(hotel.distance_m) }}
                </p>
                <p class="hotel-card__meta">
                  坐标：{{ formatCoordinate(hotel.location?.longitude) }},
                  {{ formatCoordinate(hotel.location?.latitude) }}
                </p>
              </div>
            </article>
          </div>

          <div v-else class="empty-state">
            <p class="empty-state__title">暂无附近酒店候选</p>
            <p class="empty-state__text">当前记录没有返回可直接展示的酒店列表。</p>
          </div>
        </div>

        <div class="card">
          <h3 class="card__title">逐日行程</h3>

          <div v-if="detail.plan?.days?.length" class="day-list">
            <article
              v-for="day in detail.plan.days"
              :key="`${day.day}-${day.date}`"
              class="day-card"
            >
              <div class="day-card__header">
                <div>
                  <h4 class="day-card__title">Day {{ day.day }} · {{ day.theme || "未命名主题" }}</h4>
                  <p class="day-card__date">{{ day.date || "未填写日期" }}</p>
                </div>
              </div>

              <div v-if="day.activities?.length" class="activity-list">
                <article
                  v-for="(activity, index) in day.activities"
                  :key="`${day.day}-${index}-${activity.name}`"
                  class="activity-card"
                >
                  <div class="activity-card__header">
                    <div>
                      <h5 class="activity-card__title">{{ activity.name || "未命名活动" }}</h5>
                      <p class="activity-card__meta">
                        {{ activity.time_slot || "时间待补充" }} ·
                        {{ activity.type || "other" }} ·
                        {{ activity.indoor ? "室内" : "室外" }}
                      </p>
                    </div>
                  </div>

                  <p class="activity-card__location">
                    地点：{{ activity.location || activity.resolved_address || "暂无地点信息" }}
                  </p>
                  <p class="activity-card__description">
                    {{ activity.description || "暂无活动描述。" }}
                  </p>

                  <p
                    v-if="typeof activity.longitude === 'number' && typeof activity.latitude === 'number'"
                    class="activity-card__coord"
                  >
                    坐标：{{ formatCoordinate(activity.longitude) }},
                    {{ formatCoordinate(activity.latitude) }}
                  </p>
                </article>
              </div>

              <div v-else class="empty-state empty-state--compact">
                <p class="empty-state__text">这一天暂无活动明细。</p>
              </div>
            </article>
          </div>

          <div v-else class="empty-state">
            <p class="empty-state__title">暂无逐日行程</p>
            <p class="empty-state__text">当前计划没有返回可展示的天级行程结构。</p>
          </div>
        </div>
      </template>
    </div>
  </AppLayout>
</template>

<script setup>
import { computed, onMounted, reactive, ref } from "vue";
import { RouterLink, useRoute } from "vue-router";
import AppLayout from "../layouts/AppLayout.vue";
import MapContainer from "../components/MapContainer.vue";
import { getPlanDetail } from "../api/plans";

const route = useRoute();

const loading = ref(false);
const errorMessage = ref("");
const detail = ref(null);
const brokenImages = reactive({});

const pageTitle = computed(() => detail.value?.title || "计划详情");

const mapMarkers = computed(() => {
  if (!detail.value) {
    return [];
  }

  const markers = [];

  const center = detail.value.hotel_areas?.recommended_center;
  if (hasCoordinate(center?.longitude, center?.latitude)) {
    markers.push({
      longitude: center.longitude,
      latitude: center.latitude,
      title: "推荐住宿中心"
    });
  }

  for (const hotel of detail.value.hotel_areas?.nearby_hotels || []) {
    if (hasCoordinate(hotel.location?.longitude, hotel.location?.latitude)) {
      markers.push({
        longitude: hotel.location.longitude,
        latitude: hotel.location.latitude,
        title: hotel.name || "酒店"
      });
    }
  }

  for (const day of detail.value.plan?.days || []) {
    for (const activity of day.activities || []) {
      if (hasCoordinate(activity.longitude, activity.latitude)) {
        markers.push({
          longitude: activity.longitude,
          latitude: activity.latitude,
          title: activity.name || `Day ${day.day} 活动`
        });
      }
    }
  }

  return markers;
});

onMounted(() => {
  fetchDetail();
});

async function fetchDetail() {
  const id = route.params.id;
  if (!id) {
    errorMessage.value = "缺少计划 ID。";
    return;
  }

  loading.value = true;
  errorMessage.value = "";
  try {
    const response = await getPlanDetail(id);
    detail.value = response.data;
  } catch (error) {
    errorMessage.value = error?.response?.data?.error || "获取计划详情失败，请稍后重试。";
  } finally {
    loading.value = false;
  }
}

function hasCoordinate(longitude, latitude) {
  return typeof longitude === "number" && typeof latitude === "number";
}

function markBrokenImage(url) {
  if (!url) {
    return;
  }
  brokenImages[url] = true;
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

function formatDistance(distance) {
  if (!distance && distance !== 0) {
    return "未知";
  }
  if (distance >= 1000) {
    return `${(distance / 1000).toFixed(1)} km`;
  }
  return `${distance} m`;
}

function formatCoordinate(value) {
  if (typeof value !== "number") {
    return "未知";
  }
  return value.toFixed(6);
}
</script>
