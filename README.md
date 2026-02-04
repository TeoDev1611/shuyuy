# SHUYUY 📸

![License](https://img.shields.io/badge/license-GPLv3-blue.svg)  ![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8.svg) ![Vue Version](https://img.shields.io/badge/Vue-3.x-4FC08D.svg) ![Wails](https://img.shields.io/badge/Wails-v2-red.svg)

**Shuyuy** es una potente aplicación de escritorio diseñada para fotógrafos profesionales y entusiastas que necesitan optimizar su flujo de trabajo de post-procesamiento. Shuyuy permite la gestión masiva de imágenes, aplicando marcas de agua, redimensionamiento, compresión y filtros de manera eficiente y visual.

Construida sobre la robustez de **Go (Golang)** y la flexibilidad de **Vue 3**, Shuyuy ofrece un rendimiento nativo con una interfaz moderna y reactiva.

---

## ✨ Características Principales

*   **⚡ Procesamiento Ultrarrápido:** Motor escrito en Go que aprovecha la concurrencia para procesar cientos de imágenes en segundos.
*   **📂 Clasificación Inteligente:** Detecta y agrupa imágenes automáticamente por resolución (Landscape, Portrait, etc.).
*   **🖌️ Editor Visual WYSIWYG:** Posiciona, escala y rota tu marca de agua visualmente sobre la imagen.
*   **🔗 Sistema de Herencia (Vinculación):** Aplica cambios a una sola foto y replícalos automáticamente a todas las imágenes del mismo grupo (resolución).
*   **✂️ Recorte de Logos:** Herramienta integrada para recortar y ajustar tu logo sin salir de la aplicación.
*   **🎨 Filtros y Ajustes:** Brillo, contraste, saturación, nitidez y conversión a blanco y negro.
*   **💾 Exportación Flexible:** Control total sobre la calidad de compresión JPEG y opciones para sobrescribir archivos originales o guardar en nueva ubicación.

---

## 📖 Manual de Usuario

### 1. Primeros Pasos
Al iniciar Shuyuy, selecciona una carpeta que contenga tus imágenes. La aplicación escaneará recursivamente todas las subcarpetas compatibles (`.jpg`, `.jpeg`, `.png`).

### 2. Panel de Librería (Izquierda)
*   **Agrupación:** Las fotos se organizan por resolución.
*   **Selección:** Haz clic en una foto para cargarla en el área de trabajo.
*   **Iconos de Estado:**
    *   🔗 **Vinculado:** La imagen sigue las reglas del grupo.
    *   🔗❌ **Individual:** La imagen tiene ajustes únicos.

### 3. Inspector (Derecha)

#### Pestaña "Logo"
*   **Cargar Marca de Agua:** Soporta PNG (transparencia), JPG.
*   **Controles:** Opacidad, Escala y Rotación.
*   **Botón Recortar:** Abre un modal para recortar los bordes vacíos de tu logo.

#### Pestaña "Editar"
*   Ajustes no destructivos para mejorar la imagen antes de exportar.

#### Pestaña "Exportar"
*   **Calidad:** Slider de 1 a 100.
    *   *Recomendación Web:* 70-80%
    *   *Recomendación Impresión:* 90-100%
*   **Modo Sobrescribir:** ⚠️ **¡Cuidado!** Reemplaza los archivos originales. Útil para flujos de trabajo rápidos donde no quieres duplicar archivos.

### 4. Controles Globales
*   **Fijar como Maestro:** Convierte la configuración de la imagen actual en la regla para todo su grupo de resolución.
*   **Botón Mundial (🌍):** Aplica la configuración actual a **TODAS** las imágenes de la sesión.

---

## 🛠️ Guía Técnica y Desarrollo

### Requisitos del Sistema
*   **Go**: v1.21 o superior.
*   **Node.js**: v18+ y npm.
*   **Wails CLI**: `go install github.com/wailsapp/wails/v2/cmd/wails@latest`
*   **Dependencias de Linux**: `libgtk-3-dev`, `libwebkit2gtk-4.0-dev` (Debian/Ubuntu).

### Instalación del Proyecto

1.  **Clonar el repositorio:**
    ```bash
    git clone https://github.com/tu-usuario/shuyuy.git
    cd shuyuy/shuyuy
    ```

2.  **Instalar dependencias Frontend:**
    ```bash
    cd frontend
    npm install
    ```

3.  **Ejecutar en modo desarrollo:**
    ```bash
    wails dev
    ```
    *   Backend: Puerto aleatorio.
    *   Frontend: Servidor Vite con Hot Reload.

### Compilación (Build)

Para generar binarios de producción:

```bash
# Desde la carpeta shuyuy/
wails build
```
El binario optimizado se generará en `build/bin/`.

### Arquitectura del Sistema

Shuyuy sigue la arquitectura híbrida de Wails:

1.  **Backend (Go):**
    *   `app.go`: Contiene la lógica de negocio y los métodos expuestos al frontend.
    *   `ExportImages`: Utiliza `sync.WaitGroup` para paralelizar el procesamiento de imágenes. Cada imagen se procesa en su propia goroutine para maximizar el uso de CPU multi-núcleo.
    *   `imaging` & `gg`: Librerías robustas para manipulación de píxeles y renderizado de gráficos vectoriales.

2.  **Frontend (Vue 3):**
    *   **Estado Reactivo:** Uso extensivo de `ref` y `computed` para manejar la configuración de cientos de imágenes sin lag.
    *   **Comunicación:** Llamadas asíncronas a Go mediante `window.runtime`.
    *   **UI:** Vuetify 3 con personalización de tema oscuro (`Dark Mode`).

### Solución de Problemas (Troubleshooting)

*   **Error: "gcc" not found**: Asegúrate de tener instalado un compilador C (GCC o MinGW en Windows). Es necesario para compilar las librerías gráficas de Go.
*   **La aplicación no abre en Linux**: Verifica que `webkit2gtk` está instalado.
*   **Imágenes muy grandes (50MP+)**: El procesamiento puede consumir mucha RAM. Shuyuy intenta liberar memoria agresivamente, pero se recomienda tener al menos 8GB de RAM para lotes grandes de alta resolución.

### 🧪 Generación de Datos de Prueba

El proyecto incluye un script de Python para generar un lote de imágenes de prueba con diferentes resoluciones y un logo con transparencia. Esto es útil para probar la clasificación y el rendimiento.

**Uso:**

1.  Asegúrate de tener Python 3 y la librería `Pillow` instalada:
    ```bash
    pip install Pillow
    ```
2.  Ejecuta el script desde la raíz del proyecto:
    ```bash
    python3 test/generar_test.py
    ```
3.  Esto creará la carpeta `test/pruebas_shuyuy/` con 20 imágenes (Landscape y Portrait) y un archivo `logo_test.png`.

---

## 🤝 Contribuir

¡Las contribuciones son bienvenidas!
1.  Haz un Fork del proyecto.
2.  Crea una rama (`git checkout -b feature/nueva-caracteristica`).
3.  Commit tus cambios (`git commit -m 'Añadir nueva característica'`).
4.  Push a la rama (`git push origin feature/nueva-caracteristica`).
5.  Abre un Pull Request.

---

## 📄 Licencia

Este proyecto está bajo la Licencia GNU General Public License v3.0 (GPLv3). Ver el archivo [LICENSE](LICENSE) para más detalles. Esta licencia protege la libertad del software exigiendo que cualquier trabajo derivado sea también de código abierto bajo la misma licencia.
