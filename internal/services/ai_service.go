package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"time"

	"donationbars/internal/interfaces"
	"donationbars/internal/models"

	"github.com/sashabaranov/go-openai"
)

type AIService struct {
	client  *openai.Client
	timeout time.Duration
}

func NewAIService(apiKey string, timeout time.Duration) interfaces.AIServiceInterface {
	if apiKey == "" || apiKey == "your_openai_api_key_here" {
		slog.Error("OpenAI API key missing or invalid",
			"key_provided", apiKey != "",
			"is_placeholder", apiKey == "your_openai_api_key_here")
		return &AIService{client: nil, timeout: timeout}
	}

	slog.Info("AI Service initialized successfully",
		"timeout", timeout,
		"model", "gpt-4o-mini")

	return &AIService{
		client:  openai.NewClient(apiKey),
		timeout: timeout,
	}
}

// GenerateBar generates a donation bar using AI
func (s *AIService) GenerateBar(req *models.GenerateBarRequest) (*models.AIGenerateResponse, error) {
	if s.client == nil {
		slog.Error("AI service unavailable - client not initialized")
		return nil, errors.New("OpenAI API key eksik veya geçersiz. Lütfen .env dosyasında OPENAI_API_KEY değişkenini ayarlayın")
	}

	slog.Info("Starting AI bar generation",
		"prompt_length", len(req.Prompt),
		"language", req.Language,
		"theme", req.Theme,
		"timeout", s.timeout)

	startTime := time.Now()
	prompt := s.buildEnhancedPrompt(req)

	// Create context with configured timeout
	ctx, cancel := context.WithTimeout(context.Background(), s.timeout)
	defer cancel()

	resp, err := s.client.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model: openai.GPT4oMini,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleUser,
					Content: prompt,
				},
			},
			MaxTokens:   3000, // Increased for better quality
			Temperature: 0.3,  // Lower for more consistent results
		},
	)

	duration := time.Since(startTime)

	if err != nil {
		slog.Error("OpenAI API request failed",
			"error", err.Error(),
			"duration", duration,
			"prompt_length", len(prompt))
		return nil, fmt.Errorf("OpenAI API hatası: %v", err)
	}

	if len(resp.Choices) == 0 {
		slog.Error("OpenAI API returned no choices",
			"duration", duration)
		return nil, errors.New("AI'dan yanıt alınamadı")
	}

	slog.Info("OpenAI API request completed",
		"duration", duration,
		"response_length", len(resp.Choices[0].Message.Content),
		"tokens_used", resp.Usage.TotalTokens)

	// Parse the AI response with enhanced parsing
	content := resp.Choices[0].Message.Content
	result, err := s.parseAIResponseEnhanced(content, req.Language, req.Theme)
	if err != nil {
		slog.Error("Failed to parse AI response",
			"error", err.Error(),
			"response_length", len(content))
		return nil, err
	}

	slog.Info("AI bar generation completed successfully",
		"html_length", len(result.HTML),
		"css_length", len(result.CSS),
		"has_injections", result.Metadata.HasInjections,
		"total_duration", time.Since(startTime))

	return result, nil
}

// buildEnhancedPrompt creates an improved OpenAI prompt with better design guidance
func (s *AIService) buildEnhancedPrompt(req *models.GenerateBarRequest) string {
	var langInstructions string
	var designExamples string

	if req.Language == "tr" {
		langInstructions = "Tüm metinler Türkçe olmalı. Yüzde için '%' sembolü kullan."
		designExamples = "ÖRNEK KALİTELİ TASARIM VE LAYOUT:\n" +
			"- Modern gradyan arka planlar (linear-gradient kullan)\n" +
			"- Yumuşak gölgeler (box-shadow: 0 4px 15px rgba(0,0,0,0.1))\n" +
			"- Rounded köşeler (border-radius: 8px-15px arası)\n" +
			"- İyi tipografi (font-size: 14px-18px arası)\n" +
			"- Progress bar animasyonları (transition: all 0.3s ease)\n" +
			"- Renk uyumu (ana renk + açık/koyu tonları)\n\n" +
			"💡 İDEAL LAYOUT DÜZENİ:\n" +
			"- Üstte: {description} açıklaması (ortalanmış)\n" +
			"- Ortada: Progress bar + merkezi bilgiler\n" +
			"- Progress bar üzerinde: {total} ₺ ve %{percentage}\n" +
			"- Altta sol köşe: Başlangıç tutarı\n" +
			"- Altta sağ köşe: Hedef tutar {goal} ₺\n" +
			"- Position: relative/absolute kullanarak konumlandır\n" +
			"- Center overlay: z-index ile üstte göster"
	} else {
		langInstructions = "All texts should be in English. Use '%' symbol for percentage."
		designExamples = "QUALITY DESIGN EXAMPLES AND LAYOUT:\n" +
			"- Modern gradient backgrounds (use linear-gradient)\n" +
			"- Soft shadows (box-shadow: 0 4px 15px rgba(0,0,0,0.1))\n" +
			"- Rounded corners (border-radius: 8px-15px range)\n" +
			"- Good typography (font-size: 14px-18px range)\n" +
			"- Progress bar animations (transition: all 0.3s ease)\n" +
			"- Color harmony (main color + light/dark variants)\n\n" +
			"💡 IDEAL LAYOUT STRUCTURE:\n" +
			"- Top: {description} text (centered)\n" +
			"- Middle: Progress bar + center info\n" +
			"- On progress bar: {total} ₺, %{percentage} and {remaining} ₺\n" +
			"- Bottom left corner: Starting amount\n" +
			"- Bottom right corner: Goal amount {goal} ₺\n" +
			"- Position: use relative/absolute for positioning\n" +
			"- Center overlay: show on top with z-index\n" +
			"- ⚠️ CRITICAL: ALL 5 INJECTION FIELDS MUST BE PRESENT!"
	}

	progressBarExample := "ÖRNEK LAYOUT YAPISI:\n" +
		"<div class=\"donation-bar\">\n" +
		"  <div class=\"description\">{description}</div>\n" +
		"  <div class=\"progress-section\">\n" +
		"    <div class=\"progress-track\">\n" +
		"      <div class=\"progress-fill\" style=\"width: {percentage}%\"></div>\n" +
		"    </div>\n" +
		"    <div class=\"center-info\">\n" +
		"      <span class=\"amount\">{total} ₺</span>\n" +
		"      <span class=\"percentage\">%{percentage}</span>\n" +
		"      <span class=\"remaining\">Kalan: {remaining} ₺</span>\n" +
		"    </div>\n" +
		"  </div>\n" +
		"  <div class=\"amounts-row\">\n" +
		"    <span class=\"start-amount\">Başlangıç: {total} ₺</span>\n" +
		"    <span class=\"goal-amount\">Hedef: {goal} ₺</span>\n" +
		"  </div>\n" +
		"</div>"

	prompt := "Sen profesyonel bir OBS donation bar tasarımcısısın. YÜKSEK KALİTELİ ve görsel olarak ÇEKİCİ tasarım üret.\n\n" +
		"🎯 ZORUNLU INJECTION ALANLARI (Her biri mutlaka yer almalı):\n" +
		"- {goal}: Hedef bağış tutarı\n" +
		"- {total}: Anlık bağış tutarı\n" +
		"- {percentage}: Toplama oranı (% olmadan sadece sayı, örn: 75)\n" +
		"- {remaining}: Kalan tutar (MUTLAKA ekle!)\n" +
		"- {description}: Bar açıklama metni\n\n" +
		"⚠️ KRİTİK: TÜM 5 INJECTION ALANI MUTLAKA HTML'DE YER ALMALI! {remaining} eksik olursa sistem çalışmaz!\n\n" +
		"📏 BOYUT KISITLAMALARI (KESİNLİKLE uyulmalı):\n" +
		"- width: max 800px (max-width: 800px !important;)\n" +
		"- height: max 200px (max-height: 200px !important;)\n" +
		"- Sabit boyutlu tasarım (px cinsinden değerler)\n" +
		"- @media queries KESİNLİKLE yasak\n" +
		"- Viewport units (vw, vh, vmin, vmax) yasak\n\n" +
		"🚫 TEKNİK YASAKLAR:\n" +
		"- JavaScript/onclick/onload KESİNLİKLE YOK\n" +
		"- Harici kaynaklar YOK (Google Fonts, CDN, http/https URL'ler)\n" +
		"- SVG, iframe, embed, object yasaklı\n" +
		"- expression, behavior, filter, @import yasaklı\n\n" +
		langInstructions + "\n\n" +
		"🎨 TASARIM PRENSİPLERİ:\n" +
		designExamples + "\n\n" +
		"⚡ PROGRESS BAR YAPISI (Mutlaka ekle):\n" +
		progressBarExample + "\n\n" +
		"⚠️ KRİTİK: {percentage} kullanırken tek % kullan! Örnek: width: {percentage}% (çift %% DEĞİL!)\n\n" +
		"📋 KULLANICI İSTEĞİ: \"" + req.Prompt + "\"\n" +
		"🎭 TEMA: \"" + req.Theme + "\"\n\n" +
		"⚠️ MUTLAKA JSON FORMATINDA YANIT VER:\n" +
		"{\n" +
		"  \"html\": \"<div class='donation-bar'>[TAM HTML KOD]</div>\",\n" +
		"  \"css\": \".donation-bar { max-width: 800px; max-height: 200px; [TAM CSS KOD] }\",\n" +
		"  \"metadata\": {\n" +
		"    \"language\": \"" + req.Language + "\",\n" +
		"    \"theme\": \"" + req.Theme + "\",\n" +
		"    \"injection\": true\n" +
		"  }\n" +
		"}\n\n" +
		"🔥 KALİTE KONTROL:\n" +
		"- Progress bar animasyonlu olmalı\n" +
		"- Renk geçişleri yumuşak\n" +
		"- Typography okunabilir\n" +
		"- Modern ve temiz görünüm\n" +
		"- Injection alanları görünür şekilde yerleştirilmiş\n" +
		"- {percentage} sadece tek % ile kullan!\n" +
		"- TÜM 5 INJECTION ALANI MUTLAKA MEVCUT OLMALI: {goal}, {total}, {percentage}, {remaining}, {description}\n\n" +
		"SADECE JSON yanıtı ver, hiç açıklama yapma!"

	return prompt
}

// parseAIResponseEnhanced improved parsing with better JSON extraction
func (s *AIService) parseAIResponseEnhanced(content, language, theme string) (*models.AIGenerateResponse, error) {
	// Clean the content first
	content = strings.TrimSpace(content)

	// DEBUG: Log the raw AI response
	slog.Debug("AI Raw Response (first 800 chars)",
		"response_length", len(content),
		"first_800_chars", content[:min(len(content), 800)])

	// Try to find JSON in various formats
	var jsonContent string

	// Method 1: Smart bracket counting to find complete JSON
	jsonContent = s.extractCompleteJSON(content)
	if jsonContent != "" {
		slog.Debug("Found JSON via Method 1 (smart bracket counting)")
	} else {
		// Method 2: Look for JSON between html and metadata with fixed regex
		jsonRegex := regexp.MustCompile(`(?s)\{\s*"html".*?"metadata"\s*:\s*\{[^}]*\}\s*\}`)
		if matches := jsonRegex.FindString(content); matches != "" {
			jsonContent = matches
			slog.Debug("Found JSON via Method 2 (fixed regex)")
		} else {
			// Method 3: Extract from code blocks
			codeBlockRegex := regexp.MustCompile("(?s)```(?:json)?\\s*(\\{.*?\\})\\s*```")
			if matches := codeBlockRegex.FindStringSubmatch(content); len(matches) > 1 {
				jsonContent = matches[1]
				slog.Debug("Found JSON via Method 3 (code blocks)")
			} else {
				// Method 4: Find any JSON-like structure and complete it
				anyJsonRegex := regexp.MustCompile(`(?s)\{[^{}]*"html"[^{}]*"css".*`)
				if matches := anyJsonRegex.FindString(content); matches != "" {
					jsonContent = s.completeIncompleteJSON(matches)
					slog.Debug("Found JSON via Method 4 (completed incomplete)")
				}
			}
		}
	}

	if jsonContent == "" {
		slog.Error("No JSON found in AI response",
			"full_content_length", len(content))
		return nil, errors.New("AI yanıtında geçerli JSON bulunamadı")
	}

	// DEBUG: Log the extracted JSON
	slog.Debug("Extracted JSON content",
		"json_length", len(jsonContent))

	// Validate JSON before parsing
	if !s.isValidJSON(jsonContent) {
		slog.Warn("JSON is invalid, attempting to fix...",
			"json_length", len(jsonContent))
		jsonContent = s.fixBrokenJSON(jsonContent)
		slog.Debug("Fixed JSON",
			"fixed_json_length", len(jsonContent))
	}

	var response models.AIGenerateResponse
	err := json.Unmarshal([]byte(jsonContent), &response)
	if err != nil {
		slog.Error("JSON parse error",
			"error", err.Error(),
			"json_length", len(jsonContent))
		slog.Error("Problematic JSON",
			"json_content", jsonContent)
		return nil, fmt.Errorf("JSON parse hatası: %v", err)
	}

	slog.Debug("JSON parsed successfully!")

	// Clean and validate the HTML/CSS
	response.HTML = s.cleanAndValidateHTML(response.HTML)
	response.CSS = s.cleanAndValidateCSS(response.CSS)

	// Enhanced validation
	if !s.validateAIResponseEnhanced(&response) {
		return nil, errors.New("AI yanıtı kalite kontrollerini geçemedi")
	}

	// Set metadata if not properly set
	if response.Metadata.Language == "" {
		response.Metadata.Language = language
	}
	if response.Metadata.Theme == "" {
		response.Metadata.Theme = theme
	}
	response.Metadata.HasInjections = s.validateInjections(response.HTML)

	return &response, nil
}

// extractCompleteJSON uses bracket counting to extract complete JSON
func (s *AIService) extractCompleteJSON(content string) string {
	start := strings.Index(content, "{")
	if start == -1 {
		return ""
	}

	bracketCount := 0
	inString := false
	escape := false

	for i := start; i < len(content); i++ {
		char := content[i]

		if escape {
			escape = false
			continue
		}

		if escape {
			escape = false
			continue
		}

		switch char {
		case '\\':
			escape = true
			continue
		case '"':
			inString = !inString
			continue
		case '{':
			if !inString {
				bracketCount++
			}
		case '}':
			if !inString {
				bracketCount--
				if bracketCount == 0 {
					return content[start : i+1]
				}
			}
		}
	}

	return ""
}

// completeIncompleteJSON attempts to complete a broken JSON
func (s *AIService) completeIncompleteJSON(jsonStr string) string {
	// Remove trailing content that might break JSON
	jsonStr = strings.TrimSpace(jsonStr)

	// Count open and close brackets
	openBrackets := strings.Count(jsonStr, "{")
	closeBrackets := strings.Count(jsonStr, "}")

	// Add missing closing brackets
	for i := closeBrackets; i < openBrackets; i++ {
		jsonStr += "}"
	}

	return jsonStr
}

// isValidJSON checks if a string is valid JSON
func (s *AIService) isValidJSON(jsonStr string) bool {
	var temp interface{}
	return json.Unmarshal([]byte(jsonStr), &temp) == nil
}

// fixBrokenJSON attempts to fix common JSON issues
func (s *AIService) fixBrokenJSON(jsonStr string) string {
	// Remove trailing commas
	jsonStr = regexp.MustCompile(`,(\s*[}\]])`).ReplaceAllString(jsonStr, "$1")

	// Fix incomplete quotes
	lines := strings.Split(jsonStr, "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, `"`) && !strings.HasSuffix(trimmed, `"`) && !strings.HasSuffix(trimmed, `",`) {
			lines[i] = line + `"`
		}
	}
	jsonStr = strings.Join(lines, "\n")

	// Ensure proper closing
	jsonStr = s.completeIncompleteJSON(jsonStr)

	return jsonStr
}

// Helper function for min
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// cleanAndValidateHTML cleans and validates HTML content
func (s *AIService) cleanAndValidateHTML(html string) string {
	// Remove any potential script tags or dangerous content
	html = regexp.MustCompile(`(?i)<script[^>]*>.*?</script>`).ReplaceAllString(html, "")
	html = regexp.MustCompile(`(?i)javascript:`).ReplaceAllString(html, "")
	html = regexp.MustCompile(`(?i)on\w+\s*=`).ReplaceAllString(html, "data-removed=")

	// Ensure proper structure
	if !strings.Contains(html, "donation-bar") {
		html = fmt.Sprintf(`<div class="donation-bar">%s</div>`, html)
	}

	return strings.TrimSpace(html)
}

// cleanAndValidateCSS cleans and validates CSS content
func (s *AIService) cleanAndValidateCSS(css string) string {
	// Remove dangerous CSS properties
	css = regexp.MustCompile(`(?i)expression\([^)]*\)`).ReplaceAllString(css, "")
	css = regexp.MustCompile(`(?i)@import[^;]*;`).ReplaceAllString(css, "")
	css = regexp.MustCompile(`(?i)url\(http[^)]*\)`).ReplaceAllString(css, "")

	// Ensure size constraints are present
	if !strings.Contains(css, "max-width") {
		css = ".donation-bar { max-width: 800px !important; " + css
	}
	if !strings.Contains(css, "max-height") {
		css = strings.Replace(css, "max-width:", "max-height: 200px !important; max-width:", 1)
	}

	return strings.TrimSpace(css)
}

// validateAIResponseEnhanced enhanced validation with stricter checks
func (s *AIService) validateAIResponseEnhanced(response *models.AIGenerateResponse) bool {
	// Check for required injections
	if !s.validateInjections(response.HTML) {
		slog.Error("Injection validation failed")
		return false
	}

	// Enhanced CSS size constraints check
	if !s.validateCSSSizeConstraintsStrict(response.CSS) {
		slog.Error("CSS size constraints validation failed")
		return false
	}

	// Check for forbidden content with more patterns
	forbidden := []string{
		"javascript:", "<script", "expression(", "behavior:", "@import",
		"url(http", "url(//", "<iframe", "<embed", "<object",
		"@media", "vw", "vh", "vmin", "vmax", "onclick", "onload",
		"onfocus", "onmouseover", "eval(", "document.",
	}

	content := strings.ToLower(response.HTML + response.CSS)
	for _, forbidden := range forbidden {
		if strings.Contains(content, forbidden) {
			slog.Error("Forbidden content found",
				"forbidden_string", forbidden)
			return false
		}
	}

	// Check minimum quality requirements
	if !s.validateDesignQuality(response.HTML, response.CSS) {
		slog.Error("Design quality validation failed")
		return false
	}

	// Size limits
	if len(response.HTML) > 15000 || len(response.CSS) > 15000 {
		slog.Error("Content too large",
			"html_length", len(response.HTML),
			"css_length", len(response.CSS))
		return false
	}

	return true
}

// validateCSSSizeConstraintsStrict stricter CSS size validation
func (s *AIService) validateCSSSizeConstraintsStrict(css string) bool {
	cssLower := strings.ToLower(css)

	// Must have max-width constraint
	hasMaxWidth := strings.Contains(cssLower, "max-width") &&
		(strings.Contains(cssLower, "800px") || strings.Contains(cssLower, "800"))

	// Must have max-height constraint
	hasMaxHeight := strings.Contains(cssLower, "max-height") &&
		(strings.Contains(cssLower, "200px") || strings.Contains(cssLower, "200"))

	// Check for forbidden units
	forbiddenUnits := []string{"vw", "vh", "vmin", "vmax", "%"}
	for _, unit := range forbiddenUnits {
		if strings.Contains(cssLower, unit) && !strings.Contains(cssLower, "100%") {
			slog.Warn("Forbidden unit found",
				"forbidden_unit", unit)
			return false
		}
	}

	return hasMaxWidth && hasMaxHeight
}

// validateDesignQuality checks for basic design quality indicators
func (s *AIService) validateDesignQuality(html, css string) bool {
	htmlLower := strings.ToLower(html)
	cssLower := strings.ToLower(css)

	// Check for progress bar structure
	hasProgressBar := strings.Contains(htmlLower, "progress") &&
		(strings.Contains(htmlLower, "bar") || strings.Contains(cssLower, "progress"))

	// Check for modern CSS features
	hasModernCSS := strings.Contains(cssLower, "border-radius") ||
		strings.Contains(cssLower, "box-shadow") ||
		strings.Contains(cssLower, "gradient")

	// Check for proper styling
	hasBasicStyling := strings.Contains(cssLower, "background") &&
		strings.Contains(cssLower, "color")

	return hasProgressBar && (hasModernCSS || hasBasicStyling)
}

// validateInjections checks if all required injection fields are present
func (s *AIService) validateInjections(html string) bool {
	for _, injection := range models.RequiredInjections {
		if !strings.Contains(html, injection) {
			slog.Error("Missing injection",
				"injection_string", injection)
			return false
		}
	}
	return true
}
