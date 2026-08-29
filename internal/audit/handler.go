package audit

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"retail-pos-system/internal/config"
	"retail-pos-system/internal/permissions"
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

func (h *Handler) RegisterRoutes(r *gin.RouterGroup, auth gin.HandlerFunc, perm func(permissions.Code) gin.HandlerFunc) {
	r.GET("/audit-logs", auth, perm(permissions.AuditView), h.ListAuditLogs)
	r.GET("/audit-logs/:id", auth, perm(permissions.AuditView), h.GetAuditLog)
	r.GET("/audit-logs/export", auth, perm(permissions.AuditExport), h.ExportAuditLogs)
	r.GET("/audit-logs/entity-types", auth, perm(permissions.AuditView), h.ListEntityTypes)
}

func (h *Handler) ListAuditLogs(c *gin.Context) {
	limit, offset := shared.ParsePaginationParams(c.Query("limit"), c.Query("offset"))

	var userID *int
	if uid := c.Query("user_id"); uid != "" {
		if val, err := strconv.Atoi(uid); err == nil {
			userID = &val
		}
	}

	search := c.Query("search")
	action := c.Query("action")
	entityType := c.Query("entity_type")

	var entityID *int
	if eid := c.Query("entity_id"); eid != "" {
		if val, err := strconv.Atoi(eid); err == nil {
			entityID = &val
		}
	}

	startDate := parseDateParam(c.Query("start_date"))
	endDate := parseDateParam(c.Query("end_date"))

	logs, total, err := h.svc.GetAuditLogs(c.Request.Context(), limit, offset, userID, search, action, entityType, entityID, startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch audit logs"})
		return
	}

	if logs == nil {
		logs = []LogListItem{}
	}

	shared.JSONPaginated(c, logs, total, limit, offset)
}

func (h *Handler) GetAuditLog(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	log, err := h.svc.GetAuditLogByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "audit log not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": log})
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

	var entityID *int
	if eid := c.Query("entity_id"); eid != "" {
		if val, err := strconv.Atoi(eid); err == nil {
			entityID = &val
		}
	}

	startDate := parseDateParam(c.Query("start_date"))
	endDate := parseDateParam(c.Query("end_date"))

	const maxExportRows = 10000
	logs, _, err := h.svc.GetAuditLogs(c.Request.Context(), maxExportRows, 0, userID, search, action, entityType, entityID, startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch audit logs"})
		return
	}

	if logs == nil {
		logs = []LogListItem{}
	}

	// Record the export itself as a security-critical audit event. Failing
	// closed means an export that cannot be audited is not served (P2 #5).
	exportLog := &Log{
		Username:    shared.GetUsername(c),
		Role:        shared.GetRole(c),
		Action:      "audit_exported",
		EntityType:  "audit",
		IPAddress:   shared.GetIPAddress(c),
		UserAgent:   shared.GetUserAgent(c),
		Description: fmt.Sprintf("Exported %d audit log(s) as %s", len(logs), format),
		StoreID:     shared.GetStoreID(c),
	}
	if uid := shared.GetUserID(c); uid > 0 {
		exportLog.UserID = &uid
	}
	if !WriteFailClosed(c.Request.Context(), h.svc, exportLog) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to write audit log"})
		return
	}

	cfg := config.Load()
	now := time.Now().In(cfg.Timezone)
	filename := "audit-logs-" + now.Format("2006-01-02")

	switch format {
	case "xlsx":
		wb := excelize.NewFile()
		sheet := "Audit Logs"
		_ = wb.SetSheetName("Sheet1", sheet)

		headers := []string{"Timestamp", "Actor", "Role", "Action", "Resource", "Entity ID", "Description", "Changes", "IP Address", "User Agent", "Store", "Correlation ID"}
		for i, hdr := range headers {
			col, _ := excelize.ColumnNumberToName(i + 1)
			_ = wb.SetCellValue(sheet, col+"1", hdr)
		}
		headerStyle, _ := wb.NewStyle(&excelize.Style{
			Font: &excelize.Font{Bold: true, Color: "FFFFFF"},
			Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"7C3AED"}},
		})
		_ = wb.SetCellStyle(sheet, "A1", "L1", headerStyle)

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
			_ = wb.SetCellValue(sheet, fmt.Sprintf("F%d", r), formatEntityID(log.EntityID))
			_ = wb.SetCellValue(sheet, fmt.Sprintf("G%d", r), log.Description)
			_ = wb.SetCellValue(sheet, fmt.Sprintf("H%d", r), formatChanges(log.OldValues, log.NewValues))
			_ = wb.SetCellValue(sheet, fmt.Sprintf("I%d", r), log.IPAddress)
			_ = wb.SetCellValue(sheet, fmt.Sprintf("J%d", r), log.UserAgent)
			_ = wb.SetCellValue(sheet, fmt.Sprintf("K%d", r), storeLabel(log))
			_ = wb.SetCellValue(sheet, fmt.Sprintf("L%d", r), log.CorrelationID)
		}

		colWidths := []float64{22, 20, 15, 12, 15, 10, 50, 50, 18, 30, 10, 24}
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
		_ = shared.WriteCSVRow(writer, []string{"Timestamp", "Actor", "Role", "Action", "Resource", "Entity ID", "Description", "Changes", "IP Address", "User Agent", "Store", "Correlation ID"})
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
				formatEntityID(log.EntityID),
				log.Description,
				formatChanges(log.OldValues, log.NewValues),
				log.IPAddress,
				log.UserAgent,
				storeLabel(log),
				log.CorrelationID,
			})
		}
		writer.Flush()
	}
}

func storeLabel(log LogListItem) string {
	if log.StoreName != "" {
		return log.StoreName
	}
	if log.StoreID != nil {
		return fmt.Sprintf("%d", *log.StoreID)
	}
	return ""
}

func GenerateAuditDescription(log *Log) string {
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

func formatEntityID(entityID *int) string {
	if entityID == nil {
		return ""
	}
	return strconv.Itoa(*entityID)
}

var formatSensitiveFields = map[string]struct{}{
	"password":      {},
	"password_hash": {},
	"token":         {},
	"salt":          {},
	"refresh_token": {},
}

func formatChanges(oldValues, newValues interface{}) string {
	oldMap := changesToMap(oldValues)
	newMap := changesToMap(newValues)
	if len(oldMap) == 0 && len(newMap) == 0 {
		return ""
	}

	keys := map[string]struct{}{}
	for k := range oldMap {
		keys[k] = struct{}{}
	}
	for k := range newMap {
		keys[k] = struct{}{}
	}

	sortedKeys := make([]string, 0, len(keys))
	for k := range keys {
		sortedKeys = append(sortedKeys, k)
	}
	sort.Strings(sortedKeys)

	var parts []string
	for _, k := range sortedKeys {
		if _, isSensitive := formatSensitiveFields[k]; isSensitive {
			continue
		}
		oldV, hasOld := oldMap[k]
		newV, hasNew := newMap[k]
		switch {
		case hasOld && hasNew && fmt.Sprintf("%v", oldV) == fmt.Sprintf("%v", newV):
			continue
		case hasOld && hasNew:
			parts = append(parts, fmt.Sprintf("%s: %v -> %v", k, oldV, newV))
		case hasNew:
			parts = append(parts, fmt.Sprintf("%s: %v", k, newV))
		case hasOld:
			parts = append(parts, fmt.Sprintf("%s: %v (removed)", k, oldV))
		}
	}
	return strings.Join(parts, "\n")
}

func changesToMap(v interface{}) map[string]interface{} {
	result := map[string]interface{}{}
	if v == nil {
		return result
	}
	if m, ok := v.(map[string]interface{}); ok {
		return m
	}
	raw, err := json.Marshal(v)
	if err != nil {
		return result
	}
	var m map[string]interface{}
	if err := json.Unmarshal(raw, &m); err != nil {
		return result
	}
	if m != nil {
		return m
	}
	return result
}
