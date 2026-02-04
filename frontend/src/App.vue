<script setup>
import { ref, onMounted, computed, watch, nextTick } from 'vue'
import { SelectDirectory, SelectWatermarkFile, SelectOutputDirectory, ExportImages, GetFonts, DeleteImage, SaveTempWatermark, GetImageBase64 } from '../wailsjs/go/main/App'
import Workspace from './components/Workspace.vue'
import Cropper from 'cropperjs'
import 'cropperjs/dist/cropper.css'

const drawer = ref(true)
const settingsDrawer = ref(true)
const files = ref([])
const searchQuery = ref('')
const selectedFile = ref(null)
const zoom = ref(100)
const activeTab = ref('elements')
const fonts = ref([])
const processing = ref(false)
const snackbar = ref(false)
const snackbarMsg = ref('')
const openGroups = ref([])
const overwriteFiles = ref(false)
const progress = ref(0)

// Crop State
const cropDialog = ref(false)
const cropImgSrc = ref('')
const cropImage = ref(null)
let cropperInstance = null

const defaultConfig = {
    type: 'image', content: '', opacity: 0.8, scale: 0.2, positionX: 0.5, positionY: 0.5,
    rotation: 0, quality: 90, brightness: 0, contrast: 0, saturation: 0, sharpness: 0,
    grayscale: false, textFont: '', textColor: '#ffffff', isLinked: true
}

const configsByResolution = ref({})
const individualOverrides = ref({})
const config = ref({ ...defaultConfig })

let isInternalChange = false

const getResKey = (file) => file ? `${file.width}x${file.height}` : null

const groupedFiles = computed(() => {
    const groups = {}
    const list = searchQuery.value 
        ? files.value.filter(f => f.name.toLowerCase().includes(searchQuery.value.toLowerCase()))
        : files.value

    list.forEach(file => {
        const res = getResKey(file)
        if (!groups[res]) groups[res] = []
        groups[res].push(file)
    })
    return groups
})

watch(config, (newVal) => {
    if (isInternalChange || !selectedFile.value) return
    const id = selectedFile.value.id
    const res = getResKey(selectedFile.value)
    const configCopy = JSON.parse(JSON.stringify(newVal))
    
    if (newVal.isLinked) {
        configsByResolution.value[res] = configCopy
        if (individualOverrides.value[id]) delete individualOverrides.value[id]
    } else {
        individualOverrides.value[id] = configCopy
    }
}, { deep: true })

watch(selectedFile, (newFile) => {
    if (newFile) {
        isInternalChange = true
        const id = newFile.id
        const res = getResKey(newFile)
        
        if (individualOverrides.value[id]) {
            config.value = { ...individualOverrides.value[id] }
        } else if (configsByResolution.value[res]) {
            config.value = { ...configsByResolution.value[res], isLinked: true }
        } else {
            const currentConfig = JSON.parse(JSON.stringify(config.value))
            configsByResolution.value[res] = { ...currentConfig, isLinked: true }
        }
        nextTick(() => { isInternalChange = false })
    }
})

const selectWatermark = async () => {
    const path = await SelectWatermarkFile()
    if (path) { config.value.content = path; config.value.type = 'image' }
}

const openCropDialog = async () => {
    if (!config.value.content || config.value.type !== 'image') return
    try {
        const base64 = await GetImageBase64(config.value.content)
        cropImgSrc.value = base64
        cropDialog.value = true
        nextTick(() => {
            if (cropperInstance) cropperInstance.destroy()
            cropperInstance = new Cropper(cropImage.value, {
                viewMode: 1,
                dragMode: 'move',
                autoCropArea: 0.8,
                responsive: true,
                background: false
            })
        })
    } catch (e) {
        snackbarMsg.value = "Error al cargar imagen para recorte"
        snackbar.value = true
    }
}

const saveCrop = async () => {
    if (!cropperInstance) return
    const canvas = cropperInstance.getCroppedCanvas()
    if (!canvas) return
    
    try {
        const base64 = canvas.toDataURL('image/png')
        const newPath = await SaveTempWatermark(base64)
        config.value.content = newPath
        cropDialog.value = false
        snackbarMsg.value = "¡Recorte aplicado!"
        snackbar.value = true
    } catch (e) {
        snackbarMsg.value = "Error al guardar recorte: " + e
        snackbar.value = true
    }
}

const syncToGroup = () => {
    if (!selectedFile.value) return
    const res = getResKey(selectedFile.value)
    
    // Clear overrides for ALL files in this group so they sync to master
    if (groupedFiles.value[res]) {
        groupedFiles.value[res].forEach(f => {
            if (individualOverrides.value[f.id]) {
                delete individualOverrides.value[f.id]
            }
        })
    }

    // Force current config into the group master map
    const masterConfig = JSON.parse(JSON.stringify(config.value))
    masterConfig.isLinked = true
    configsByResolution.value[res] = masterConfig
    
    // Ensure current active config is also linked
    config.value.isLinked = true
    
    snackbarMsg.value = `Posición maestra fijada para ${res}`
    snackbar.value = true
}

const copyToAllGroups = () => {
    if (!confirm("¿Aplicar a TODOS?")) return
    const currentStyle = JSON.parse(JSON.stringify(config.value))
    
    // Clear ALL overrides globally
    individualOverrides.value = {}

    Object.keys(groupedFiles.value).forEach(res => {
        configsByResolution.value[res] = { ...currentStyle, isLinked: true }
    })
    
    // Ensure current config is marked as linked
    config.value.isLinked = true

    snackbarMsg.value = "Sincronización global completada"
    snackbar.value = true
}

const updatePos = ({ x, y }) => {
    config.value.positionX = x
    config.value.positionY = y
}

const updateScale = (s) => {
    config.value.scale = s
}

const centerWatermark = () => {
    config.value.positionX = 0.5
    config.value.positionY = 0.5
}

const deleteFileFromList = (file) => {
    const index = files.value.findIndex(f => f.id === file.id)
    files.value.splice(index, 1)
    if (selectedFile.value?.id === file.id) {
        selectedFile.value = files.value.length > 0 ? files.value[0] : null
    }
}

const openFolder = async () => {
    const result = await SelectDirectory()
    if (result) {
        files.value = result
        if (files.value.length > 0) {
            selectedFile.value = files.value[0]
            openGroups.value = [getResKey(files.value[0])]
        }
    }
}

const exportAll = async () => {
    try {
        let outDir = ""
        if (!overwriteFiles.value) {
            outDir = await SelectOutputDirectory()
            if (!outDir) return
        }

        processing.value = true
        progress.value = 0
        let processedCount = 0
        const total = files.value.length

        for (const file of files.value) {
            const res = getResKey(file)
            const id = file.id
            let exportConfig = individualOverrides.value[id] || configsByResolution.value[res] || config.value
            
            // If overwriting, use the file's own directory
            const targetDir = overwriteFiles.value ? file.path.substring(0, file.path.lastIndexOf('/')) : outDir
            await ExportImages([file], exportConfig, targetDir)
            
            processedCount++
            progress.value = Math.round((processedCount / total) * 100)
        }
        snackbarMsg.value = "¡Exportación finalizada!"
        snackbar.value = true
    } catch (e) {
        snackbarMsg.value = "Error: " + e
        snackbar.value = true
    } finally { 
        processing.value = false
        setTimeout(() => { progress.value = 0 }, 500)
    }
}

onMounted(async () => {
    fonts.value = await GetFonts()
    window.addEventListener('keydown', (e) => {
        if (e.target.tagName === 'INPUT' || e.target.tagName === 'TEXTAREA') return
        const list = files.value
        const idx = list.findIndex(f => f.id === selectedFile.value?.id)
        if (e.key === 'ArrowDown' || e.key === 'ArrowRight') {
            if (idx < list.length - 1) selectedFile.value = list[idx + 1]
        } else if (e.key === 'ArrowUp' || e.key === 'ArrowLeft') {
            if (idx > 0) selectedFile.value = list[idx - 1]
        }
    })
})
</script>

<template>
  <v-app theme="dark">
    <!-- Left Drawer (Library) -->
    <v-navigation-drawer v-model="drawer" app permanent width="240" color="#0a0a0a" border="0">
      <div class="pa-4 d-flex align-center justify-space-between" style="min-height: 64px;">
          <div class="text-h5 font-weight-black text-white" style="letter-spacing: -1.5px; line-height: 1;">SHUYUY<span class="text-primary">.</span></div>
          <v-btn icon="mdi-backburger" variant="text" size="small" @click="drawer = false" color="grey-darken-1"></v-btn>
      </div>
      
      <div class="px-3 pb-3">
          <v-btn block color="primary" variant="flat" prepend-icon="mdi-plus" @click="openFolder" class="rounded-lg mb-3 text-none" size="small">Importar</v-btn>
          <v-text-field v-model="searchQuery" prepend-inner-icon="mdi-magnify" placeholder="Buscar..." variant="solo-filled" density="compact" flat hide-details class="rounded-lg"></v-text-field>
      </div>

      <v-divider></v-divider>

      <v-list v-model:opened="openGroups" open-strategy="multiple" density="compact" class="py-0">
          <v-list-group v-for="(filesInGroup, res) in groupedFiles" :key="res" :value="res">
              <template v-slot:activator="{ props }">
                  <v-list-item v-bind="props" :title="res" prepend-icon="mdi-folder-outline" class="group-header"></v-list-item>
              </template>
              <v-list-item 
                v-for="file in filesInGroup" :key="file.id" :value="file"
                :title="file.name" @click="selectedFile = file"
                :active="selectedFile?.id === file.id" color="primary" class="pl-6"
              >
                  <template v-slot:prepend>
                      <v-icon size="10" :color="individualOverrides[file.id] ? 'orange' : 'primary'" class="mr-2">
                          {{ individualOverrides[file.id] ? 'mdi-link-variant-off' : 'mdi-link-variant' }}
                      </v-icon>
                  </template>
                  <template v-slot:append>
                      <v-btn icon="mdi-close" variant="text" size="x-small" @click.stop="deleteFileFromList(file)" class="delete-btn"></v-btn>
                  </template>
              </v-list-item>
          </v-list-group>
      </v-list>
    </v-navigation-drawer>

    <!-- Right Drawer (Settings) -->
    <v-navigation-drawer v-model="settingsDrawer" location="right" app width="300" color="#0a0a0a" border="0">
        <div class="pa-4" v-if="selectedFile">
            <div class="d-flex align-center justify-space-between mb-4">
                <span class="text-caption font-weight-black text-grey">INSPECTOR</span>
                <v-btn icon="mdi-forwardburger" variant="text" size="small" @click="settingsDrawer = false" color="grey"></v-btn>
            </div>

            <v-card color="#121212" class="rounded-lg pa-3 mb-4" elevation="0" border="sm grey-darken-4">
                <div class="d-flex align-center justify-space-between mb-1">
                    <span class="text-caption font-weight-black" :class="config.isLinked ? 'text-primary' : 'text-orange'">{{ config.isLinked ? 'VINCULADO' : 'INDIVIDUAL' }}</span>
                    <v-switch v-model="config.isLinked" hide-details density="compact" color="primary" inset></v-switch>
                </div>
                <v-btn block size="x-small" color="primary" variant="tonal" @click="syncToGroup" class="mt-2 text-none">Fijar como Maestro</v-btn>
            </v-card>

            <div class="d-flex gap-2 mb-4 bg-black pa-1 rounded-pill justify-space-between">
                <v-btn icon="mdi-target" variant="text" size="x-small" @click="centerWatermark"></v-btn>
                <v-btn icon="mdi-refresh" variant="text" size="x-small" @click="config = {...defaultConfig, content: config.content}"></v-btn>
                <v-btn icon="mdi-earth" variant="text" size="x-small" @click="copyToAllGroups" color="primary"></v-btn>
                <v-chip size="x-small" variant="flat" color="grey-darken-4">{{ getResKey(selectedFile) }}</v-chip>
            </div>

            <v-tabs v-model="activeTab" density="compact" color="primary" grow class="mb-4">
                <v-tab value="elements" class="text-caption">Logo</v-tab>
                <v-tab value="filters" class="text-caption">Editar</v-tab>
                <v-tab value="export" class="text-caption">Exportar</v-tab>
            </v-tabs>

            <div v-if="activeTab === 'elements'">
                <div class="d-flex gap-2 mb-4">
                    <v-btn color="primary" variant="tonal" size="small" class="flex-grow-1 rounded-lg text-none" @click="selectWatermark">Cambiar</v-btn>
                    <v-btn color="grey-darken-3" variant="flat" size="small" class="rounded-lg text-none" @click="openCropDialog" :disabled="!config.content || config.type !== 'image'">
                        <v-icon icon="mdi-crop" />
                    </v-btn>
                </div>
                
                <div class="mb-3">
                   <div class="d-flex justify-space-between text-caption mb-1">
                       <span class="text-capitalize text-grey">Opacidad</span>
                       <span>{{ config.opacity }}</span>
                   </div>
                   <v-slider v-model="config.opacity" :min="0" :max="1" :step="0.01" hide-details color="primary" density="compact"></v-slider>
                </div>
                
                <div class="mb-3">
                   <div class="d-flex justify-space-between text-caption mb-1">
                       <span class="text-capitalize text-grey">Escala</span>
                       <span>{{ config.scale }}</span>
                   </div>
                   <v-slider v-model="config.scale" :min="0" :max="1" :step="0.01" hide-details color="primary" density="compact"></v-slider>
                </div>
                
                <div class="mb-3">
                   <div class="d-flex justify-space-between text-caption mb-1">
                       <span class="text-capitalize text-grey">Rotación</span>
                       <span>{{ config.rotation }}</span>
                   </div>
                   <v-slider v-model="config.rotation" :min="0" :max="360" :step="1" hide-details color="primary" density="compact"></v-slider>
                </div>
            </div>

            <div v-if="activeTab === 'filters'">
                <div class="mb-3">
                   <div class="d-flex justify-space-between text-caption mb-1">
                       <span class="text-capitalize text-grey">Brillo</span>
                       <span>{{ config.brightness }}%</span>
                   </div>
                   <v-slider v-model="config.brightness" :min="-100" :max="100" :step="1" hide-details color="primary" density="compact"></v-slider>
                </div>
                <div class="mb-3">
                   <div class="d-flex justify-space-between text-caption mb-1">
                       <span class="text-capitalize text-grey">Contraste</span>
                       <span>{{ config.contrast }}%</span>
                   </div>
                   <v-slider v-model="config.contrast" :min="-100" :max="100" :step="1" hide-details color="primary" density="compact"></v-slider>
                </div>
                <div class="mb-3">
                   <div class="d-flex justify-space-between text-caption mb-1">
                       <span class="text-capitalize text-grey">Saturación</span>
                       <span>{{ config.saturation }}%</span>
                   </div>
                   <v-slider v-model="config.saturation" :min="-100" :max="100" :step="1" hide-details color="primary" density="compact"></v-slider>
                </div>
                <div class="mb-3">
                   <div class="d-flex justify-space-between text-caption mb-1">
                       <span class="text-capitalize text-grey">Nitidez</span>
                       <span>{{ config.sharpness }}%</span>
                   </div>
                   <v-slider v-model="config.sharpness" :min="0" :max="100" :step="1" hide-details color="primary" density="compact"></v-slider>
                </div>
                <v-switch v-model="config.grayscale" label="Escala de Grises" hide-details density="compact" color="primary" size="small"></v-switch>
            </div>

            <div v-if="activeTab === 'export'">
                <div class="mb-6">
                   <div class="d-flex justify-space-between text-caption mb-1">
                       <span class="text-capitalize text-grey">Calidad (Compresión)</span>
                       <span class="text-primary font-weight-bold">{{ config.quality }}%</span>
                   </div>
                   <v-slider v-model="config.quality" :min="1" :max="100" :step="1" hide-details color="primary" density="compact"></v-slider>
                   <div class="text-caption text-grey-darken-1 mt-1">Valores bajos reducen el peso del archivo.</div>
                </div>

                <v-card color="#1a1a1a" class="rounded-lg pa-3 mb-4" border="sm orange-darken-4" v-if="overwriteFiles">
                    <div class="d-flex align-center gap-2 text-orange text-caption mb-1">
                        <v-icon size="small">mdi-alert</v-icon>
                        <span class="font-weight-black">MODO PELIGROSO</span>
                    </div>
                    <div class="text-caption text-grey">Se reemplazarán las fotos originales.</div>
                </v-card>

                <v-switch v-model="overwriteFiles" label="Sobrescribir Originales" hide-details density="compact" color="orange" size="small" class="mb-4"></v-switch>
            </div>

            <v-btn block size="large" :color="overwriteFiles ? 'orange' : 'primary'" @click="exportAll" :loading="processing" class="rounded-lg mt-6 font-weight-black text-none">
                {{ overwriteFiles ? 'Sobrescribir Todo' : 'Exportar Todo' }}
            </v-btn>
        </div>
    </v-navigation-drawer>

    <!-- Main View -->
    <v-main class="bg-black pa-0 overflow-hidden">
      <!-- Re-open buttons -->
      <v-btn v-if="!drawer" icon="mdi-menu" variant="flat" color="primary" size="x-small" class="position-absolute top-0 left-0 ma-2 z-index-100 rounded-lg" @click="drawer = true"></v-btn>
      <v-btn v-if="!settingsDrawer" icon="mdi-cog" variant="flat" color="primary" size="x-small" class="position-absolute top-0 right-0 ma-2 z-index-100 rounded-lg" @click="settingsDrawer = true"></v-btn>

      <div class="fill-height d-flex flex-column align-center justify-center position-relative">
        <div v-if="selectedFile" class="fill-height w-100 position-relative">
            <Workspace :file="selectedFile" :config="config" :zoom="zoom" @update-position="updatePos" @update-scale="updateScale" />
            
            <div class="zoom-bar-container">
                <div class="zoom-bar d-flex align-center px-3 py-1 rounded-pill elevation-20 border border-grey-darken-4">
                    <v-icon size="x-small" @click="zoom = Math.max(10, zoom - 10)">mdi-magnify-minus</v-icon>
                    <v-slider v-model="zoom" min="10" max="300" step="1" hide-details class="mx-2" color="primary"></v-slider>
                    <v-icon size="x-small" @click="zoom = Math.min(300, zoom + 10)">mdi-magnify-plus</v-icon>
                    <div class="ml-2 font-weight-bold text-caption" style="width: 35px">{{ zoom }}%</div>
                </div>
            </div>
        </div>
        <div v-else class="text-center">
            <h1 class="text-h4 font-weight-black text-grey-darken-3">SHUYUY</h1>
            <v-btn color="primary" size="large" class="mt-4 rounded-lg px-8 text-none" @click="openFolder">Seleccionar Carpeta</v-btn>
        </div>
      </div>
      
      <!-- Crop Dialog -->
      <v-dialog v-model="cropDialog" max-width="800px" persistent>
          <v-card color="#1e1e1e" class="rounded-lg">
              <v-card-title class="text-white">Recortar Logo</v-card-title>
              <v-card-text>
                  <div style="height: 400px; width: 100%; background: #000;">
                      <img ref="cropImage" :src="cropImgSrc" style="max-width: 100%; max-height: 100%; display: block;" />
                  </div>
              </v-card-text>
              <v-card-actions class="justify-end">
                  <v-btn variant="text" color="grey" @click="cropDialog = false">Cancelar</v-btn>
                  <v-btn color="primary" variant="flat" @click="saveCrop">Aplicar Recorte</v-btn>
              </v-card-actions>
          </v-card>
      </v-dialog>
      
      <!-- Loading Overlay -->
      <v-overlay :model-value="processing" class="align-center justify-center" persistent style="backdrop-filter: blur(8px); background: rgba(0,0,0,0.6);">
        <div class="text-center pa-8 rounded-xl bg-grey-darken-4 elevation-24 border border-grey-darken-3 position-relative overflow-hidden" style="min-width: 320px; box-shadow: 0 0 100px rgba(0,0,0,0.5);">
            <!-- Background Glow -->
            <div class="position-absolute" style="top: -50%; left: -50%; width: 200%; height: 200%; background: radial-gradient(circle, rgba(var(--v-theme-primary), 0.1) 0%, transparent 70%); pointer-events: none;"></div>
            
            <div class="mb-6 position-relative d-inline-block">
                <v-progress-circular :model-value="progress" :size="120" :width="8" color="primary" bg-color="grey-darken-3">
                    <div class="d-flex flex-column align-center">
                        <span class="text-h4 font-weight-black text-white" style="line-height: 1;">{{ progress }}<span class="text-caption">%</span></span>
                    </div>
                </v-progress-circular>
            </div>
            <div class="text-h6 font-weight-bold text-white mb-2" style="letter-spacing: 0.5px;">PROCESANDO</div>
            <div class="text-caption text-grey-lighten-1">Optimizando y aplicando marcas de agua...</div>
            <div class="mt-4 text-caption text-primary font-weight-bold" v-if="overwriteFiles">MODO SOBRESCRITURA</div>
        </div>
      </v-overlay>
      
      <v-snackbar v-model="snackbar" timeout="2000" color="grey-darken-3">{{ snackbarMsg }}</v-snackbar>
    </v-main>
  </v-app>
</template>

<style>
body { background: #000; overflow: hidden; margin: 0; }
.v-main { height: 100vh; position: relative; }
.zoom-bar-container { position: absolute; bottom: 20px; left: 0; right: 0; display: flex; justify-content: center; pointer-events: none; }
.zoom-bar { background: rgba(20, 20, 20, 0.8); backdrop-filter: blur(8px); width: 280px; pointer-events: auto; }
.gap-2 { gap: 6px; }
.z-index-100 { z-index: 100; }
.group-header { font-size: 0.7rem !important; color: #555 !important; min-height: 32px !important; }
.delete-btn { opacity: 0; transition: 0.2s; }
.v-list-item:hover .delete-btn { opacity: 1; }
/* Clean UI overrides */
.v-navigation-drawer { border-radius: 0 !important; margin: 0 !important; height: 100% !important; }
</style>