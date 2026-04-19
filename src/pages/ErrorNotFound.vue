<template>
  <q-page class="error-page">
    <div class="error-inner">
      <div class="error-glyph">◈</div>
      <div class="error-code">404</div>
      <h2 class="error-title">页面不存在</h2>
      <p class="error-desc">您访问的页面可能已被移除或地址有误</p>
      <q-btn unelevated color="primary" label="返回首页" to="/" class="error-btn" />
    </div>
  </q-page>
</template>

<style scoped lang="scss">
.error-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--color-bg-primary);
}

.error-inner {
  text-align: center;
  position: relative;
  z-index: 1;
}

.error-glyph {
  font-size: 6rem;
  color: var(--color-border);
  margin-bottom: 16px;
  line-height: 1;
}

.error-code {
  font-family: var(--font-mono);
  font-size: 8rem;
  font-weight: 700;
  color: var(--color-bg-elevated);
  line-height: 1;
  margin-bottom: 16px;
  background: linear-gradient(135deg, var(--color-accent-gold) 0%, var(--color-accent-teal) 100%);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
}

.error-title {
  font-family: var(--font-display);
  font-size: 1.8rem;
  font-weight: 700;
  margin-bottom: 8px;
}

.error-desc {
  color: var(--color-text-muted);
  font-size: 0.9rem;
  margin-bottom: 32px;
}

.error-btn {
  font-size: 0.95rem;
  font-weight: 600;
  padding: 10px 32px;
}
</style>
