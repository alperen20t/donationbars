package handlers

import (
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"
	"time"

	"donationbars/internal/interfaces"
	"donationbars/internal/models"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	barService interfaces.BarServiceInterface
	aiService  interfaces.AIServiceInterface
	tmpl       *template.Template
}

func New(barService interfaces.BarServiceInterface, aiService interfaces.AIServiceInterface) *Handler {
	// Load HTML templates
	tmpl := template.Must(template.ParseGlob("templates/*.html"))

	return &Handler{
		barService: barService,
		aiService:  aiService,
		tmpl:       tmpl,
	}
}

// Server-Side Rendering Handlers

// HomePage renders the main page
func (h *Handler) HomePage(c *gin.Context) {
	userID := c.GetHeader("X-User-ID")
	if userID == "" {
		userID = "test-user"
	}

	bars, err := h.barService.GetUserBars(userID)
	if err != nil {
		bars = []*models.DonationBar{}
	}

	// Check if user wants to show only active bars
	showOnlyActive := c.Query("show_only_active") == "true"

	activeBars := 0
	var filteredBars []*models.DonationBar

	for _, bar := range bars {
		if bar.IsActive {
			activeBars++
		}

		// Filter bars based on user preference
		if showOnlyActive {
			if bar.IsActive {
				filteredBars = append(filteredBars, bar)
			}
		} else {
			filteredBars = append(filteredBars, bar)
		}
	}

	data := gin.H{
		"Title":          "Donation Bars - AI Powered OBS Bar Designer",
		"UserID":         userID,
		"Bars":           filteredBars,
		"TotalBars":      len(bars), // Total count should show all bars
		"ActiveBars":     activeBars,
		"MaxBars":        5,
		"ShowOnlyActive": showOnlyActive,
	}

	// Handle success/error messages from URL query parameters
	if success := c.Query("success"); success != "" {
		data["Success"] = success
	}
	if errorMsg := c.Query("error"); errorMsg != "" {
		data["Error"] = errorMsg
	}

	c.HTML(http.StatusOK, "index.html", data)
}

// CreatePage renders the bar creation page
func (h *Handler) CreatePage(c *gin.Context) {
	mode := c.Query("mode") // ai or manual

	data := gin.H{
		"Title": "Yeni Bar Oluştur - Donation Bars",
		"Mode":  mode,
	}

	c.HTML(http.StatusOK, "create.html", data)
}

// ManagePage renders the bar management page
func (h *Handler) ManagePage(c *gin.Context) {
	userID := c.GetHeader("X-User-ID")
	if userID == "" {
		userID = "test-user"
	}

	bars, err := h.barService.GetUserBars(userID)
	if err != nil {
		bars = []*models.DonationBar{}
	}

	activeBars := 0
	for _, bar := range bars {
		if bar.IsActive {
			activeBars++
		}
	}

	data := gin.H{
		"Title":      "Bar Yönetimi - Donation Bars",
		"Bars":       bars,
		"TotalBars":  len(bars),
		"ActiveBars": activeBars,
	}

	// Handle success/error messages from URL query parameters
	if success := c.Query("success"); success != "" {
		data["Success"] = success
	}
	if errorMsg := c.Query("error"); errorMsg != "" {
		data["Error"] = errorMsg
	}

	c.HTML(http.StatusOK, "manage.html", data)
}

// Form Handlers

// CreateBarForm handles manual bar creation form
func (h *Handler) CreateBarForm(c *gin.Context) {
	var req models.CreateBarRequest
	if err := c.ShouldBind(&req); err != nil {
		c.HTML(http.StatusBadRequest, "create.html", gin.H{
			"Title": "Yeni Bar Oluştur - Donation Bars",
			"Error": err.Error(),
		})
		return
	}

	userID := c.GetHeader("X-User-ID")
	if userID == "" {
		userID = "test-user"
	}

	bar, err := h.barService.CreateBar(userID, &req)
	if err != nil {
		c.HTML(http.StatusBadRequest, "create.html", gin.H{
			"Title": "Yeni Bar Oluştur - Donation Bars",
			"Error": err.Error(),
		})
		return
	}

	c.Redirect(http.StatusFound, "/?success=Bar başarıyla oluşturuldu&id="+bar.ID.Hex())
}

// CreateBarAIForm handles AI bar creation form submission
func (h *Handler) CreateBarAIForm(c *gin.Context) {
	// Use ShouldBind to properly handle form data with validation
	var req models.GenerateBarRequest
	if err := c.ShouldBind(&req); err != nil {
		c.HTML(http.StatusBadRequest, "create.html", gin.H{
			"Title": "Yeni Bar Oluştur - Donation Bars",
			"Mode":  "ai",
			"Error": err.Error(),
		})
		return
	}

	userID := c.GetHeader("X-User-ID")
	if userID == "" {
		userID = "test-user"
	}

	// Check daily rate limit (cursorrules.rules: 5 bars/day)
	dailyCount, err := h.barService.GetUserDailyBarCount(userID)
	if err == nil && dailyCount >= 5 {
		c.HTML(http.StatusBadRequest, "create.html", gin.H{
			"Title": "Yeni Bar Oluştur - Donation Bars",
			"Mode":  "ai",
			"Error": "Günlük bar oluşturma limitine ulaştınız (5 bar/gün). Yarın tekrar deneyebilirsiniz.",
		})
		return
	}

	// Generate with AI
	aiResponse, err := h.aiService.GenerateBar(&req)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "create.html", gin.H{
			"Title": "Yeni Bar Oluştur - Donation Bars",
			"Mode":  "ai",
			"Error": "AI ile bar oluşturulurken hata oluştu: " + err.Error(),
		})
		return
	}

	// Create preview HTML with sample data AND embedded CSS using actual bar amounts
	previewHTML := aiResponse.HTML
	goalValue := fmt.Sprintf("%.0f", req.GoalAmount)
	totalValue := fmt.Sprintf("%.0f", req.InitialAmount)
	var percentageValue, remainingValue string
	if req.GoalAmount > 0 {
		percentage := (req.InitialAmount / req.GoalAmount) * 100
		remaining := req.GoalAmount - req.InitialAmount
		percentageValue = fmt.Sprintf("%.0f", percentage)
		remainingValue = fmt.Sprintf("%.0f", remaining)
	} else {
		percentageValue = "0"
		remainingValue = goalValue
	}

	previewHTML = strings.Replace(previewHTML, "{goal}", goalValue, -1)
	previewHTML = strings.Replace(previewHTML, "{total}", totalValue, -1)
	previewHTML = strings.Replace(previewHTML, "{percentage}", percentageValue, -1)
	previewHTML = strings.Replace(previewHTML, "{remaining}", remainingValue, -1)
	previewHTML = strings.Replace(previewHTML, "{description}", "Oyun geliştirme için bağış kampanyası", -1)

	// Fix double percentage issue (XX%% -> XX%)
	previewHTML = strings.Replace(previewHTML, percentageValue+"%%", percentageValue+"%", -1)

	// Create complete preview with embedded CSS for proper rendering
	completePreviewHTML := `<style>` + aiResponse.CSS + `</style>` + previewHTML

	// Show AI result page
	c.HTML(http.StatusOK, "ai_result.html", gin.H{
		"Title":         "AI Bar Sonucu - Donation Bars",
		"HTML":          template.HTML(aiResponse.HTML),
		"CSS":           template.HTML(aiResponse.CSS),
		"PreviewHTML":   template.HTML(completePreviewHTML),
		"RawHTML":       aiResponse.HTML,
		"RawCSS":        aiResponse.CSS,
		"Prompt":        req.Prompt,
		"Language":      req.Language,
		"Theme":         req.Theme,
		"InitialAmount": req.InitialAmount,
		"GoalAmount":    req.GoalAmount,
		"CreatedAt":     time.Now().Format("02.01.2006 15:04"),
	})
}

// SaveAIBarForm handles saving AI generated bar
func (h *Handler) SaveAIBarForm(c *gin.Context) {
	prompt := c.PostForm("prompt")
	language := c.PostForm("language")
	theme := c.PostForm("theme")
	html := c.PostForm("html")
	css := c.PostForm("css")
	initialAmountStr := c.PostForm("initial_amount")
	goalAmountStr := c.PostForm("goal_amount")

	// Validate required fields
	if prompt == "" || html == "" || css == "" {
		c.Redirect(http.StatusFound, "/?error=Geçersiz bar verisi")
		return
	}

	// Parse amounts
	initialAmount := 0.0
	goalAmount := 1000.0 // default

	if initialAmountStr != "" {
		if parsed, err := strconv.ParseFloat(initialAmountStr, 64); err == nil {
			initialAmount = parsed
		}
	}

	if goalAmountStr != "" {
		if parsed, err := strconv.ParseFloat(goalAmountStr, 64); err == nil && parsed > 0 {
			goalAmount = parsed
		}
	}

	userID := c.GetHeader("X-User-ID")
	if userID == "" {
		userID = "test-user"
	}

	// Create AI response object
	aiResponse := &models.AIGenerateResponse{
		HTML: html,
		CSS:  css,
		Metadata: models.AIGenerateMetadata{
			Language:      language,
			Theme:         theme,
			HasInjections: true, // AI always includes injections
		},
	}

	// Save to database using existing service with amounts
	bar, err := h.barService.CreateBarFromAI(userID, prompt, aiResponse, initialAmount, goalAmount)
	if err != nil {
		c.Redirect(http.StatusFound, "/?error="+err.Error())
		return
	}

	c.Redirect(http.StatusFound, "/?success=AI Bar başarıyla kaydedildi: "+bar.Name)
}

// EditPage renders the bar edit page
func (h *Handler) EditPage(c *gin.Context) {
	userID := c.GetHeader("X-User-ID")
	if userID == "" {
		userID = "test-user"
	}

	barID := c.Param("id")
	bar, err := h.barService.GetBar(userID, barID)
	if err != nil {
		c.HTML(http.StatusNotFound, "error.html", gin.H{
			"Title": "Bar Bulunamadı",
			"Error": err.Error(),
		})
		return
	}

	data := gin.H{
		"Title": "Bar Düzenle - " + bar.Name,
		"Bar":   bar,
	}

	// Handle success/error messages from URL query parameters
	if success := c.Query("success"); success != "" {
		data["Success"] = success
	}
	if errorMsg := c.Query("error"); errorMsg != "" {
		data["Error"] = errorMsg
	}

	c.HTML(http.StatusOK, "edit.html", data)
}

// EditBarForm handles bar editing form submission
func (h *Handler) EditBarForm(c *gin.Context) {
	userID := c.GetHeader("X-User-ID")
	if userID == "" {
		userID = "test-user"
	}

	barID := c.Param("id")

	// Get existing bar to check if it's AI generated
	existingBar, err := h.barService.GetBar(userID, barID)
	if err != nil {
		c.Redirect(http.StatusFound, "/manage?error="+err.Error())
		return
	}

	name := c.PostForm("name")
	description := c.PostForm("description")
	language := c.PostForm("language")
	theme := c.PostForm("theme")
	isActiveStr := c.PostForm("is_active")
	initialAmountStr := c.PostForm("initial_amount")
	goalAmountStr := c.PostForm("goal_amount")
	isActive := isActiveStr == "true"

	// Validate required fields
	if name == "" {
		c.Redirect(http.StatusFound, "/edit/"+barID+"?error=Bar adı zorunludur")
		return
	}

	if len(name) > 100 {
		c.Redirect(http.StatusFound, "/edit/"+barID+"?error=Bar adı en fazla 100 karakter olabilir")
		return
	}

	if language != "tr" && language != "en" {
		c.Redirect(http.StatusFound, "/edit/"+barID+"?error=Geçerli bir dil seçmelisiniz")
		return
	}

	// Parse amounts
	var initialAmount, goalAmount *float64
	var initialAmountValue, goalAmountValue float64 = 0.0, 1000.0 // defaults

	if initialAmountStr != "" {
		if parsed, err := strconv.ParseFloat(initialAmountStr, 64); err == nil && parsed >= 0 {
			initialAmount = &parsed
			initialAmountValue = parsed
		} else {
			c.Redirect(http.StatusFound, "/edit/"+barID+"?error=Geçersiz başlangıç tutarı")
			return
		}
	}

	if goalAmountStr != "" {
		if parsed, err := strconv.ParseFloat(goalAmountStr, 64); err == nil && parsed > 0 {
			goalAmount = &parsed
			goalAmountValue = parsed
		} else {
			c.Redirect(http.StatusFound, "/edit/"+barID+"?error=Geçersiz hedef tutarı")
			return
		}
	}

	// Build update request
	updateReq := &models.UpdateBarRequest{
		Name:          &name,
		Description:   &description,
		IsActive:      &isActive,
		InitialAmount: initialAmount,
		GoalAmount:    goalAmount,
	}

	// For non-AI generated bars, allow HTML/CSS editing
	if !existingBar.AIGenerated {
		html := c.PostForm("html")
		css := c.PostForm("css")

		if html == "" || css == "" {
			c.Redirect(http.StatusFound, "/edit/"+barID+"?error=HTML ve CSS kodları zorunludur")
			return
		}

		// Update the bar service to handle HTML/CSS updates
		// For now, we'll use a workaround by using the CreateBarRequest struct
		fullUpdateReq := &models.CreateBarRequest{
			Name:          name,
			Description:   description,
			HTML:          html,
			CSS:           css,
			Language:      language,
			Theme:         theme,
			InitialAmount: initialAmountValue,
			GoalAmount:    goalAmountValue,
		}

		// Call a new update method that handles HTML/CSS
		err = h.barService.UpdateBarComplete(userID, barID, fullUpdateReq, isActive)
		if err != nil {
			c.Redirect(http.StatusFound, "/edit/"+barID+"?error="+err.Error())
			return
		}
	} else {
		// For AI generated bars, only update basic fields
		_, err = h.barService.UpdateBar(userID, barID, updateReq)
		if err != nil {
			c.Redirect(http.StatusFound, "/edit/"+barID+"?error="+err.Error())
			return
		}
	}

	c.Redirect(http.StatusFound, "/edit/"+barID+"?success=Bar başarıyla güncellendi")
}

// ToggleBarStatus toggles bar active status
func (h *Handler) ToggleBarStatus(c *gin.Context) {
	userID := c.GetHeader("X-User-ID")
	if userID == "" {
		userID = "test-user"
	}

	barID := c.Param("id")
	isActive := c.PostForm("is_active") == "true"

	req := models.UpdateBarRequest{
		IsActive: &isActive,
	}

	_, err := h.barService.UpdateBar(userID, barID, &req)
	if err != nil {
		c.Redirect(http.StatusFound, "/manage?error="+err.Error())
		return
	}

	c.Redirect(http.StatusFound, "/manage?success=Bar durumu güncellendi")
}

// DeleteBarForm deletes a bar
func (h *Handler) DeleteBarForm(c *gin.Context) {
	userID := c.GetHeader("X-User-ID")
	if userID == "" {
		userID = "test-user"
	}

	barID := c.Param("id")
	err := h.barService.DeleteBar(userID, barID)
	if err != nil {
		c.Redirect(http.StatusFound, "/manage?error="+err.Error())
		return
	}

	c.Redirect(http.StatusFound, "/manage?success=Bar başarıyla silindi")
}

// PreviewBar renders a bar preview
func (h *Handler) PreviewBar(c *gin.Context) {
	userID := c.GetHeader("X-User-ID")
	if userID == "" {
		userID = "test-user"
	}

	barID := c.Param("id")
	bar, err := h.barService.GetBar(userID, barID)
	if err != nil {
		c.HTML(http.StatusNotFound, "error.html", gin.H{
			"Title": "Bar Bulunamadı",
			"Error": err.Error(),
		})
		return
	}

	// Use bar's actual data or sample data for preview
	goalValue := fmt.Sprintf("%.0f", bar.GoalAmount)
	totalValue := fmt.Sprintf("%.0f", bar.InitialAmount)
	var percentageValue, remainingValue string
	if bar.GoalAmount > 0 {
		percentage := (bar.InitialAmount / bar.GoalAmount) * 100
		remaining := bar.GoalAmount - bar.InitialAmount
		percentageValue = fmt.Sprintf("%.0f", percentage)
		remainingValue = fmt.Sprintf("%.0f", remaining)
	} else {
		percentageValue = "0"
		remainingValue = goalValue
	}
	descriptionValue := "Oyun geliştirme için bağış kampanyası"

	// Replace injection fields in HTML
	html := bar.HTML
	html = strings.Replace(html, "{goal}", goalValue, -1)
	html = strings.Replace(html, "{total}", totalValue, -1)
	html = strings.Replace(html, "{percentage}", percentageValue, -1)
	html = strings.Replace(html, "{remaining}", remainingValue, -1)
	html = strings.Replace(html, "{description}", descriptionValue, -1)

	// Fix double percentage issue (75%% -> 75%)
	html = strings.Replace(html, percentageValue+"%%", percentageValue+"%", -1)

	// Create preview HTML with embedded CSS
	previewHTML := `<!DOCTYPE html>
<html lang="tr">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Bar Önizleme - ` + bar.Name + `</title>
    <style>
        body {
            margin: 0;
            padding: 20px;
            background: #2b2b2b;
            font-family: Arial, sans-serif;
            color: white;
        }
        
        .preview-container {
            max-width: 1000px;
            margin: 0 auto;
        }
        
        .preview-header {
            background: #333;
            padding: 15px;
            border-radius: 10px;
            margin-bottom: 20px;
            text-align: center;
        }
        
        .preview-area {
            background: #1a1a1a;
            padding: 20px;
            border-radius: 10px;
            border: 2px dashed #555;
            text-align: center;
        }
        
        .preview-info {
            margin-bottom: 20px;
            background: #333;
            padding: 10px;
            border-radius: 5px;
            font-size: 12px;
        }
        
        /* Bar CSS */
        ` + bar.CSS + `
    </style>
</head>
<body>
    <div class="preview-container">
        <div class="preview-header">
            <h1>📺 OBS Donation Bar Önizlemesi</h1>
            <p>Bar: <strong>` + bar.Name + `</strong> | `

	if bar.AIGenerated {
		previewHTML += `🤖 AI ile Oluşturuldu`
	} else {
		previewHTML += `✏️ Manuel Oluşturuldu`
	}

	previewHTML += ` | 🌐 `
	if bar.Language == "tr" {
		previewHTML += `Türkçe`
	} else {
		previewHTML += `English`
	}

	previewHTML += `</p>
        </div>
        
        <div class="preview-info">
            <strong>ℹ️ Önizleme Bilgisi:</strong> Bu önizlemede gerçek bağış verileri simüle edilmiştir. 
            OBS'de kullanırken {goal}, {total}, {percentage}, {remaining}, {description} alanları 
            gerçek verilerle otomatik doldurulacaktır.
        </div>
        
        <div class="preview-area">
            ` + html + `
        </div>
        
        <div class="preview-info">
            <strong>📋 Injection Alanları:</strong><br>
            • {goal} → ` + goalValue + ` ₺<br>
            • {total} → ` + totalValue + ` ₺<br>
            • {percentage} → ` + percentageValue + `%<br>
            • {remaining} → ` + remainingValue + ` ₺<br>
            • {description} → "` + descriptionValue + `"
        </div>
        
        <div style="text-align: center; margin-top: 20px;">
            <button onclick="window.close()" style="padding: 10px 20px; background: #007bff; color: white; border: none; border-radius: 5px; cursor: pointer;">
                🔙 Kapat
            </button>
        </div>
    </div>
</body>
</html>`

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, previewHTML)
}

// API Handlers (for backward compatibility)

// CreateBar creates a new donation bar (API)
func (h *Handler) CreateBar(c *gin.Context) {
	var req models.CreateBarRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.GetHeader("X-User-ID")
	if userID == "" {
		userID = "test-user"
	}

	bar, err := h.barService.CreateBar(userID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    bar,
	})
}

// GetUserBars returns all bars for the authenticated user (API)
func (h *Handler) GetUserBars(c *gin.Context) {
	userID := c.GetHeader("X-User-ID")
	if userID == "" {
		userID = "test-user"
	}

	bars, err := h.barService.GetUserBars(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    bars,
	})
}

// GetBar returns a specific bar (API)
func (h *Handler) GetBar(c *gin.Context) {
	userID := c.GetHeader("X-User-ID")
	if userID == "" {
		userID = "test-user"
	}

	barID := c.Param("id")
	bar, err := h.barService.GetBar(userID, barID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    bar,
	})
}

// UpdateBar updates a bar (API)
func (h *Handler) UpdateBar(c *gin.Context) {
	userID := c.GetHeader("X-User-ID")
	if userID == "" {
		userID = "test-user"
	}

	barID := c.Param("id")

	var req models.UpdateBarRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	bar, err := h.barService.UpdateBar(userID, barID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    bar,
	})
}

// DeleteBar deletes a bar (API)
func (h *Handler) DeleteBar(c *gin.Context) {
	userID := c.GetHeader("X-User-ID")
	if userID == "" {
		userID = "test-user"
	}

	barID := c.Param("id")
	err := h.barService.DeleteBar(userID, barID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Bar başarıyla silindi",
	})
}

// GenerateBarWithAI generates a new bar using AI (API)
func (h *Handler) GenerateBarWithAI(c *gin.Context) {
	var req models.GenerateBarRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.GetHeader("X-User-ID")
	if userID == "" {
		userID = "test-user"
	}

	// Generate with AI
	aiResponse, err := h.aiService.GenerateBar(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Save to database
	bar, err := h.barService.CreateBarFromAI(userID, req.Prompt, aiResponse, req.InitialAmount, req.GoalAmount)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success":      true,
		"data":         bar,
		"ai_generated": true,
	})
}
