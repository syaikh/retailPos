package audit

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode"

	"retail-pos-system/internal/config"
	"retail-pos-system/internal/shared"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup, auth gin.HandlerFunc, perm func(string) gin.HandlerFunc) {
	r.GET("/audit-logs", auth, perm("audit:read"), h.ListAuditLogs)
	r.GET("/audit-logs/export", auth, perm("audit:read"), h.ExportAuditLogs)
	r.GET("/audit-logs/entity-types", auth, perm("audit:read"), h.ListEntityTypes)
}

func (h *Handler) ListAuditLogs(c *gin.Context) {
	limit := 50
	if l := c.Query("limit"); l != "" {
		if val, err := strconv.Atoi(l); err == nil && val > 0 && val <= 100000 {
			limit = val
		}
	}
	offset := 0
	if o := c.Query("offset"); o != "" {
		if val, err := strconv.Atoi(o); err == nil && val >= 0 {
			offset = val
		}
	}

	var userID *int
	if uid := c.Query("user_id"); uid != "" {
		if val, err := strconv.Atoi(uid); err == nil {
			userID = &val
		}
	}

	search := c.Query("search")
	action := c.Query("action")
	entityType := c.Query("entity_type")

	startDate := parseDateParam(c.Query("start_date"))
	endDate := parseDateParam(c.Query("end_date"))

	logs, total, err := h.svc.GetAuditLogs(c.Request.Context(), limit, offset, userID, search, action, entityType, startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch audit logs"})
		return
	}

	if logs == nil {
		logs = []AuditLog{}
	}

	c.JSON(http.StatusOK, gin.H{"data": logs, "total": total})
}

func (h *Handler) ListEntityTypes(c *gin.Context) {
	types, err := h.svc.GetEntityTypes(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch entity types"})
		return
	}
	if types == nil {
		types = []string{}
	}
	c.JSON(http.StatusOK, gin.H{"data": types})
}

func (h *Handler) ExportAuditLogs(c *gin.Context) {
	format := c.Query("format")
	search := c.Query("search")
	action := c.Query("action")
	entityType := c.Query("entity_type")

	var userID *int
	if uid := c.Query("user_id"); uid != "" {
		if val, err := strconv.Atoi(uid); err == nil {
			userID = &val
		}
	}

	startDate := parseDateParam(c.Query("start_date"))
	endDate := parseDateParam(c.Query("end_date"))

	logs, _, err := h.svc.GetAuditLogs(c.Request.Context(), 100000, 0, userID, search, action, entityType, startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch audit logs"})
		return
	}

	if logs == nil {
		logs = []AuditLog{}
	}

	cfg := config.Load()
	now := time.Now().In(cfg.Timezone)
	filename := "audit-logs-" + now.Format("2006-01-02")

	switch format {
	case "xlsx":
		wb := excelize.NewFile()
		sheet := "Audit Logs"
		_ = wb.SetSheetName("Sheet1", sheet)

		headers := []string{"Timestamp", "Actor", "Role", "Action", "Resource", "Description", "IP Address"}
		for i, hdr := range headers {
			col, _ := excelize.ColumnNumberToName(i + 1)
			_ = wb.SetCellValue(sheet, col+"1", hdr)
		}
		headerStyle, _ := wb.NewStyle(&excelize.Style{
			Font: &excelize.Font{Bold: true, Color: "FFFFFF"},
			Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"7C3AED"}},
		})
		_ = wb.SetCellStyle(sheet, "A1", "G1", headerStyle)

		for i, log := range logs {
			r := i + 2
			t := log.CreatedAt
			if parsed, err := time.Parse(time.RFC3339, t); err == nil {
				t = parsed.In(cfg.Timezone).Format("2006-01-02 15:04:05")
			}
			_ = wb.SetCellValue(sheet, fmt.Sprintf("A%d", r), t)
			_ = wb.SetCellValue(sheet, fmt.Sprintf("B%d", r), log.Username)
			_ = wb.SetCellValue(sheet, fmt.Sprintf("C%d", r), log.Role)
			_ = wb.SetCellValue(sheet, fmt.Sprintf("D%d", r), log.Action)
			_ = wb.SetCellValue(sheet, fmt.Sprintf("E%d", r), log.EntityType)
			_ = wb.SetCellValue(sheet, fmt.Sprintf("F%d", r), log.Description)
			_ = wb.SetCellValue(sheet, fmt.Sprintf("G%d", r), log.IPAddress)
		}

		colWidths := []float64{22, 20, 15, 12, 15, 50, 18}
		for i, w := range colWidths {
			col, _ := excelize.ColumnNumberToName(i + 1)
			_ = wb.SetColWidth(sheet, col, col, w)
		}

		c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
		c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.xlsx"`, filename))

		if err := wb.Write(c.Writer); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to write xlsx"})
			return
		}

	default:
		c.Header("Content-Type", "text/csv; charset=utf-8")
		c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.csv"`, filename))
		_, _ = c.Writer.Write([]byte{0xEF, 0xBB, 0xBF})

		writer := csv.NewWriter(c.Writer)
		_ = shared.WriteCSVRow(writer, []string{"Timestamp", "Actor", "Role", "Action", "Resource", "Description", "IP Address"})
		for _, log := range logs {
			t := log.CreatedAt
			if parsed, err := time.Parse(time.RFC3339, t); err == nil {
				t = parsed.In(cfg.Timezone).Format("2006-01-02 15:04:05")
			}
			_ = shared.WriteCSVRow(writer, []string{
				t,
				log.Username,
				log.Role,
				log.Action,
				log.EntityType,
				log.Description,
				log.IPAddress,
			})
		}
		writer.Flush()
	}
}

func GenerateAuditDescription(log *AuditLog) string {
	action := strings.ToLower(log.Action)
	entity := strings.ToLower(log.EntityType)

	getIdentifier := func(val interface{}) string {
		if val == nil {
			return ""
		}
		if m, ok := val.(map[string]interface{}); ok {
			if name, ok := m["name"].(string); ok {
				return name
			}
			if name, ok := m["username"].(string); ok {
				return name
			}
			if inv, ok := m["invoice_number"].(string); ok {
				return inv
			}
		}
		return ""
	}

	identifier := getIdentifier(log.NewValues)
	if identifier == "" {
		identifier = getIdentifier(log.OldValues)
	}

	var displayAction string
	switch action {
	case "create":
		displayAction = "Created"
	case "update":
		displayAction = "Updated"
	case "delete":
		displayAction = "Deleted"
	case "login":
		displayAction = "Logged in"
	case "logout":
		displayAction = "Logged out"
	default:
		if len(action) > 0 {
			displayAction = string(unicode.ToUpper(rune(action[0]))) + action[1:]
		} else {
			displayAction = action
		}
	}

	if identifier != "" {
		return fmt.Sprintf("%s %s: %s", displayAction, entity, identifier)
	}

	if log.EntityID != nil && *log.EntityID > 0 {
		if entity == "auth" && (action == "login" || action == "logout") {
			if log.Username != "" {
				return fmt.Sprintf("%s %s", displayAction, log.Username)
			}
			return displayAction
		}
		return fmt.Sprintf("%s %s #%d", displayAction, entity, *log.EntityID)
	}

	if entity == "auth" && (action == "login" || action == "logout") {
		if log.Username != "" {
			return fmt.Sprintf("%s %s", displayAction, log.Username)
		}
		return displayAction
	}

	if log.NewValues != nil {
		if m, ok := log.NewValues.(map[string]interface{}); ok {
			if desc, ok := m["description"].(string); ok {
				return desc
			}
		}
	}

	return ""
}

func parseDateParam(s string) string {
	if s == "" {
		return ""
	}
	if _, err := time.Parse(time.RFC3339, s); err == nil {
		return s
	}
	if _, err := time.Parse("2006-01-02", s); err == nil {
		return s
	}
	return ""
}
