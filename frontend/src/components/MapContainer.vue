<script setup>
import { onMounted, onUnmounted, watch } from "vue";
import AMapLoader from "@amap/amap-jsapi-loader";

const props = defineProps({
  markers: {
    type: Array,
    default: () => []
  },
  height: {
    type: String,
    default: "480px"
  }
});

let map = null;
let AMapRef = null;
let markerInstances = [];

onMounted(async () => {
  const amapKey = import.meta.env.VITE_AMAP_KEY;
  const amapSecret = import.meta.env.VITE_AMAP_SECRET;

  if (!amapKey) {
    console.error("VITE_AMAP_KEY is not configured.");
    return;
  }

  if (amapSecret) {
    window._AMapSecurityConfig = {
      securityJsCode: amapSecret
    };
  }

  try {
    const AMap = await AMapLoader.load({
      key: amapKey,
      version: "2.0",
      plugins: ["AMap.Scale"]
    });

    AMapRef = AMap;
    map = new AMap.Map("container", {
      viewMode: "3D",
      zoom: 11,
      center: [116.397428, 39.90923]
    });

    renderMarkers();
  } catch (error) {
    console.error("AMap initialization failed:", error);
  }
});

watch(
  () => props.markers,
  () => {
    renderMarkers();
  },
  { deep: true }
);

onUnmounted(() => {
  clearMarkers();
  map?.destroy();
  map = null;
  AMapRef = null;
});

function clearMarkers() {
  if (map && markerInstances.length > 0) {
    map.remove(markerInstances);
  }
  markerInstances = [];
}

function renderMarkers() {
  if (!map || !AMapRef) {
    return;
  }

  clearMarkers();

  const validMarkers = props.markers.filter(
    (item) =>
      typeof item?.longitude === "number" &&
      typeof item?.latitude === "number"
  );

  if (validMarkers.length === 0) {
    return;
  }

  markerInstances = validMarkers.map((item) => {
    return new AMapRef.Marker({
      position: [item.longitude, item.latitude],
      title: item.title || item.name || "位置点"
    });
  });

  map.add(markerInstances);

  if (markerInstances.length === 1) {
    map.setCenter(markerInstances[0].getPosition());
    map.setZoom(13);
    return;
  }

  map.setFitView(markerInstances);
}
</script>

<template>
  <div id="container" :style="{ height }"></div>
</template>

<style scoped>
#container {
  padding: 0;
  margin: 0;
  width: 100%;
  border-radius: 20px;
  overflow: hidden;
}
</style>
