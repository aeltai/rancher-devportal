<template>
  <div class="dp-wizard">
    <header class="dp-wizard-head">
      <h2>{{ bannerTitle }}</h2>
      <nav class="dp-wizard-steps">
        <button
          v-for="(step, idx) in steps"
          :key="step.name"
          type="button"
          :class="['dp-wizard-step', { active: idx === current, done: idx < current && step.ready }]"
          :disabled="idx > current && !step.ready"
          @click="goTo(idx)"
        >
          <span class="num">{{ idx + 1 }}</span>
          <span class="label">{{ step.label }}</span>
        </button>
      </nav>
    </header>

    <div v-if="errors.length" class="banner error">
      <ul>
        <li v-for="(err, i) in errors" :key="i">{{ err }}</li>
      </ul>
    </div>

    <section class="dp-wizard-body">
      <slot :name="currentStep.name" />
    </section>

    <footer class="dp-wizard-foot">
      <button class="btn role-secondary" type="button" @click="$emit('cancel')">Cancel</button>
      <div class="dp-wizard-nav">
        <button v-if="current > 0" class="btn role-secondary" type="button" @click="prev">Back</button>
        <button
          v-if="current < steps.length - 1"
          class="btn role-primary"
          type="button"
          :disabled="!currentStep.ready"
          @click="next"
        >
          Next
        </button>
        <button
          v-else
          class="btn role-primary"
          type="button"
          :disabled="!allReady"
          @click="$emit('finish')"
        >
          Create
        </button>
      </div>
    </footer>
  </div>
</template>

<script>
export default {
  name: 'SimpleWizard',
  props: {
    steps: { type: Array, required: true },
    errors: { type: Array, default: () => [] },
    bannerTitle: { type: String, default: 'New request' },
  },
  emits: ['cancel', 'finish'],
  data() {
    return { current: 0 };
  },
  computed: {
    currentStep() {
      return this.steps[this.current] || { name: 'basics', ready: false };
    },
    allReady() {
      return this.steps.every((s) => s.ready);
    },
  },
  watch: {
    steps: {
      deep: true,
      handler(steps) {
        if (this.current >= steps.length) {
          this.current = Math.max(0, steps.length - 1);
        }
      },
    },
  },
  methods: {
    goTo(idx) {
      if (idx <= this.current || (this.steps[idx - 1] && this.steps[idx - 1].ready)) {
        this.current = idx;
      }
    },
    next() {
      if (this.current < this.steps.length - 1 && this.currentStep.ready) {
        this.current += 1;
      }
    },
    prev() {
      if (this.current > 0) this.current -= 1;
    },
  },
};
</script>

<style scoped>
.dp-wizard {
  display: flex;
  flex-direction: column;
  gap: 12px;
  width: 100%;
  align-self: stretch;
  text-align: left;
}
.dp-wizard-head h2 { margin: 0 0 8px; font-size: 1.1em; text-align: left; }
.dp-wizard-steps { display: flex; flex-wrap: wrap; gap: 6px; justify-content: flex-start; }
.dp-wizard-step {
  display: flex; align-items: center; gap: 6px; padding: 4px 10px;
  border: 1px solid var(--border); border-radius: 4px; background: var(--body-bg);
  cursor: pointer; font-size: .85em;
}
.dp-wizard-step.active { border-color: var(--primary); background: var(--primary); color: #fff; }
.dp-wizard-step.done:not(.active) { border-color: var(--success); }
.dp-wizard-step:disabled { opacity: .5; cursor: not-allowed; }
.dp-wizard-step .num { font-weight: 600; }
.dp-wizard-body { min-height: 200px; width: 100%; }
.dp-wizard-foot {
  display: flex;
  justify-content: flex-start;
  align-items: center;
  gap: 8px;
  width: 100%;
}
.dp-wizard-nav { display: flex; gap: 8px; margin-left: auto; }
.banner.error { padding: 8px 10px; border-radius: 4px; background: rgba(204,74,74,.12); color: var(--error, #c00); }
</style>
