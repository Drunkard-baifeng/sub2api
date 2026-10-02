<template>
  <!-- Custom Home Content: Full Page Mode -->
  <div v-if="hasHomeContent" class="min-h-screen">
    <!-- iframe mode -->
    <iframe
      v-if="isHomeContentUrl"
      :src="homeContent.trim()"
      class="h-screen w-full border-0"
      allowfullscreen
    ></iframe>
    <!-- HTML mode - SECURITY: homeContent is admin-only setting, XSS risk is acceptable -->
    <div v-else v-html="homeContent"></div>
  </div>

  <!-- Compact Home Page -->
  <div
    v-else-if="compactHomeEnabled"
    data-testid="compact-home"
    class="flex min-h-screen flex-col bg-gray-50 text-gray-900 dark:bg-dark-950 dark:text-white"
  >
    <header class="border-b border-gray-200 px-4 py-4 sm:px-6 dark:border-dark-800">
      <nav class="mx-auto flex max-w-5xl flex-wrap items-center justify-between gap-3 sm:gap-4">
        <div class="flex min-w-0 flex-1 items-center gap-3">
          <img
            :src="siteLogo || '/logo.svg'"
            alt="Logo"
            class="h-9 w-9 shrink-0 rounded-lg object-contain"
          />
          <span class="min-w-0 truncate text-base font-semibold">{{ siteName }}</span>
        </div>
        <div class="flex max-w-full shrink-0 flex-wrap items-center justify-end gap-2">
          <LocaleSwitcher />
          <a
            v-if="docUrl"
            :href="docUrl"
            target="_blank"
            rel="noopener noreferrer"
            class="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg text-gray-500 hover:bg-gray-100 dark:text-dark-400 dark:hover:bg-dark-800"
            :title="t('home.viewDocs')"
          >
            <Icon name="book" size="md" />
          </a>
          <router-link
            v-if="showModelPlazaEntry"
            to="/model-plaza"
            class="flex h-10 shrink-0 items-center gap-1.5 rounded-lg px-2.5 text-sm font-medium text-gray-500 hover:bg-gray-100 hover:text-gray-700 dark:text-dark-400 dark:hover:bg-dark-800 dark:hover:text-white"
            :title="t('nav.modelPlaza')"
          >
            <Icon name="grid" size="md" />
            <span class="hidden sm:inline">{{ t('nav.modelPlaza') }}</span>
          </router-link>
          <button
            class="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg text-gray-500 hover:bg-gray-100 dark:text-dark-400 dark:hover:bg-dark-800"
            :title="isDark ? t('home.switchToLight') : t('home.switchToDark')"
            @click="toggleTheme"
          >
            <Icon v-if="isDark" name="sun" size="md" />
            <Icon v-else name="moon" size="md" />
          </button>
          <router-link
            :to="isAuthenticated ? dashboardPath : '/login'"
            class="inline-flex min-h-10 shrink-0 items-center justify-center rounded-lg bg-gray-900 px-4 py-2 text-sm font-medium text-white hover:bg-gray-800 dark:bg-white dark:text-gray-900 dark:hover:bg-gray-200"
          >
            {{ isAuthenticated ? t('home.dashboard') : t('home.login') }}
          </router-link>
        </div>
      </nav>
    </header>

    <main class="flex min-w-0 flex-1 items-center justify-center px-4 py-16 sm:px-6">
      <div class="min-w-0 max-w-2xl text-center">
        <img
          :src="siteLogo || '/logo.svg'"
          alt="Logo"
          class="mx-auto mb-6 h-20 w-20 rounded-2xl object-contain"
        />
        <h1 class="[overflow-wrap:anywhere] text-3xl font-bold md:text-4xl">{{ siteName }}</h1>
        <p class="mt-4 whitespace-pre-wrap [overflow-wrap:anywhere] text-base text-gray-600 dark:text-dark-300">{{ siteSubtitle }}</p>
        <router-link
          :to="isAuthenticated ? dashboardPath : '/login'"
          class="mt-8 inline-flex min-h-10 items-center justify-center rounded-lg bg-primary-600 px-5 py-2.5 text-sm font-medium text-white hover:bg-primary-700"
        >
          {{ isAuthenticated ? t('home.goToDashboard') : t('home.login') }}
        </router-link>
      </div>
    </main>

    <footer class="min-w-0 border-t border-gray-200 px-4 py-5 text-center text-sm text-gray-500 [overflow-wrap:anywhere] sm:px-6 dark:border-dark-800 dark:text-dark-400">
      &copy; {{ currentYear }} {{ siteName }}
    </footer>
  </div>

  <!-- Default Home Page -->
  <div v-else class="home-page" data-testid="default-home">
    <div class="home-glow home-glow-left" aria-hidden="true"></div>
    <div class="home-glow home-glow-right" aria-hidden="true"></div>
    <div class="home-grid" aria-hidden="true"></div>

    <header class="home-header">
      <router-link to="/" class="home-brand" :aria-label="t('home.glass.homeLink')">
        <span class="home-brand-mark">
          <img :src="siteLogo || '/jisu-ai-logo.svg'" :alt="siteName" />
        </span>
        <span class="home-brand-name">{{ siteName }}</span>
      </router-link>

      <nav class="home-nav" :aria-label="t('home.glass.navigation')">
        <a v-if="docUrl" :href="docUrl" target="_blank" rel="noopener noreferrer" class="home-nav-link">
          <Icon name="book" size="sm" />
          <span>{{ t('home.viewDocs') }}</span>
        </a>
        <router-link v-if="showModelPlazaEntry" to="/model-plaza" class="home-nav-link">
          <Icon name="grid" size="sm" />
          <span>{{ t('nav.modelPlaza') }}</span>
        </router-link>
        <LocaleSwitcher />
        <button class="home-icon-button" :title="isDark ? t('home.switchToLight') : t('home.switchToDark')" @click="toggleTheme">
          <Icon v-if="isDark" name="sun" size="sm" />
          <Icon v-else name="moon" size="sm" />
        </button>
        <router-link :to="isAuthenticated ? dashboardPath : '/login'" class="home-login-button">
          {{ isAuthenticated ? t('home.dashboard') : t('home.login') }}
          <Icon name="arrowRight" size="sm" />
        </router-link>
      </nav>
    </header>

    <main class="home-main">
      <section class="home-hero">
        <div class="home-copy">
          <div class="home-eyebrow"><span class="home-pulse"></span>{{ t('home.glass.eyebrow') }}</div>
          <h1>{{ t('home.glass.title') }}<br /><span>{{ t('home.glass.titleAccent') }}</span></h1>
          <p class="home-subtitle">{{ t('home.glass.description') }}</p>
          <div class="home-actions">
            <router-link :to="isAuthenticated ? dashboardPath : '/login'" class="home-primary-button">
              {{ isAuthenticated ? t('home.goToDashboard') : t('home.getStarted') }}
              <Icon name="arrowRight" size="sm" />
            </router-link>
            <router-link v-if="showModelPlazaEntry" to="/model-plaza" class="home-secondary-button">
              {{ t('nav.modelPlaza') }}<Icon name="grid" size="sm" />
            </router-link>
          </div>
          <div class="hero-tools" aria-hidden="true">
            <span><Icon name="chat" size="sm" />{{ t('home.glass.chat') }}</span>
            <span><Icon name="cpu" size="sm" />{{ t('home.glass.code') }}</span>
            <span><Icon name="chart" size="sm" />{{ t('home.glass.analysis') }}</span>
          </div>
        </div>

        <div class="gateway-preview">
          <div class="preview-top"><span><Icon name="cpu" size="sm" />{{ t('home.glass.gateway') }}</span><small>{{ t('home.glass.preview') }}</small></div>
          <div class="preview-flow" aria-hidden="true">
            <span class="flow-app"><Icon name="grid" size="md" /></span>
            <span class="flow-wire"></span>
            <span class="flow-hub"><img src="/jisu-ai-logo.svg" alt="" /></span>
            <span class="flow-wire"></span>
            <span class="flow-model" :class="'model-' + selectedModel">{{ modelSymbols[selectedModel] }}</span>
          </div>
          <div class="model-tabs" :aria-label="t('home.glass.chooseModel')">
            <button v-for="model in modelOptions" :key="model" type="button" :aria-pressed="selectedModel === model" :class="['model-tab', 'model-' + model, { active: selectedModel === model }]" @click="selectedModel = model">
              <span>{{ modelSymbols[model] }}</span>{{ model }}
            </button>
          </div>
          <div class="preview-chat" aria-live="polite">
            <div class="chat-question"><Icon name="chat" size="sm" />{{ modelCopy[selectedModel].prompt }}</div>
            <div class="chat-answer">
              <span class="answer-spark">✦</span>
              <div>
                <strong>{{ modelCopy[selectedModel].answer }}</strong>
                <div v-if="selectedModel === 'Claude'" class="mini-code" aria-hidden="true"><span>function</span> connect() &#123;<br />&nbsp; <span>return</span> ideas.map(create)<br />&#125;</div>
                <div v-else-if="selectedModel === 'GPT'" class="writing-preview" aria-hidden="true"><i></i><i></i><i></i></div>
                <div v-else class="analysis-preview" aria-hidden="true"><i v-for="(height, index) in chartHeights" :key="index" :style="{ height: height + '%' }"></i></div>
              </div>
            </div>
          </div>
          <div class="preview-bottom"><span>{{ t('home.glass.unifiedEntry') }}</span><span>{{ t('home.glass.exampleOnly') }}</span></div>
        </div>
      </section>

      <section class="model-shelf" :aria-label="t('home.glass.ecosystem')">
        <div class="shelf-heading"><span class="section-label">{{ t('home.glass.ecosystem') }}</span><p>{{ t('home.glass.modelNote') }}</p></div>
        <div class="shelf-models">
          <button v-for="model in modelOptions" :key="model" type="button" :class="['shelf-model', 'model-' + model, { selected: selectedModel === model }]" :aria-pressed="selectedModel === model" @click="selectedModel = model">
            <b>{{ modelSymbols[model] }}</b><span><strong>{{ model }}</strong><small>{{ modelCopy[model].role }}</small></span><Icon name="arrowRight" size="sm" />
          </button>
        </div>
      </section>

      <section class="home-workspace" :aria-label="t('home.glass.capabilities')">
        <article class="workspace-card setup-card">
          <div class="card-heading"><span class="feature-icon"><Icon name="link" size="md" /></span><div><h2>{{ t('home.glass.setupTitle') }}</h2><p>{{ t('home.glass.setupDescription') }}</p></div></div>
          <ol class="setup-steps">
            <li><span>01</span><Icon name="key" size="md" /><strong>{{ t('home.glass.stepKey') }}</strong></li>
            <li><span>02</span><Icon name="link" size="md" /><strong>{{ t('home.glass.stepConnect') }}</strong></li>
            <li><span>03</span><Icon name="chart" size="md" /><strong>{{ t('home.glass.stepUsage') }}</strong></li>
          </ol>
          <a v-if="docUrl" class="card-link" :href="docUrl" target="_blank" rel="noopener noreferrer">{{ t('home.viewDocs') }}<Icon name="arrowRight" size="sm" /></a>
          <router-link v-else class="card-link" :to="isAuthenticated ? dashboardPath : '/login'">{{ isAuthenticated ? t('home.goToDashboard') : t('home.getStarted') }}<Icon name="arrowRight" size="sm" /></router-link>
        </article>
        <article class="workspace-card usage-card">
          <div class="card-heading"><span class="feature-icon"><Icon name="chart" size="md" /></span><div><h2>{{ t('home.glass.usageTitle') }}</h2><p>{{ t('home.glass.usageDescription') }}</p></div></div>
          <div class="usage-visual" aria-hidden="true">
            <div class="usage-bars"><i v-for="(height, index) in usageHeights" :key="index" :style="{ height: height + '%' }"></i></div>
            <div class="usage-legend"><span>Claude</span><span>GPT</span><span>Gemini</span></div>
          </div>
          <small class="visual-note">{{ t('home.glass.usagePreview') }}</small>
        </article>
      </section>
    </main>
    <footer class="home-footer">
      <span>&copy; {{ currentYear }} {{ siteName }}</span>
    </footer>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAuthStore, useAppStore } from '@/stores'
import LocaleSwitcher from '@/components/common/LocaleSwitcher.vue'
import Icon from '@/components/icons/Icon.vue'
import { sanitizeUrl } from '@/utils/url'
import { FeatureFlags, isFeatureFlagEnabled } from '@/utils/featureFlags'

const { t } = useI18n()

const authStore = useAuthStore()
const appStore = useAppStore()

// Site settings - directly from appStore (already initialized from injected config)
const siteName = computed(() => appStore.cachedPublicSettings?.site_name || appStore.siteName || 'Sub2API')
const siteLogo = computed(() => sanitizeUrl(appStore.cachedPublicSettings?.site_logo || appStore.siteLogo || '', { allowRelative: true, allowDataUrl: true }))
const siteSubtitle = computed(() => appStore.cachedPublicSettings?.site_subtitle || 'AI API Gateway Platform')
const docUrl = computed(() => sanitizeUrl(appStore.cachedPublicSettings?.doc_url || appStore.docUrl || ''))
const homeContent = computed(() => appStore.cachedPublicSettings?.home_content || '')
const hasHomeContent = computed(() => homeContent.value.trim().length > 0)
const compactHomeEnabled = computed(() => appStore.cachedPublicSettings?.compact_home_enabled === true)
const modelPlazaEnabled = computed(() => isFeatureFlagEnabled(FeatureFlags.modelPlaza))

// Check if homeContent is a URL (for iframe display)
const isHomeContentUrl = computed(() => {
  const content = homeContent.value.trim()
  return content.startsWith('http://') || content.startsWith('https://')
})

const modelOptions = ['Claude', 'GPT', 'Gemini'] as const
const selectedModel = ref<(typeof modelOptions)[number]>('Claude')
const modelSymbols = { Claude: '✳', GPT: '◈', Gemini: '✦' }
const modelCopy = computed(() => ({
  Claude: {
    prompt: t('home.glass.promptClaude'),
    answer: t('home.glass.answerClaude'),
    role: t('home.glass.roleClaude'),
  },
  GPT: {
    prompt: t('home.glass.promptGPT'),
    answer: t('home.glass.answerGPT'),
    role: t('home.glass.roleGPT'),
  },
  Gemini: {
    prompt: t('home.glass.promptGemini'),
    answer: t('home.glass.answerGemini'),
    role: t('home.glass.roleGemini'),
  },
}))
const chartHeights = [32, 50, 42, 68, 58, 82, 72, 94]
const usageHeights = [24, 38, 32, 50, 42, 64, 53, 72, 58, 82, 68, 92, 77, 87, 72, 96]
// Theme
const isDark = ref(document.documentElement.classList.contains('dark'))


// Auth state
const isAuthenticated = computed(() => authStore.isAuthenticated)
const modelPlazaRequiresAuth = computed(
  () => appStore.cachedPublicSettings?.model_plaza_require_auth === true,
)
const showModelPlazaEntry = computed(
  () => modelPlazaEnabled.value && (isAuthenticated.value || !modelPlazaRequiresAuth.value),
)
const isAdmin = computed(() => authStore.isAdmin)
const dashboardPath = computed(() => isAdmin.value ? '/admin/dashboard' : '/dashboard')

// Current year for footer
const currentYear = computed(() => new Date().getFullYear())

// Toggle theme
function toggleTheme() {
  isDark.value = !isDark.value
  document.documentElement.classList.toggle('dark', isDark.value)
  localStorage.setItem('theme', isDark.value ? 'dark' : 'light')
}

// Initialize theme
function initTheme() {
  const savedTheme = localStorage.getItem('theme')
  if (
    savedTheme === 'dark' ||
    (!savedTheme && window.matchMedia('(prefers-color-scheme: dark)').matches)
  ) {
    isDark.value = true
    document.documentElement.classList.add('dark')
  }
}

onMounted(() => {
  initTheme()

  // Check auth state
  authStore.checkAuth()

  // Ensure public settings are loaded (will use cache if already loaded from injected config)
  if (!appStore.publicSettingsLoaded) {
    appStore.fetchPublicSettings()
  }
})
</script>

<style scoped>
.home-page {
  --home-ink: #163d43; --home-muted: #59757e; --home-glass: #ffffff70; --home-border: #ffffffb8; --home-inset: #ffffff5c; --home-accent: #138b7d;
  position: relative; isolation: isolate; display: flex; min-height: 100svh; flex-direction: column; overflow: hidden; background: #eef5f6; color: var(--home-ink);
}
.home-glow { position: absolute; z-index: -1; width: 1100px; height: 900px; border-radius: 50%; filter: blur(90px); pointer-events: none; }
.home-glow-left { top: -380px; left: -280px; background: radial-gradient(circle, #a6e8d8, transparent 65%); opacity: .65; }
.home-glow-right { top: -170px; right: -240px; background: radial-gradient(circle, #b8c9ed, #d7dfed 40%, transparent 65%); opacity: .7; }
.home-grid { position: absolute; z-index: -1; inset: 0; pointer-events: none; background-image: radial-gradient(#54959920 .8px, transparent .8px); background-size: 24px 24px; mask-image: linear-gradient(transparent, #0005 35%, transparent 85%); }
.home-header, .home-main, .home-footer { width: min(100% - 72px, 1400px); margin: 0 auto; }
.home-header { position: relative; z-index: 10; display: flex; align-items: center; justify-content: space-between; gap: 28px; padding: 28px 0; }
.home-brand { display: inline-flex; align-items: center; gap: 12px; min-width: 0; }
.home-brand-mark { flex-shrink: 0; width: 44px; height: 44px; border-radius: 13px; overflow: hidden; }
.home-brand-mark img { width: 100%; height: 100%; object-fit: contain; }
.home-brand-name { font-size: 23px; font-weight: 700; letter-spacing: -.5px; overflow-wrap: anywhere; }
.home-nav { display: flex; align-items: center; justify-content: flex-end; gap: 18px; flex-wrap: wrap; }
.home-nav-link { display: inline-flex; align-items: center; gap: 7px; font-size: 13px; color: var(--home-muted); }
.home-nav-link:hover { color: var(--home-accent); }
.home-icon-button { display: grid; place-items: center; flex-shrink: 0; width: 38px; height: 38px; border: 1px solid var(--home-border); border-radius: 12px; color: var(--home-muted); background: var(--home-glass); backdrop-filter: blur(16px); }
.home-login-button { display: inline-flex; align-items: center; justify-content: center; gap: 10px; min-height: 38px; padding: 0 16px; border-radius: 12px; background: #173b41; color: #fff; font-size: 13px; font-weight: 600; }
.home-main { display: flex; flex: 1; flex-direction: column; justify-content: center; gap: 26px; padding: 24px 0 40px; }
.home-hero { display: grid; grid-template-columns: .85fr 1.15fr; align-items: center; gap: clamp(40px, 5vw, 90px); padding: 14px 0 24px; }
.home-copy { min-width: 0; padding-left: 8px; }
.home-eyebrow { display: inline-flex; align-items: center; gap: 9px; color: var(--home-accent); font-size: 13px; letter-spacing: 1px; margin-bottom: 22px; }
.home-pulse { width: 6px; height: 6px; border-radius: 50%; background: #19b9a0; box-shadow: 0 0 0 4px #1bb99f12; }
.home-copy h1 { font-size: clamp(30px, 2.5vw, 42px); line-height: 1.5; font-weight: 600; letter-spacing: -1.5px; }
.home-copy h1 > span { color: var(--home-accent); }
.home-subtitle { margin-top: 18px; max-width: 390px; color: var(--home-muted); font-size: 14px; line-height: 1.9; }
.home-actions { display: flex; align-items: center; flex-wrap: wrap; gap: 12px; margin-top: 26px; }
.home-primary-button, .home-secondary-button { display: inline-flex; align-items: center; justify-content: center; gap: 13px; min-height: 46px; padding: 0 22px; border-radius: 13px; font-size: 14px; font-weight: 600; transition: transform .2s; }
.home-primary-button { background: linear-gradient(115deg, #0d9e94, #108c89); color: #fff; box-shadow: 0 8px 22px #0b918620, inset 0 1px 0 #ffffff30; }
.home-secondary-button { background: var(--home-glass); border: 1px solid var(--home-border); }
.home-primary-button:hover, .home-secondary-button:hover { transform: translateY(-2px); }
.hero-tools { display: flex; gap: 22px; margin-top: 32px; color: var(--home-muted); font-size: 12px; }
.hero-tools span { display: flex; align-items: center; gap: 7px; }
.gateway-preview, .workspace-card { border: 1px solid var(--home-border); background: var(--home-glass); backdrop-filter: blur(28px); box-shadow: inset 0 1px 0 #ffffff80, 0 18px 60px #30566d09; border-radius: 26px; }
.gateway-preview { padding: 23px 28px 18px; min-width: 0; position: relative; }
.preview-top, .preview-bottom { display: flex; align-items: center; justify-content: space-between; gap: 12px; }
.preview-top > span { display: flex; align-items: center; gap: 8px; font-weight: 600; font-size: 13px; }
.preview-top small { font-size: 10px; color: var(--home-muted); padding: 4px 8px; border: 1px solid #789ba82b; border-radius: 6px; }
.preview-flow { display: flex; justify-content: center; align-items: center; padding: 18px 20px; }
.flow-app, .flow-model { display: grid; place-items: center; width: 44px; height: 44px; flex-shrink: 0; border: 1px solid var(--home-border); background: var(--home-inset); border-radius: 14px; color: var(--home-accent); font-size: 27px; }
.flow-wire { width: 90px; height: 1px; background: linear-gradient(90deg, #4bb8a822, #4bb8a899, #4bb8a822); position: relative; }
.flow-wire::after { content: ''; position: absolute; top: -2px; left: 30%; width: 5px; height: 5px; border-radius: 50%; background: #69cab8; animation: route-pulse 3s ease-in-out infinite; }
.flow-hub { display: grid; place-items: center; width: 68px; height: 68px; padding: 9px; border-radius: 22px; background: #c5eadc55; border: 1px solid var(--home-border); box-shadow: 0 0 0 7px #6bc2af0c; flex-shrink: 0; }
.flow-hub img { width: 100%; height: 100%; }
.model-tabs { display: grid; grid-template-columns: repeat(3, 1fr); gap: 7px; padding: 5px; background: #7194a00a; border-radius: 13px; }
.model-tab { display: flex; align-items: center; justify-content: center; gap: 7px; padding: 9px 5px; border-radius: 9px; border: 1px solid transparent; color: var(--home-muted); font-size: 12px; transition: background .2s; }
.model-tab span { font-size: 19px; }
.model-tab.active { border-color: var(--home-border); background: var(--home-inset); color: var(--home-ink); box-shadow: 0 2px 6px #28445406; }
.model-Claude > b, .model-Claude > span:first-child, .flow-model.model-Claude { color: #b2795c; }
.model-GPT > b, .model-GPT > span:first-child, .flow-model.model-GPT { color: #259784; }
.model-Gemini > b, .model-Gemini > span:first-child, .flow-model.model-Gemini { color: #7b82cb; }
.preview-chat { padding: 20px 5px 16px; min-height: 162px; }
.chat-question { display: flex; align-items: center; gap: 10px; font-size: 12px; color: var(--home-muted); }
.chat-answer { display: grid; grid-template-columns: 24px 1fr; gap: 10px; margin-top: 16px; }
.answer-spark { color: var(--home-accent); font-size: 22px; line-height: 1; }
.chat-answer strong { display: block; font-size: 12px; font-weight: 500; }
.mini-code { margin-top: 9px; font-family: ui-monospace, monospace; font-size: 11px; line-height: 1.7; color: var(--home-muted); }
.mini-code span { color: var(--home-accent); }
.writing-preview { display: grid; gap: 9px; margin-top: 16px; }
.writing-preview i { height: 6px; width: 92%; border-radius: 4px; background: #77b8ad26; }
.writing-preview i:nth-child(2) { width: 78%; }.writing-preview i:last-child { width: 54%; }
.analysis-preview { display: flex; height: 65px; align-items: flex-end; gap: 12px; margin-top: 10px; }
.analysis-preview i { width: 9%; border-radius: 4px 4px 0 0; background: linear-gradient(#85a3d8aa, #85a3d826); }
.preview-bottom { padding-top: 12px; border-top: 1px solid #6e9ba31f; font-size: 10px; color: var(--home-muted); }
.model-shelf { display: grid; grid-template-columns: .75fr 2fr; align-items: center; gap: 26px; padding: 22px 0; border-top: 1px solid #71979e26; border-bottom: 1px solid #71979e26; }
.section-label { font-size: 15px; font-weight: 600; }
.shelf-heading p { font-size: 11px; color: var(--home-muted); margin-top: 6px; line-height: 1.6; }
.shelf-models { display: grid; grid-template-columns: repeat(3, 1fr); gap: 12px; }
.shelf-model { display: flex; align-items: center; gap: 12px; min-width: 0; text-align: left; border: 1px solid transparent; border-radius: 15px; padding: 13px; transition: background .2s; }
.shelf-model:hover, .shelf-model.selected { background: var(--home-glass); border-color: var(--home-border); }
.shelf-model > b { display: grid; place-items: center; width: 36px; height: 36px; background: var(--home-inset); border-radius: 11px; font-size: 24px; font-weight: 400; flex-shrink: 0; }
.shelf-model strong { display: block; font-size: 14px; font-weight: 600; }
.shelf-model small { display: block; margin-top: 3px; font-size: 10px; color: var(--home-muted); }
.shelf-model > svg { margin-left: auto; flex-shrink: 0; opacity: .4; }
.home-workspace { display: grid; grid-template-columns: 1.15fr 1fr; gap: 24px; }
.workspace-card { padding: 24px 28px 20px; min-width: 0; }
.card-heading { display: flex; align-items: center; gap: 14px; }
.feature-icon { display: grid; place-items: center; width: 38px; height: 38px; border-radius: 12px; background: #68bda414; color: var(--home-accent); flex-shrink: 0; }
.card-heading h2 { font-size: 16px; font-weight: 600; }
.card-heading p { margin-top: 4px; font-size: 12px; line-height: 1.6; color: var(--home-muted); }
.setup-steps { display: grid; grid-template-columns: repeat(3, 1fr); gap: 12px; margin-top: 21px; padding: 0; list-style: none; }
.setup-steps li { display: grid; grid-template-columns: 1fr auto; align-items: center; gap: 8px; background: var(--home-inset); border: 1px solid var(--home-border); border-radius: 13px; padding: 12px 14px; }
.setup-steps li > span { font-size: 10px; color: var(--home-muted); font-family: ui-monospace, monospace; }
.setup-steps svg { color: var(--home-accent); }
.setup-steps strong { grid-column: 1 / -1; font-size: 12px; font-weight: 500; }
.card-link { display: flex; align-items: center; justify-content: flex-end; gap: 8px; margin-top: 16px; font-size: 11px; color: var(--home-accent); }
.usage-visual { display: flex; align-items: center; gap: 24px; padding: 20px 2px 0; }
.usage-bars { display: flex; align-items: flex-end; flex: 1; height: 72px; gap: 6px; border-bottom: 1px solid #7aa6a333; background: repeating-linear-gradient(to top, transparent 0, transparent 23px, #719b9a12 23px, #719b9a12 24px); }
.usage-bars i { flex: 1; background: linear-gradient(#60bba99c, #92d0c429); border-radius: 4px 4px 0 0; }
.usage-bars i:nth-child(3n) { background: linear-gradient(#859dce99, #abc5e326); }
.usage-legend { display: grid; gap: 10px; color: var(--home-muted); font-size: 10px; }
.usage-legend span::before { content: ''; display: inline-block; width: 5px; height: 5px; margin-right: 7px; border-radius: 2px; background: #6fbaa9; }
.usage-legend span:last-child::before { background: #859dce; }
.visual-note { display: block; text-align: right; font-size: 10px; color: var(--home-muted); margin-top: 12px; }
.home-footer { padding: 18px 0 24px; text-align: center; color: var(--home-muted); font-size: 12px; }
.home-page :is(a, button):focus-visible { outline: 2px solid var(--home-accent); outline-offset: 4px; border-radius: 9px; }
.dark .home-page { --home-ink: #e0eeee; --home-muted: #9bb1b9; --home-glass: #243b4b65; --home-border: #b9dfeb1f; --home-inset: #8cb8c00a; --home-accent: #74d8c1; background: #0e191f; }
.dark .home-glow-left { background: radial-gradient(circle, #1a514d, transparent 65%); }
.dark .home-glow-right { background: radial-gradient(circle, #263b55, transparent 65%); }
.dark .home-login-button { background: #b1e7dc; color: #153b3b; }
.dark .gateway-preview, .dark .workspace-card { box-shadow: inset 0 1px 0 #ffffff09, 0 18px 60px #00000014; }
.dark .flow-hub { background: #80e0bf0c; }
.dark .model-Claude > b, .dark .model-Claude > span:first-child, .dark .flow-model.model-Claude { color: #dfab8d; }
.dark .model-GPT > b, .dark .model-GPT > span:first-child, .dark .flow-model.model-GPT { color: #7ad8bc; }
.dark .model-Gemini > b, .dark .model-Gemini > span:first-child, .dark .flow-model.model-Gemini { color: #b2b8f5; }
@keyframes route-pulse { 0%, 100% { left: 15%; opacity: .3; } 60% { left: 80%; opacity: 1; } }
@media (min-width: 1800px) {
  .home-header, .home-main, .home-footer { width: min(80%, 1920px); }
  .home-main { gap: 38px; padding-top: 42px; padding-bottom: 42px; }
  .home-hero { gap: 8%; padding-bottom: 28px; }
  .home-copy h1 { font-size: 44px; }
  .home-subtitle { font-size: 16px; max-width: 440px; }
  .gateway-preview { padding: 28px 38px 22px; }
  .preview-flow { padding: 22px; }
  .preview-chat { min-height: 184px; padding-top: 24px; }
  .chat-question, .chat-answer strong { font-size: 14px; }
  .mini-code { font-size: 13px; }
  .model-shelf { padding: 28px 0; }
  .workspace-card { padding: 30px 34px 24px; }
  .setup-steps { margin-top: 28px; }
  .setup-steps li { padding: 18px; }
  .usage-bars { height: 94px; }
}
@media (max-width: 1050px) {
  .home-hero { gap: 28px; grid-template-columns: 1fr 1.1fr; }
  .home-copy h1 { font-size: 33px; }
  .gateway-preview { padding: 20px; }
  .model-shelf { grid-template-columns: 1fr; gap: 12px; }
  .home-workspace { gap: 18px; }
  .workspace-card { padding: 22px; }
}
@media (max-width: 760px) {
  .home-header, .home-main, .home-footer { width: calc(100% - 40px); }
  .home-header { flex-wrap: wrap; gap: 18px; padding: 22px 0; }
  .home-brand-name { font-size: 22px; }
  .home-nav { gap: 12px; flex: 1; }
  .home-nav-link { font-size: 12px; }
  .home-main { gap: 24px; padding: 18px 0 28px; }
  .home-hero { grid-template-columns: 1fr; gap: 32px; padding: 10px 0 0; }
  .home-copy { text-align: center; padding: 0; }
  .home-copy h1 { font-size: 32px; }
  .home-subtitle { margin: 15px auto 0; font-size: 13px; }
  .home-actions, .hero-tools { justify-content: center; }
  .hero-tools { margin-top: 22px; }
  .gateway-preview { padding: 20px 18px; }
  .shelf-heading { text-align: center; }
  .shelf-models { gap: 4px; }
  .shelf-model { flex-direction: column; padding: 12px 4px; gap: 7px; text-align: center; }
  .shelf-model > svg { display: none; }
  .shelf-model strong { font-size: 12px; }
  .shelf-model small { font-size: 9px; }
  .home-workspace { grid-template-columns: 1fr; gap: 18px; }
  .setup-steps { gap: 7px; }
  .setup-steps li { padding: 11px; }
  .setup-steps strong { font-size: 11px; }
}
@media (prefers-reduced-motion: reduce) {
  .flow-wire::after { animation: none; }
  .home-primary-button, .home-secondary-button, .model-tab, .shelf-model { transition: none; }
}
</style>
