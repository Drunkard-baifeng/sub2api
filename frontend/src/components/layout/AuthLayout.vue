<template>
  <div class="auth-shell">
    <div class="auth-atmosphere" aria-hidden="true">
      <div class="ambient ambient-mint"></div>
      <div class="ambient ambient-blue"></div>
      <div class="ambient ambient-violet"></div>
    </div>

    <header class="auth-header">
      <router-link to="/home" class="header-action home-link" :aria-label="t('home.glass.homeLink')" :title="t('home.glass.homeLink')">
        <Icon name="arrowLeft" size="sm" />
      </router-link>
      <div class="brand">
        <div class="brand-mark">
          <img :src="siteLogo || '/jisu-ai-logo.svg'" :alt="siteName" />
        </div>
        <span>{{ siteName }}</span>
      </div>
      <button
        type="button"
        class="header-action theme-toggle"
        :aria-label="isDark ? '切换至白天模式' : '切换至黑夜模式'"
        :title="isDark ? '切换至白天模式' : '切换至黑夜模式'"
        @click="toggleTheme"
      >
        <Icon :name="isDark ? 'sun' : 'moon'" size="sm" />
      </button>
    </header>

    <main class="auth-main">
      <div class="auth-composition">
        <section class="auth-showcase" aria-label="平台介绍">
          <div class="gateway-art" role="img" aria-label="连接 GPT、Claude 和 Gemini 的 AI 网关示意图">
            <div class="art-halo"></div>
            <div class="orbit orbit-outer"></div>
            <div class="orbit orbit-inner"></div>
            <div class="orb">
              <div class="orb-latitude latitude-one"></div>
              <div class="orb-latitude latitude-two"></div>
              <div class="orb-meridian meridian-one"></div>
              <div class="orb-meridian meridian-two"></div>
              <span class="orb-label">AI<span>无限可能</span></span>
            </div>
            <div class="orbit-spark spark-one"></div>
            <div class="orbit-spark spark-two"></div>
            <div class="model-tile tile-claude"><span class="model-symbol symbol-claude">✳</span>Claude</div>
            <div class="model-tile tile-gpt"><span class="model-symbol symbol-gpt"><Icon name="cpu" size="md" /></span>GPT</div>
            <div class="model-tile tile-gemini"><span class="model-symbol symbol-gemini">✦</span>Gemini</div>
          </div>

          <div class="showcase-caption">
            <h1>连接智能，<span>即刻开始。</span></h1>
            <p>一个入口，轻松连接你需要的 AI 模型。</p>
          </div>
        </section>

        <section class="auth-form-panel" aria-label="账户验证">
          <div class="auth-card">
            <slot />
            <div v-if="$slots.footer" class="auth-switch"><slot name="footer" /></div>
          </div>
        </section>
      </div>
    </main>

    <footer class="auth-copyright">
      &copy; {{ currentYear }} {{ siteName }}
    </footer>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores'
import { sanitizeUrl } from '@/utils/url'
import Icon from '@/components/icons/Icon.vue'

const appStore = useAppStore()
const { t } = useI18n()

const siteName = computed(() => appStore.siteName || 'Sub2API')
const siteLogo = computed(() => sanitizeUrl(appStore.siteLogo || '', { allowRelative: true, allowDataUrl: true }))

const currentYear = computed(() => new Date().getFullYear())
const isDark = ref(false)

function applyTheme(dark: boolean): void {
  document.documentElement.classList.toggle('dark', dark)
  isDark.value = dark
  localStorage.setItem('theme', dark ? 'dark' : 'light')
}

function toggleTheme(): void {
  applyTheme(!isDark.value)
}

onMounted(() => {
  isDark.value = document.documentElement.classList.contains('dark')
  appStore.fetchPublicSettings()
  const savedTheme = localStorage.getItem('theme')
  if (savedTheme === 'dark' || savedTheme === 'light') applyTheme(savedTheme === 'dark')
})
</script>

<style scoped>
.auth-shell {
  --auth-ink: #17313b;
  --auth-muted: #5f7882;
  --auth-border: rgba(255, 255, 255, .8);
  --auth-glass: rgba(255, 255, 255, .42);
  position: relative;
  isolation: isolate;
  display: flex;
  min-height: 100vh;
  min-height: 100svh;
  flex-direction: column;
  overflow: hidden;
  color: var(--auth-ink);
  background: #edf4f6;
}
.auth-atmosphere { position: absolute; inset: 0; z-index: -1; overflow: hidden; pointer-events: none; }
.ambient { position: absolute; border-radius: 50%; filter: blur(65px); }
.ambient-mint { width: 680px; height: 680px; left: calc(50% - 740px); top: calc(50% - 470px); background: radial-gradient(circle, #90e5d5, #b7eadf 45%, transparent 70%); opacity: .7; }
.ambient-blue { width: 620px; height: 620px; left: calc(50% - 140px); top: calc(50% - 150px); background: radial-gradient(circle, #a8c6f7, #d0ddf7 50%, transparent 70%); opacity: .8; }
.ambient-violet { width: 430px; height: 430px; left: calc(50% + 160px); top: calc(50% - 390px); background: #d5cef1; opacity: .48; }
.auth-header { position: relative; z-index: 2; display: grid; grid-template-columns: 44px minmax(0, 1fr) 44px; align-items: center; gap: 24px; width: 100%; max-width: 1440px; margin: 0 auto; padding: 30px 48px; }
.auth-header .home-link { grid-column: 1; }
.auth-header .brand { grid-column: 2; justify-self: center; }
.auth-header .theme-toggle { grid-column: 3; }
.header-action {
  flex-shrink: 0;
  display: grid; place-items: center; width: 44px; height: 44px;
  border: 1px solid var(--auth-border); border-radius: 14px;
  background: var(--auth-glass); color: var(--auth-muted);
  backdrop-filter: blur(16px); transition: background .2s, transform .2s;
}
.header-action:hover { background: rgba(255, 255, 255, .7); transform: translateY(-2px); }
.header-action:focus-visible { outline: 2px solid #0d9488; outline-offset: 4px; }
.auth-main { display: flex; align-items: center; justify-content: center; flex: 1; width: 100%; padding: 24px 32px 64px; }
.auth-composition {
  display: grid; grid-template-columns: 1.08fr 1fr; align-items: center;
  width: 100%; max-width: 1120px; gap: 48px;
}
.auth-showcase { min-width: 0; padding: 24px 8px; }
.brand { display: flex; align-items: center; min-width: 0; gap: 14px; font-size: 28px; font-weight: 700; line-height: 1.25; letter-spacing: -.6px; }
.brand > span { overflow-wrap: anywhere; }
.brand-mark { flex-shrink: 0; width: 52px; height: 52px; overflow: hidden; border-radius: 15px; box-shadow: 0 5px 15px #135b6814; }
.brand-mark img { width: 100%; height: 100%; object-fit: contain; }
.gateway-art { position: relative; width: 100%; height: 360px; margin: 8px 0 0; }
.art-halo { position: absolute; inset: 14% 13%; border-radius: 50%; background: #57cdbd; opacity: .17; filter: blur(36px); }
.orb {
  position: absolute; top: 50%; left: 50%; width: 226px; height: 226px;
  transform: translate(-50%, -50%); border: 1px solid rgba(255, 255, 255, .9); border-radius: 50%;
  overflow: hidden;
  background: radial-gradient(circle at 30% 22%, #ffffffed, transparent 42%),
    radial-gradient(circle at 70% 80%, #a7c8ef99, transparent 55%),
    linear-gradient(135deg, #b7f1e994, #75cdbf55 55%, #f4faffbd);
  box-shadow: inset 8px 8px 22px #ffffffcc, inset -10px -10px 28px #53a9ba44,
    0 24px 60px #33798526, 0 0 0 12px #ffffff14;
  backdrop-filter: blur(8px);
}
.orb::after { content: ''; position: absolute; inset: 9px; border-radius: inherit; border: 1px solid #ffffff4d; }
.orb-label { position: absolute; inset: 0; display: flex; align-items: center; justify-content: center; flex-direction: column; font-size: 68px; font-weight: 600; letter-spacing: -5px; color: #247e80; text-shadow: 0 2px 1px #ffffffa6; }
.orb-label span { margin-top: 0; padding-left: 4px; font-size: 10px; font-weight: 500; letter-spacing: 5px; color: #427e87; text-shadow: none; }
.orb-latitude, .orb-meridian { position: absolute; border: 1px solid #ffffff66; border-radius: 50%; }
.orb-latitude { left: -5%; width: 110%; height: 34%; transform: rotate(-25deg); }
.latitude-one { top: 10%; }.latitude-two { bottom: 10%; }
.orb-meridian { top: -5%; height: 110%; width: 45%; transform: rotate(30deg); }
.meridian-one { left: 10%; }.meridian-two { right: 10%; }
.orbit { position: absolute; top: 50%; left: 50%; border: 1px solid #589d9c30; border-radius: 50%; }
.orbit-outer { width: 92%; height: 61%; transform: translate(-50%, -50%) rotate(-24deg); }
.orbit-inner { width: 82%; height: 84%; transform: translate(-50%, -50%) rotate(28deg); border-color: #ffffff9c; }
.orbit-spark { position: absolute; width: 7px; height: 7px; border-radius: 50%; background: #fff; box-shadow: 0 0 0 5px #ffffff38, 0 0 20px #52bdaa66; }
.spark-one { top: 17%; left: 52%; }.spark-two { bottom: 19%; left: 29%; width: 5px; height: 5px; background: #40b5aa; }
.model-tile {
  position: absolute; display: flex; align-items: center; gap: 10px;
  padding: 12px 17px 12px 12px; border: 1px solid var(--auth-border); border-radius: 18px;
  color: #36545f; background: rgba(255, 255, 255, .58); backdrop-filter: blur(18px);
  box-shadow: inset 0 1px 0 #ffffffb3, 0 12px 30px #426b7912;
  font-size: 13px; font-weight: 600; animation: glass-float 7s ease-in-out infinite;
}
.model-symbol { display: grid; place-items: center; width: 30px; height: 30px; border-radius: 10px; font-size: 24px; }
.symbol-claude { color: #b77558; background: #e9cdb333; }
.symbol-gpt { color: #158778; background: #b3eadc4d; }
.symbol-gemini { color: #6481d3; background: #c5d6f54d; }
.tile-claude { left: 0; top: 23%; transform: rotate(-7deg); }
.tile-gpt { right: 0; top: 17%; transform: rotate(7deg); animation-delay: -2s; }
.tile-gemini { right: 4%; bottom: 12%; transform: rotate(-5deg); animation-delay: -4s; }
.showcase-caption { padding: 4px 0 0; }
.showcase-caption h1 { font-size: clamp(25px, 2.7vw, 34px); font-weight: 600; line-height: 1.5; letter-spacing: -1.4px; }
.showcase-caption h1 span { color: #168b82; }
.showcase-caption p { margin-top: 12px; font-size: 14px; line-height: 1.8; color: var(--auth-muted); }
.auth-form-panel { min-width: 0; width: 100%; max-width: 480px; justify-self: end; }
.auth-card {
  position: relative; padding: 40px; border: 1px solid var(--auth-border); border-radius: 30px;
  background: linear-gradient(145deg, #ffffff94, #ffffff47);
  -webkit-backdrop-filter: blur(30px) saturate(135%);
  backdrop-filter: blur(30px) saturate(135%);
  box-shadow: inset 0 1px 0 #ffffffd9, inset 0 -1px 0 #ffffff4d, 0 24px 80px #35596d12, 0 4px 16px #35596d05;
}
.auth-switch { margin-top: 28px; padding-top: 22px; border-top: 1px solid #71949826; text-align: center; font-size: 13px; }
.auth-card :deep(.input) { min-height: 48px; border-radius: 13px; background-color: #ffffff66; box-shadow: inset 0 1px 3px #22485b03; }
.auth-card :deep(.input:not(.input-error):not([class*="border-red-"]):not([class*="border-green-"])) { border-color: #73979f33; }
.auth-card :deep(.input:focus) { background-color: #ffffffb3; }
.auth-card :deep(.input:not(.input-error):not([class*="border-red-"]):not([class*="border-green-"]):focus) { border-color: #14b8a6; }
.auth-card :deep(.input:disabled) { opacity: .6; }
.auth-card :deep(.input::placeholder) { color: #7c909c; }
.auth-card :deep(.password-toggle) { width: 44px; justify-content: center; padding-right: 0; border-radius: 12px; }
.auth-card :deep(.password-toggle:focus-visible) { outline: 2px solid #0d9488; outline-offset: -4px; }
.auth-card :deep(.auth-code-fields) { display: grid; gap: 20px; }
.auth-card :deep(.auth-code-fields-paired) { grid-template-columns: repeat(2, minmax(0, 1fr)); }
.auth-card :deep(.btn-primary) { min-height: 47px; border-radius: 13px; background: linear-gradient(115deg, #0d9e94, #108c89); box-shadow: 0 6px 16px #0b918626, inset 0 1px 0 #ffffff30; }
.auth-card :deep(.btn-secondary) { min-height: 45px; background-color: #ffffff59; }
.auth-card :deep(.btn:focus-visible) { outline: 2px solid #0d9488; outline-offset: 3px; }
.auth-copyright { position: relative; padding: 16px 24px 22px; color: var(--auth-muted); text-align: center; font-size: 12px; }

.dark .auth-shell { --auth-ink: #e1f1f3; --auth-muted: #97aeb9; --auth-border: #d3e9ff21; --auth-glass: #1c344159; background: #0c171e; }
.dark .ambient-mint { background: radial-gradient(circle, #196c67, #174c46 45%, transparent 70%); opacity: .6; }
.dark .ambient-blue { background: radial-gradient(circle, #284876, #203454 50%, transparent 70%); opacity: .65; }
.dark .ambient-violet { background: #52466c; opacity: .3; }
.dark .header-action:hover { background: #304c6266; }
.dark .auth-card { background: linear-gradient(135deg, #243d4b80, #16233370); box-shadow: inset 0 1px 0 #d2f4ff14, 0 24px 80px #00000030; }
.dark .orb { border-color: #c0fff766; background: radial-gradient(circle at 28% 20%, #c1fff74d, transparent 45%), radial-gradient(circle at 70% 80%, #4b73a866, transparent 55%), linear-gradient(135deg, #235752b3, #2a686066 55%, #385d7599); box-shadow: inset 6px 6px 25px #c7fff530, inset -10px -10px 26px #0a1f3b66, 0 24px 65px #00000040, 0 0 0 12px #a7fff905; }
.dark .orb-label { color: #c1f7e9; text-shadow: 0 0 30px #6ee7cd4d; }
.dark .orb-label span { color: #9ed6d0; }
.dark .orb-latitude, .dark .orb-meridian { border-color: #d4fff526; }
.dark .orb::after { border-color: #ffffff1f; }
.dark .orbit-outer { border-color: #8bd8cd26; }
.dark .orbit-inner { border-color: #c4f8f21a; }
.dark .model-tile { background: #253e4c8c; color: #d3e7ec; box-shadow: inset 0 1px 0 #ffffff0d, 0 12px 30px #00000026; }
.dark .symbol-claude { color: #e6ac8d; }.dark .symbol-gpt { color: #84e6cc; }.dark .symbol-gemini { color: #b1c4ff; }
.dark .showcase-caption h1 span { color: #69d7c5; }
.dark .auth-switch { border-color: #c4f8f21a; }
.dark .auth-card :deep(.input) { background-color: #091c2b47; }
.dark .auth-card :deep(.input:not(.input-error):not([class*="border-red-"]):not([class*="border-green-"])) { border-color: #bbdbeb29; }
.dark .auth-card :deep(.input:focus) { background-color: #122c3b99; }
.dark .auth-card :deep(.input:not(.input-error):not([class*="border-red-"]):not([class*="border-green-"]):focus) { border-color: #48c7b5; }
.dark .auth-card :deep(.input::placeholder) { color: #8da4b2; }
.dark .auth-card :deep(.btn-secondary) { background-color: #203a4a80; }

@keyframes glass-float { 0%, 100% { translate: 0 0; } 50% { translate: 0 -9px; } }
@media (min-width: 1600px) { .auth-composition { max-width: 1200px; gap: 100px; } .gateway-art { height: 390px; } }
@media (max-width: 1000px) and (min-width: 761px) {
  .auth-composition { gap: 24px; grid-template-columns: 1fr 1fr; }
  .auth-card { padding: 30px; }
  .auth-card :deep(.auth-code-fields-paired) { grid-template-columns: 1fr; }
  .gateway-art { height: 320px; }
  .orb { width: 185px; height: 185px; }
  .model-tile { padding: 9px 12px 9px 9px; gap: 6px; }
  .showcase-caption h1 { font-size: 26px; }
}
@media (max-width: 760px) {
  .auth-header { padding: 20px; gap: 10px; }
  .auth-main { padding: 12px 20px 24px; }
  .auth-composition { max-width: 460px; grid-template-columns: 1fr; gap: 26px; }
  .auth-showcase { display: none; }
  .brand { gap: 11px; font-size: 23px; }
  .brand-mark { width: 42px; height: 42px; border-radius: 12px; }
  .gateway-art { display: none; }
  .showcase-caption { text-align: center; margin-top: 0; }
  .showcase-caption h1 { font-size: 25px; letter-spacing: -.8px; }
  .showcase-caption p { margin-top: 7px; font-size: 12px; }
  .auth-card { padding: 28px 24px; border-radius: 25px; }
  .header-action { border-radius: 12px; }
  .auth-card :deep(.input) { font-size: 16px; }
  .auth-card :deep(.auth-code-fields-paired) { grid-template-columns: 1fr; }
  .auth-switch { margin-top: 22px; padding-top: 18px; }
  .ambient-mint { left: -320px; top: -200px; }
  .ambient-blue { left: 0; top: 30%; }
}
@media (max-width: 420px) {
  .auth-header { padding-inline: 16px; }
  .brand { font-size: 21px; gap: 8px; }
  .brand-mark { width: 36px; height: 36px; }
  .auth-main { padding-inline: 16px; }
  .auth-card { padding: 26px 20px; }
}
@media (prefers-reduced-motion: reduce) {
  .model-tile { animation: none; }
  .header-action { transition: none; }
}
</style>
