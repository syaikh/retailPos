package audit

import (
	"context"
	"encoding/json"
	"log"
	"strings"
	"time"

	"retail-pos-system/internal/brand"
	"retail-pos-system/internal/category"
	"retail-pos-system/internal/customer"
	"retail-pos-system/internal/eventbus"
	"retail-pos-system/internal/inventory"
	"retail-pos-system/internal/middleware"
	"retail-pos-system/internal/product"
	"retail-pos-system/internal/sale"
	"retail-pos-system/internal/uom"
	"retail-pos-system/internal/user"
)

type AuditLogCreator interface {
	CreateAuditLog(ctx context.Context, log *AuditLog) error
}

func NewAuditListener(svc AuditLogCreator) eventbus.Listener {
	return eventbus.NewListenerFunc(
		[]eventbus.EventType{
			"sale.created",
			"product.created", "product.updated", "product.deleted",
			"customer.created", "customer.updated", "customer.deleted",
			"category.created", "category.updated", "category.deleted",
			"user.created", "user.updated", "user.deleted",
			"role.created", "role.updated", "role.deleted",
			"auth.login", "auth.logout",
			"brand.created", "brand.updated", "brand.deleted",
			"uom.created", "uom.updated", "uom.deleted",
			eventbus.StockAdjusted,
		},
		func(ctx context.Context, event eventbus.Event) error {
			userID := middleware.UserIDFromContext(ctx)
			username := middleware.UsernameFromContext(ctx)
			role := middleware.RoleFromContext(ctx)

			action, entityType, entityID, oldV, newV := extractEventData(event)
			if action == "" {
				return nil
			}

			alog := &AuditLog{
				UserID:     userID,
				Username:   username,
				Role:       role,
				Action:     action,
				EntityType: entityType,
				EntityID:   entityID,
				OldValues:  oldV,
				NewValues:  newV,
				IPAddress:  middleware.IPAddressFromContext(ctx),
				UserAgent:  middleware.UserAgentFromContext(ctx),
			}

			alog.Description = GenerateAuditDescription(alog)

			// Use background context for DB write — the request context (ctx) may
			// already be cancelled since dispatch runs asynchronously (goroutine).
			// Context values (userID, username, etc.) survive cancellation, but
			// pgx Exec uses ctx.Done() and would fail immediately on a cancelled ctx.
			dbCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := svc.CreateAuditLog(dbCtx, alog); err != nil {
				log.Printf("[audit] failed to create log: %v", err)
			}
			return nil
		},
	)
}

func extractEventData(event eventbus.Event) (action, entityType string, entityID *int, oldValues, newValues interface{}) {
	switch e := event.Payload.(type) {
	case *sale.Sale:
		return "create", "sale", &e.ID, nil, toJSONMap(e)

	case *product.Product:
		return extractFromEventType(string(event.Type), "product", e.ID, toJSONMap(e))

	case inventory.StockAdjustedEvent:
		return "update", "stock", &e.ProductID, nil, toJSONMap(e)

	case *customer.Customer:
		return extractFromEventType(string(event.Type), "customer", e.ID, toJSONMap(e))

	case *category.Category:
		return extractFromEventType(string(event.Type), "category", e.ID, toJSONMap(e))

	case *user.User:
		return extractFromEventType(string(event.Type), "user", e.ID, toJSONMap(e))

	case *brand.Brand:
		return extractFromEventType(string(event.Type), "brand", e.ID, toJSONMap(e))

	case *uom.UnitOfMeasure:
		return extractFromEventType(string(event.Type), "uom", e.ID, toJSONMap(e))

	case *user.Role:
		return extractFromEventType(string(event.Type), "role", e.ID, toJSONMap(e))

	case int:
		return extractFromEventType(string(event.Type), "entity", e, nil)

	case eventbus.UpdatePayload:
		etyp, eid, ok := extractEntityInfo(e.New)
		if !ok {
			etyp, eid, ok = extractEntityInfo(e.Old)
			if !ok {
				return "", "", nil, nil, nil
			}
		}
		action, _, _, _, _ := extractFromEventType(string(event.Type), etyp, eid, nil)
		return action, etyp, &eid, toJSONMap(e.Old), toJSONMap(e.New)

	case map[string]interface{}:
		uid := intFromMap(e, "user_id")
		switch string(event.Type) {
		case "auth.login":
			return "login", "auth", uid, nil, e
		case "auth.logout":
			return "logout", "auth", uid, nil, nil
		}
	}

	return "", "", nil, nil, nil
}

func extractEntityInfo(v interface{}) (entityType string, entityID int, ok bool) {
	switch e := v.(type) {
	case *product.Product:
		return "product", e.ID, true
	case *customer.Customer:
		return "customer", e.ID, true
	case *category.Category:
		return "category", e.ID, true
	case *user.User:
		return "user", e.ID, true
	case *user.Role:
		return "role", e.ID, true
	case *brand.Brand:
		return "brand", e.ID, true
	case *uom.UnitOfMeasure:
		return "uom", e.ID, true
	case *sale.Sale:
		return "sale", e.ID, true
	}
	return "", 0, false
}

func extractFromEventType(evtType, defaultEntity string, id int, newV interface{}) (action, entityType string, entityID *int, oldValues, newValues interface{}) {
	parts := strings.SplitN(evtType, ".", 2)
	entityType = defaultEntity
	if len(parts) >= 2 && parts[0] != "" {
		entityType = parts[0]
	}

	switch {
	case strings.Contains(evtType, ".created"):
		action = "create"
	case strings.Contains(evtType, ".updated"):
		action = "update"
	case strings.Contains(evtType, ".deleted"):
		action = "delete"
	default:
		return "", "", nil, nil, nil
	}

	entityID = &id
	newValues = newV
	return
}

func toJSONMap(v interface{}) map[string]interface{} {
	if v == nil {
		return nil
	}
	data, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return nil
	}
	return m
}

func intFromMap(m map[string]interface{}, key string) *int {
	if v, ok := m[key]; ok {
		switch n := v.(type) {
		case int:
			return &n
		case float64:
			i := int(n)
			return &i
		}
	}
	return nil
}
