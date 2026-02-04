<script setup>
import { ref, watch, onMounted, computed, nextTick } from 'vue'
import interact from 'interactjs'
import { GetImageBase64 } from '../../wailsjs/go/main/App'

const props = defineProps({
  file: Object,
  config: Object,
  zoom: {
      type: Number,
      default: 100
  }
})

const emit = defineEmits(['update-position', 'update-scale'])

const imgSrc = ref('')
const wmSrc = ref('')
const loading = ref(false)
const watermark = ref(null)
const wrapper = ref(null)

const loadImage = async () => {
  if (!props.file) return
  loading.value = true
  imgSrc.value = '' 
  
  try {
    const base64 = await GetImageBase64(props.file.path)
    if (base64) {
        imgSrc.value = base64
    }
  } catch (e) {
    console.error("Critical error loading image:", e)
  } finally {
    loading.value = false
    nextTick(() => syncFromConfig())
  }
}

const loadWatermark = async () => {
    if (!props.config.content || props.config.type !== 'image') {
        wmSrc.value = ''
        return
    }
    try {
        wmSrc.value = await GetImageBase64(props.config.content)
    } catch (e) {
        console.error("Error loading watermark:", e)
    }
}

const filterStyle = computed(() => {
    return {
        filter: `brightness(${1 + props.config.brightness / 100}) 
                 contrast(${1 + props.config.contrast / 100}) 
                 saturate(${1 + props.config.saturation / 100})
                 grayscale(${props.config.grayscale ? 1 : 0})
                 ${props.config.sharpness > 0 ? 'contrast(1.1) brightness(1.05)' : ''}`
    }
})

const canvasStyle = computed(() => {
    if (!props.file) return {}
    return {
        aspectRatio: `${props.file.width}/${props.file.height}`,
        transform: `scale(${props.zoom / 100})`,
        transformOrigin: 'center center'
    }
})

const textStyle = computed(() => {
    if (!wrapper.value) return {}
    const rect = wrapper.value.getBoundingClientRect()
    // We must divide rect size by current zoom to get real relative size for font
    const realHeight = rect.height / (props.zoom / 100)
    return {
        color: props.config.textColor,
        fontSize: `${props.config.scale * realHeight * 1.2}px`,
        opacity: props.config.opacity,
        fontWeight: 'bold',
        textShadow: '2px 2px 4px rgba(0,0,0,0.5)',
        lineHeight: '1'
    }
})

watch(() => props.file, loadImage, { immediate: true })
watch(() => [props.config.content, props.config.type], loadWatermark, { immediate: true })

const syncFromConfig = async () => {
    await nextTick()
    if (!wrapper.value || !watermark.value || !imgSrc.value) return
    
    const rect = wrapper.value.getBoundingClientRect()
    if (rect.width === 0) return

    // Calculate logical dimensions (independent of zoom transform)
    const zoomFactor = props.zoom / 100
    const logicalWidth = rect.width / zoomFactor
    const logicalHeight = rect.height / zoomFactor

    const x = props.config.positionX * logicalWidth
    const y = props.config.positionY * logicalHeight
    
    if (props.config.type === 'image') {
        const w = props.config.scale * logicalWidth
        watermark.value.style.width = `${w}px`
        watermark.value.style.height = 'auto'
    } else {
        watermark.value.style.width = 'auto'
        watermark.value.style.height = 'auto'
    }
    
    watermark.value.style.transform = `translate(${x}px, ${y}px) rotate(${props.config.rotation}deg)`
    watermark.value.setAttribute('data-x', x)
    watermark.value.setAttribute('data-y', y)
}

// Deep watch for ALL config changes to ensure immediate visual update
watch(() => props.config, syncFromConfig, { deep: true })
watch(() => [props.zoom, imgSrc.value, wmSrc.value], syncFromConfig)

onMounted(() => {
    window.addEventListener('resize', syncFromConfig)
    
    interact(watermark.value)
        .draggable({
            modifiers: [interact.modifiers.restrictRect({ restriction: 'parent', endOnly: false })],
            listeners: {
                move (event) {
                    const target = event.target
                    // Adjust movement by zoom factor so drag feels natural
                    const zoomFactor = props.zoom / 100
                    const x = (parseFloat(target.getAttribute('data-x')) || 0) + (event.dx / zoomFactor)
                    const y = (parseFloat(target.getAttribute('data-y')) || 0) + (event.dy / zoomFactor)
                    
                    target.style.transform = `translate(${x}px, ${y}px) rotate(${props.config.rotation}deg)`
                    target.setAttribute('data-x', x)
                    target.setAttribute('data-y', y)
                    
                    if (wrapper.value) {
                        const rect = wrapper.value.getBoundingClientRect()
                        const logicalW = rect.width / zoomFactor
                        const logicalH = rect.height / zoomFactor
                        emit('update-position', { x: x / logicalW, y: y / logicalH })
                    }
                }
            }
        })
        .resizable({
            edges: { top: true, left: true, bottom: true, right: true },
            listeners: {
                move: function (event) {
                    const zoomFactor = props.zoom / 100
                    let { x, y } = event.target.dataset
                    x = (parseFloat(x) || 0) + (event.deltaRect.left / zoomFactor)
                    y = (parseFloat(y) || 0) + (event.deltaRect.top / zoomFactor)
                    
                    const logicalW = event.rect.width / zoomFactor
                    
                    Object.assign(event.target.style, {
                        width: `${logicalW}px`,
                        transform: `translate(${x}px, ${y}px) rotate(${props.config.rotation}deg)`
                    })

                    Object.assign(event.target.dataset, { x, y })
                    
                    if (wrapper.value) {
                        const rect = wrapper.value.getBoundingClientRect()
                        const parentLogicalW = rect.width / zoomFactor
                        const parentLogicalH = rect.height / zoomFactor
                         emit('update-position', { x: x / parentLogicalW, y: y / parentLogicalH })
                        emit('update-scale', logicalW / parentLogicalW)
                    }
                }
            }
        })
})

const onImageLoad = () => {
    syncFromConfig()
}
</script>

<template>
  <div class="workspace-viewport">
      <v-progress-circular v-if="loading" indeterminate color="primary" size="64" width="6"></v-progress-circular>
      
      <div v-show="imgSrc && !loading" class="canvas-scroll-container">
          <div ref="wrapper" class="image-canvas" :style="canvasStyle">
              <img :src="imgSrc" class="base-image" :style="filterStyle" @load="onImageLoad" />
              
              <div ref="watermark" class="watermark-element" :style="{ opacity: config.type === 'image' ? config.opacity : 1 }">
                  <img v-if="config.type === 'image' && wmSrc" :src="wmSrc" class="wm-image" />
                  <div v-else-if="config.type === 'text'" :style="textStyle" class="wm-text">
                      {{ config.content }}
                  </div>
              </div>
          </div>
      </div>

      <div v-if="!imgSrc && !loading" class="text-center text-grey">
          <v-icon size="64">mdi-image-off</v-icon>
          <p>Error al mostrar la imagen</p>
      </div>
  </div>
</template>

<style scoped>
.workspace-viewport {
    flex: 1;
    display: flex;
    align-items: center;
    justify-content: center;
    background-color: #000;
    overflow: hidden;
    position: relative;
    width: 100%;
    height: 100%;
}

.canvas-scroll-container {
    width: 100%;
    height: 100%;
    display: flex;
    align-items: center;
    justify-content: center;
    overflow: auto;
    padding: 20px; /* Reduced padding */
}

.image-canvas {
    position: relative;
    max-width: 95%;
    max-height: 95%;
    box-shadow: 0 30px 60px rgba(0,0,0,0.8);
    background-color: #000;
    flex-shrink: 0;
}

.base-image {
    max-width: 100%;
    max-height: 100%;
    object-fit: contain;
    display: block;
    user-select: none;
    pointer-events: none;
}

.watermark-element {
    position: absolute;
    top: 0;
    left: 0;
    z-index: 10;
    border: 1px dashed rgba(33, 150, 243, 0.4);
    display: flex;
    align-items: center;
    justify-content: center;
    user-select: none;
    touch-action: none;
    cursor: move;
}

.watermark-element:hover {
    border: 1px solid #2196F3;
    background: rgba(33, 150, 243, 0.1);
}

.wm-image {
    width: 100%;
    height: auto;
    display: block;
    pointer-events: none;
}

.wm-text {
    white-space: nowrap;
    pointer-events: none;
    padding: 5px;
}
</style>