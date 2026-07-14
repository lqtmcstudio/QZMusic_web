<script setup>
import { ref } from 'vue'
import { ImagePlus, LoaderCircle, Trash2 } from 'lucide-vue-next'
import { api } from '../lib/api'
import { appStore } from '../stores/app'

const props = defineProps({
  modelValue: { type: Array, default: () => [] },
  max: { type: Number, default: 5 },
  compact: Boolean,
})
const emit = defineEmits(['update:modelValue', 'uploaded'])
const uploading = ref(false)

async function choose(event) {
  const files = [...event.target.files]
  event.target.value = ''
  if (!files.length) return
  const remaining = props.max - props.modelValue.length
  if (remaining <= 0) return appStore.toast(`最多上传 ${props.max} 张图片`, 'warning')
  uploading.value = true
  let working = [...props.modelValue]
  try {
    for (const file of files.slice(0, remaining)) {
      if (!file.type.startsWith('image/')) {
        appStore.toast(`${file.name} 不是图片`, 'warning')
        continue
      }
      if (file.size > 25 * 1024 * 1024) {
        appStore.toast(`${file.name} 超过 25MB`, 'warning')
        continue
      }
      const result = await api.upload(file)
      working = [...working, result.url]
      emit('update:modelValue', working)
      emit('uploaded', result.url)
    }
  } catch (error) {
    appStore.toast(error.message, 'error')
  } finally {
    uploading.value = false
  }
}

function remove(index) {
  emit('update:modelValue', props.modelValue.filter((_, i) => i !== index))
}
</script>

<template>
  <div class="uploader" :class="{ 'uploader--compact': compact }">
    <div v-if="modelValue.length" class="upload-previews">
      <div v-for="(url, index) in modelValue" :key="url" class="upload-preview">
        <img :src="url" alt="已上传图片" />
        <button type="button" aria-label="移除图片" @click="remove(index)"><Trash2 :size="15" /></button>
      </div>
    </div>
    <label v-if="modelValue.length < max" class="upload-trigger" :class="{ 'is-uploading': uploading }">
      <LoaderCircle v-if="uploading" class="spin" :size="20" />
      <ImagePlus v-else :size="20" />
      <span>{{ uploading ? '正在上传…' : modelValue.length ? '继续添加' : '上传图片' }}</span>
      <small>{{ modelValue.length }}/{{ max }} · 单张不超过 25MB</small>
      <input type="file" accept="image/*" multiple :disabled="uploading" @change="choose" />
    </label>
  </div>
</template>
