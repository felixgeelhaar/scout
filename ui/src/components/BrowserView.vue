<script setup lang="ts">
defineProps<{
  url: string;
  title: string;
  screenshot: string;
  elementCount: number;
  activeTool: string;
  isLoading: boolean;
}>();
</script>

<template>
  <div class="flex flex-col h-full bg-zinc-950">
    <!-- Chrome-style toolbar -->
    <div
      class="flex items-center gap-2.5 px-3 py-2 bg-zinc-900/80 border-b border-zinc-800/60"
    >
      <!-- Traffic lights -->
      <div class="flex gap-1.5 mr-1">
        <span class="w-2.5 h-2.5 rounded-full bg-[#ff5f57]" />
        <span class="w-2.5 h-2.5 rounded-full bg-[#febc2e]" />
        <span class="w-2.5 h-2.5 rounded-full bg-[#28c840]" />
      </div>

      <!-- URL bar -->
      <div
        class="flex-1 flex items-center gap-2 px-3 py-1 rounded-lg bg-zinc-800/80 border border-zinc-700/40"
      >
        <!-- Loading spinner or lock icon -->
        <svg
          v-if="isLoading"
          class="w-3 h-3 text-blue-400 shrink-0 animate-spin"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2.5"
        >
          <path d="M12 2v4m0 12v4m-7.07-3.93l2.83-2.83m8.48-8.48l2.83-2.83M2 12h4m12 0h4m-3.93 7.07l-2.83-2.83M7.76 7.76L4.93 4.93" />
        </svg>
        <svg
          v-else-if="url && url !== 'about:blank'"
          class="w-3 h-3 text-emerald-500 shrink-0"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2.5"
        >
          <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z" />
        </svg>
        <svg
          v-else
          class="w-3 h-3 text-zinc-500 shrink-0"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
        >
          <circle cx="12" cy="12" r="10" />
        </svg>
        <span class="text-xs font-mono text-zinc-400 truncate">
          {{ url || "about:blank" }}
        </span>
      </div>

      <!-- Badges -->
      <div
        v-if="elementCount"
        class="text-[10px] text-zinc-500 bg-zinc-800/60 px-2 py-0.5 rounded-md tabular-nums shrink-0"
      >
        {{ elementCount }} elements
      </div>
      <div
        v-if="activeTool"
        class="flex items-center gap-1.5 text-[10px] text-blue-400 bg-blue-500/10 px-2 py-0.5 rounded-md shrink-0"
      >
        <span class="relative flex h-1.5 w-1.5">
          <span
            class="animate-ping absolute inline-flex h-full w-full rounded-full bg-blue-400 opacity-75"
          />
          <span
            class="relative inline-flex rounded-full h-1.5 w-1.5 bg-blue-500"
          />
        </span>
        {{ activeTool }}
      </div>
    </div>

    <!-- Loading bar -->
    <div v-if="isLoading" class="h-0.5 bg-zinc-900 overflow-hidden">
      <div class="h-full bg-blue-500 animate-loading-bar" />
    </div>

    <!-- Viewport -->
    <div class="flex-1 overflow-auto bg-zinc-900/40 relative">
      <!-- Screenshot -->
      <img
        v-if="screenshot"
        :src="`data:image/png;base64,${screenshot}`"
        :alt="title || 'Browser screenshot'"
        class="w-full h-auto"
        :class="{ 'opacity-50': isLoading }"
      />

      <!-- Loading shimmer overlay -->
      <div
        v-if="isLoading && screenshot"
        class="absolute inset-0 flex items-center justify-center bg-zinc-950/30"
      >
        <div
          class="flex items-center gap-2 px-4 py-2 rounded-full bg-zinc-900/90 border border-zinc-700/50 shadow-lg"
        >
          <svg
            class="w-4 h-4 text-blue-400 animate-spin"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2.5"
          >
            <path d="M12 2v4m0 12v4m-7.07-3.93l2.83-2.83m8.48-8.48l2.83-2.83M2 12h4m12 0h4m-3.93 7.07l-2.83-2.83M7.76 7.76L4.93 4.93" />
          </svg>
          <span class="text-sm text-zinc-300">Navigating...</span>
        </div>
      </div>

      <!-- Loading shimmer (no screenshot yet) -->
      <div v-if="isLoading && !screenshot" class="p-4 space-y-3 animate-pulse">
        <div class="h-4 bg-zinc-800 rounded w-3/4" />
        <div class="h-4 bg-zinc-800 rounded w-1/2" />
        <div class="h-32 bg-zinc-800 rounded" />
        <div class="h-4 bg-zinc-800 rounded w-5/6" />
        <div class="h-4 bg-zinc-800 rounded w-2/3" />
      </div>

      <!-- Empty state -->
      <div
        v-if="!screenshot && !isLoading"
        class="flex items-center justify-center h-full"
      >
        <div class="text-center space-y-4">
          <div class="relative w-20 h-20 mx-auto">
            <div
              class="absolute inset-0 grid grid-cols-3 grid-rows-3 gap-1.5 opacity-20"
            >
              <span
                v-for="i in 9"
                :key="i"
                class="rounded bg-zinc-600"
              />
            </div>
            <div class="absolute inset-0 flex items-center justify-center">
              <svg
                class="w-8 h-8 text-zinc-600"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="1.5"
              >
                <rect x="2" y="3" width="20" height="14" rx="2" />
                <path d="M8 21h8m-4-4v4" />
              </svg>
            </div>
          </div>
          <div>
            <p class="text-[13px] text-zinc-500 font-medium">
              No page loaded
            </p>
            <p class="text-xs text-zinc-600 mt-1">
              Ask Scout to navigate somewhere
            </p>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
@keyframes loading-bar {
  0% {
    transform: translateX(-100%);
    width: 40%;
  }
  50% {
    width: 60%;
  }
  100% {
    transform: translateX(300%);
    width: 40%;
  }
}
.animate-loading-bar {
  animation: loading-bar 1.5s ease-in-out infinite;
}
</style>
