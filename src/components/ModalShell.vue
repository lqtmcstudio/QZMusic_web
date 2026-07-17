<script setup>
import { X } from 'lucide-vue-next'

defineProps({
  open: Boolean,
  title: String,
  eyebrow: String,
  wide: Boolean,
})

defineEmits(['close'])
</script>

<template>
  <Teleport to="body">
    <Transition name="modal">
      <div v-if="open" class="modal-backdrop" @mousedown.self="$emit('close')">
        <section class="modal-panel" :class="{ 'modal-panel--wide': wide }" role="dialog" aria-modal="true" :aria-label="title">
          <header class="modal-header">
            <div>
              <span v-if="eyebrow" class="eyebrow">{{ eyebrow }}</span>
              <h2>{{ title }}</h2>
            </div>
            <button class="icon-button" type="button" aria-label="关闭" @click="$emit('close')"><X :size="20" /></button>
          </header>
          <div class="modal-body"><slot /></div>
        </section>
      </div>
    </Transition>
  </Teleport>
</template>
