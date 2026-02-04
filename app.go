package main

import (
	"context"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/disintegration/imaging"
	"github.com/fogleman/gg"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx context.Context
}

type ImageFile struct {
	ID       string `json:"id"`
	Path     string `json:"path"`
	Name     string `json:"name"`
	Size     int64  `json:"size"`     // en Bytes
	Width    int    `json:"width"`
	Height   int    `json:"height"`
	Category string `json:"category"` // "small", "medium", "large"
}

type WatermarkConfig struct {
	Type      string  `json:"type"`      // "text" or "image"
	Content   string  `json:"content"`   // Ruta de la imagen o string del texto
	Opacity   float64 `json:"opacity"`   // 0.0 a 1.0
	Scale     float64 `json:"scale"`     // Escala relativa a la imagen base
	PositionX float64 `json:"positionX"` // 0.0 a 1.0 (Porcentaje horizontal)
	PositionY float64 `json:"positionY"` // 0.0 a 1.0 (Porcentaje vertical)
	Rotation  int     `json:"rotation"`  // Grados

	// New Fields
	Quality    int     `json:"quality"`    // 1-100
	Brightness float64 `json:"brightness"` // -100 to 100
	Contrast   float64 `json:"contrast"`   // -100 to 100
	Saturation float64 `json:"saturation"` // -100 to 100 (Vibrance)
	Sharpness  float64 `json:"sharpness"`  // 0 to 100
	Grayscale  bool    `json:"grayscale"`

	// Text specific
	TextFont  string `json:"textFont"`
	TextColor string `json:"textColor"` // Hex color
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// GetImageBase64 reads a file and returns it as a base64 data URL
func (a *App) GetImageBase64(path string) (string, error) {
	bytes, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	var mimeType string
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".png":
		mimeType = "image/png"
	case ".gif":
		mimeType = "image/gif"
	case ".webp":
		mimeType = "image/webp"
	default:
		mimeType = "image/jpeg"
	}

	base64Str := base64.StdEncoding.EncodeToString(bytes)
	return fmt.Sprintf("data:%s;base64,%s", mimeType, base64Str), nil
}

// SelectDirectory opens a dialog to select a folder and scans it
func (a *App) SelectDirectory() ([]ImageFile, error) {
	selection, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Seleccionar Carpeta de Imágenes",
	})
	if err != nil {
		return nil, err
	}
	if selection == "" {
		return nil, nil // User cancelled
	}

	return a.scanDirectory(selection)
}

// SelectOutputDirectory opens a dialog to select a folder for export
func (a *App) SelectOutputDirectory() (string, error) {
	return runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Seleccionar Carpeta de Destino",
	})
}

// SelectWatermarkFile opens a dialog to select a watermark image
func (a *App) SelectWatermarkFile() (string, error) {
	return runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Seleccionar Marca de Agua",
		Filters: []runtime.FileFilter{
			{DisplayName: "Imágenes", Pattern: "*.jpg;*.jpeg;*.png"},
		},
	})
}

// GetFonts returns a list of available system fonts
func (a *App) GetFonts() []string {
	var fonts []string
	fontDirs := []string{
		"/usr/share/fonts",
		"/usr/local/share/fonts",
		filepath.Join(os.Getenv("HOME"), ".local/share/fonts"),
	}

	for _, dir := range fontDirs {
		filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if !d.IsDir() {
				ext := strings.ToLower(filepath.Ext(path))
				if ext == ".ttf" || ext == ".otf" {
					fonts = append(fonts, path)
				}
			}
			return nil
		})
	}

	// Limit to first 50 to avoid overcrowding
	if len(fonts) > 50 {
		return fonts[:50]
	}
	return fonts
}

// SaveTempWatermark saves a base64 encoded image to a temporary file
func (a *App) SaveTempWatermark(base64Data string) (string, error) {
	// Remove data URL prefix if present
	if idx := strings.Index(base64Data, ","); idx != -1 {
		base64Data = base64Data[idx+1:]
	}

	decoded, err := base64.StdEncoding.DecodeString(base64Data)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64: %w", err)
	}

	tmpFile, err := os.CreateTemp("", "shuyuy_watermark_*.png")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer tmpFile.Close()

	if _, err := tmpFile.Write(decoded); err != nil {
		return "", fmt.Errorf("failed to write to temp file: %w", err)
	}

	return tmpFile.Name(), nil
}

// ExportImages processes and saves images with watermark
func (a *App) ExportImages(images []ImageFile, config WatermarkConfig, outputDir string) error {
	var wg sync.WaitGroup

	// Ensure output dir exists
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return err
	}

	// Prepare watermark source if image
	var wmSrc image.Image
	var err error
	if config.Type == "image" && config.Content != "" {
		wmSrc, err = imaging.Open(config.Content)
		if err != nil {
			return fmt.Errorf("failed to open watermark: %w", err)
		}
	}

	// Default quality
	quality := config.Quality
	if quality <= 0 {
		quality = 85
	}

	for _, imgFile := range images {
		wg.Add(1)
		go func(f ImageFile) {
			defer wg.Done()

			// Open main image
			src, err := imaging.Open(f.Path)
			if err != nil {
				fmt.Printf("Error opening image %s: %v\n", f.Name, err)
				return
			}

			// Apply Filters
			processed := src
			if config.Brightness != 0 {
				processed = imaging.AdjustBrightness(processed, config.Brightness)
			}
			if config.Contrast != 0 {
				processed = imaging.AdjustContrast(processed, config.Contrast)
			}
			if config.Saturation != 0 {
				processed = imaging.AdjustSaturation(processed, config.Saturation)
			}
			if config.Sharpness > 0 {
				processed = imaging.Sharpen(processed, config.Sharpness/20.0) // Normalize factor
			}
			if config.Grayscale {
				processed = imaging.Grayscale(processed)
			}

			finalImage := processed

			if config.Type == "image" && wmSrc != nil {
				// Resize watermark
				targetWmWidth := int(float64(processed.Bounds().Dx()) * config.Scale)
				if targetWmWidth < 1 {
					targetWmWidth = 1
				}

				watermark := imaging.Resize(wmSrc, targetWmWidth, 0, imaging.Lanczos)

				// Calculate position
				posX := int(float64(processed.Bounds().Dx()) * config.PositionX)
				posY := int(float64(processed.Bounds().Dy()) * config.PositionY)

				// Apply Overlay
				finalImage = imaging.Overlay(processed, watermark, image.Pt(posX, posY), config.Opacity)
			} else if config.Type == "text" && config.Content != "" {
				// Use gg to draw text
				dc := gg.NewContextForImage(processed)
				
				// Font size relative to image height
				fontSize := float64(processed.Bounds().Dy()) * config.Scale * 1.5
				if fontSize < 12 { fontSize = 12 }
				
				// Try to load font
				if config.TextFont != "" {
					if err := dc.LoadFontFace(config.TextFont, fontSize); err != nil {
						fmt.Printf("Error loading font %s: %v\n", config.TextFont, err)
					}
				}

				// Color with opacity
				r, g, b := hexToRGB(config.TextColor)
				dc.SetRGBA(r, g, b, config.Opacity)

				// Position
				x := float64(processed.Bounds().Dx()) * config.PositionX
				y := float64(processed.Bounds().Dy()) * config.PositionY
				
				dc.DrawStringAnchored(config.Content, x, y, 0.5, 0.5)
				finalImage = dc.Image()
			}

			// Save
			outPath := filepath.Join(outputDir, f.Name)
			err = imaging.Save(finalImage, outPath, imaging.JPEGQuality(quality))
			if err != nil {
				fmt.Printf("Error saving %s: %v\n", f.Name, err)
			}
		}(imgFile)
	}

	wg.Wait()
	return nil
}


// DeleteImage removes a file from the filesystem
func (a *App) DeleteImage(path string) error {
	return os.Remove(path)
}

func (a *App) scanDirectory(dirPath string) ([]ImageFile, error) {
	var images []ImageFile

	err := filepath.WalkDir(dirPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".jpg" && ext != ".jpeg" && ext != ".png" {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return nil
		}

		// Read dimensions efficiently
		file, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer file.Close()

		config, _, err := image.DecodeConfig(file)
		if err != nil {
			return nil // Not a valid image or corrupted
		}

		// Classification
		category := "small"
		maxSide := config.Width
		if config.Height > maxSide {
			maxSide = config.Height
		}

		if maxSide > 2400 {
			category = "large"
		} else if maxSide >= 1200 {
			category = "medium"
		}

		// ID generation
		hash := md5.Sum([]byte(path))
		id := hex.EncodeToString(hash[:])

		images = append(images, ImageFile{
			ID:       id,
			Path:     path,
			Name:     d.Name(),
			Size:     info.Size(),
			Width:    config.Width,
			Height:   config.Height,
			Category: category,
		})

		return nil
	})

		return images, err

	}

	

	func hexToRGB(h string) (r, g, b float64) {

		if len(h) > 0 && h[0] == '#' {

			h = h[1:]

		}

		if len(h) == 3 {

			h = string([]byte{h[0], h[0], h[1], h[1], h[2], h[2]})

		}

		if len(h) != 6 {

			return 1, 1, 1 // Default white

		}

		

		val, _ := hex.DecodeString(h)

		return float64(val[0]) / 255.0, float64(val[1]) / 255.0, float64(val[2]) / 255.0

	}

	