import os
import random

from PIL import Image, ImageDraw, ImageFont

# Configuración
CARPETA_SALIDA = "pruebas_shuyuy"
CANTIDAD_16_9 = 10  # Cuántas fotos horizontales
CANTIDAD_9_16 = 10  # Cuántas fotos verticales (historias)


def asegurar_directorio():
    if not os.path.exists(CARPETA_SALIDA):
        os.makedirs(CARPETA_SALIDA)
        print(f"📁 Carpeta creada: {CARPETA_SALIDA}")


def color_random():
    return (random.randint(0, 255), random.randint(0, 255), random.randint(0, 255))


def generar_imagen(nombre, ancho, alto, texto_etiqueta):
    """Genera una imagen JPG con ruido visual y texto"""
    img = Image.new("RGB", (ancho, alto), color=color_random())
    draw = ImageDraw.Draw(img)

    # 1. Dibujar figuras random
    for _ in range(20):
        # Generamos 4 coordenadas al azar
        coords_x = [random.randint(0, ancho), random.randint(0, ancho)]
        coords_y = [random.randint(0, alto), random.randint(0, alto)]

        # --- CORRECCIÓN AQUÍ ---
        # Ordenamos: primero el menor, luego el mayor
        x1, x2 = sorted(coords_x)
        y1, y2 = sorted(coords_y)

        # Ahora sí, Pillow recibe [min_x, min_y, max_x, max_y]
        draw.rectangle([x1, y1, x2, y2], fill=color_random(), outline=None)
        draw.ellipse([x1, y1, x2, y2], outline=color_random(), width=5)

    # 2. Poner Texto
    try:
        # Intentamos cargar una fuente grande, si no existe usa default
        # En Linux a veces es DejaVuSans.ttf
        font = ImageFont.truetype("DejaVuSans.ttf", size=100)
    except IOError:
        try:
            font = ImageFont.truetype("arial.ttf", size=100)
        except IOError:
            font = ImageFont.load_default()

    texto = f"{texto_etiqueta}\n{ancho}x{alto}"

    # Calcular centro
    bbox = draw.textbbox((0, 0), texto, font=font)
    tw = bbox[2] - bbox[0]
    th = bbox[3] - bbox[1]

    x = (ancho - tw) / 2
    y = (alto - th) / 2

    # Texto con "sombra" para legibilidad
    draw.text((x + 2, y + 2), texto, fill="black", font=font)
    draw.text((x, y), texto, fill="white", font=font)

    ruta = os.path.join(CARPETA_SALIDA, nombre)
    img.save(ruta, "JPEG", quality=90)
    print(f"✅ Generada: {nombre} ({ancho}x{alto})")


def generar_logo():
    """Genera un logo PNG con transparencia real (RGBA)"""
    ancho, alto = 500, 500
    img = Image.new("RGBA", (ancho, alto), (0, 0, 0, 0))  # Transparente
    draw = ImageDraw.Draw(img)

    draw.ellipse([50, 50, 450, 450], fill=(255, 100, 0, 200))  # Naranja Shuyuy
    draw.ellipse([100, 100, 400, 400], fill=(255, 255, 255, 255))

    try:
        font = ImageFont.truetype("DejaVuSans.ttf", size=80)
    except:
        font = ImageFont.load_default()

    texto = "SHUYUY"
    bbox = draw.textbbox((0, 0), texto, font=font)
    tw = bbox[2] - bbox[0]
    th = bbox[3] - bbox[1]

    draw.text(
        ((ancho - tw) / 2, (alto - th) / 2), texto, fill=(0, 0, 0, 255), font=font
    )

    ruta = os.path.join(CARPETA_SALIDA, "logo_test.png")
    img.save(ruta, "PNG")
    print(f"🎨 Logo generado: logo_test.png")


if __name__ == "__main__":
    asegurar_directorio()

    print("--- Generando Fotos 16:9 ---")
    for i in range(CANTIDAD_16_9):
        generar_imagen(f"foto_landscape_{i+1}.jpg", 1920, 1080, "LANDSCAPE")

    print("\n--- Generando Fotos 9:16 ---")
    for i in range(CANTIDAD_9_16):
        generar_imagen(f"foto_portrait_{i+1}.jpg", 1080, 1920, "PORTRAIT")

    print("\n--- Generando Logo ---")
    generar_logo()

    print(f"\n✨ ¡Listo! Revisa la carpeta '{CARPETA_SALIDA}'")
