package pricing

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"retail-pos-system/internal/middleware"
	"retail-pos-system/internal/permissions"

	"github.com/gin-gonic/gin"
)

// approvalCase describes one caller/author pairing for the separation-of-duties
// check. IDs are arbitrary: only equality is under test, not identity.
type approvalCase struct {
	name       string
	role       string
	callerID   *int
	authorID   *int
	wantStatus int
}

func ptr(i int) *int { return &i }

func TestAuthorizeApproval(t *testing.T) {
	cases := []approvalCase{
		{
			name:       "manager approving someone else's rule",
			role:       permissions.RoleManager,
			callerID:   ptr(2),
			authorID:   ptr(3),
			wantStatus: 0, // allowed: handler writes no response
		},
		{
			name:       "manager approving their own rule",
			role:       permissions.RoleManager,
			callerID:   ptr(2),
			authorID:   ptr(2),
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "superadmin approving their own rule",
			role:       permissions.RoleSuperadmin,
			callerID:   ptr(1),
			authorID:   ptr(1),
			wantStatus: 0,
		},
		{
			name:       "superadmin approving another user's rule",
			role:       permissions.RoleSuperadmin,
			callerID:   ptr(1),
			authorID:   ptr(2),
			wantStatus: 0,
		},
		{
			// Pre-054 rules have a NULL author. Refusing those would strand
			// every pending rule that predates the migration.
			name:       "manager approving a legacy rule with no recorded author",
			role:       permissions.RoleManager,
			callerID:   ptr(2),
			authorID:   nil,
			wantStatus: 0,
		},
		{
			// Defence in depth: if the token carried no user id we cannot claim
			// the caller is the author, so the request proceeds to the store
			// guard rather than being rejected for a reason we cannot prove.
			name:       "caller without a resolvable id is not treated as the author",
			role:       permissions.RoleManager,
			callerID:   nil,
			authorID:   ptr(2),
			wantStatus: 0,
		},
	}

	gin.SetMode(gin.TestMode)
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(rec)

			// AuthMiddleware populates both the gin context and the request
			// context, and authorizeApproval reads one of each: isAdmin reads
			// "role" off the gin context while the author comparison reads the
			// user id from the request context. Populate both, as the middleware
			// does, or the test exercises a shape production never produces.
			c.Set("role", tc.role)
			ctx := context.WithValue(context.Background(), middleware.CtxKeyRole, tc.role)
			if tc.callerID != nil {
				c.Set("userID", *tc.callerID)
				ctx = context.WithValue(ctx, middleware.CtxKeyUserID, *tc.callerID)
			}
			c.Request = httptest.NewRequest(http.MethodPost, "/pricing-rules/1/approve", nil).
				WithContext(ctx)

			ok := authorizeApproval(c, &Rule{CreatedBy: tc.authorID})

			if tc.wantStatus == 0 {
				if !ok {
					t.Fatalf("authorizeApproval = false, want true (body: %s)", rec.Body.String())
				}
				return
			}
			if ok {
				t.Fatalf("authorizeApproval = true, want false")
			}
			if rec.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tc.wantStatus)
			}
		})
	}
}

// statusIgnoringRepo records what the service asked to persist, so the test can
// assert on the status that would actually reach the database.
type statusIgnoringRepo struct {
	Repository
	existing  *Rule
	saved     *Rule
	updated   bool
	nameTaken bool
}

func (r *statusIgnoringRepo) GetByID(_ context.Context, id int) (*Rule, error) {
	if r.existing == nil {
		return nil, errors.New("no existing rule")
	}
	copied := *r.existing
	copied.ID = id
	return &copied, nil
}

func (r *statusIgnoringRepo) NameExists(_ context.Context, _ string, _ int) (bool, error) {
	return r.nameTaken, nil
}

func (r *statusIgnoringRepo) Update(_ context.Context, rule *Rule) error {
	copied := *rule
	r.saved = &copied
	r.updated = true
	return nil
}

// TestUpdateCannotGrantApproval covers the status side door on the edit route.
// Rule.Status carries a `json:"status"` tag, so before it was pinned to the
// stored row a caller holding only pricing.update could POST status:"approved"
// and approve their own rule without ever reaching the pricing.approve gate or
// authorizeApproval. The store_id sibling of this bug is covered by
// TestBindStoreScopedRule.
func TestUpdateCannotGrantApproval(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cases := []struct {
		name           string
		bodyStatus     RuleStatus
		existingStatus RuleStatus
		wantStatus     RuleStatus
		wantDeactiv    bool
	}{
		{
			name:           "body status approved cannot approve a pending rule",
			bodyStatus:     StatusApproved,
			existingStatus: StatusPending,
			wantStatus:     StatusPending,
		},
		{
			name:           "body status rejected cannot reject an approved rule",
			bodyStatus:     StatusRejected,
			existingStatus: StatusApproved,
			wantStatus:     StatusApproved,
		},
		{
			name:           "omitted status keeps the stored value",
			bodyStatus:     "",
			existingStatus: StatusPending,
			wantStatus:     StatusPending,
		},
		{
			// Deactivation is a legitimate edit and is preserved. It cannot
			// activate a rule on its own, since resolution requires both
			// is_active and status='approved'.
			name:           "is_active false is still honoured",
			bodyStatus:     StatusApproved,
			existingStatus: StatusPending,
			wantStatus:     StatusPending,
			wantDeactiv:    true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			storeID := 1
			productID := 1
			// The stored rule mirrors the request body exactly. These cases are
			// about whether a body-supplied status can change the rule's workflow
			// state; if the stored rule differed economically, the approved-rule
			// reset (service.Update sends an economically edited approved rule back
			// to pending) would fire and mask what is being asserted.
			repo := &statusIgnoringRepo{existing: &Rule{
				ID:              1,
				StoreID:         &storeID,
				ProductID:       &productID,
				Type:            PricingTypePromotion,
				Method:          PricingMethodFixedPrice,
				PricingValue:    10,
				MinimumQuantity: 1,
				Status:          tc.existingStatus,
				IsActive:        true,
			}}
			svc := &service{repo: repo}
			h := &Handler{svc: svc}

			body := fmt.Sprintf(`{"product_id":1,"pricing_type":"promotion",`+
				`"pricing_method":"fixed_price","pricing_value":10,"name":"r",`+
				`"minimum_quantity":1,"is_active":%t`, !tc.wantDeactiv)
			if tc.bodyStatus != "" {
				body += fmt.Sprintf(`,"status":%q`, tc.bodyStatus)
			}
			body += "}"

			rec := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(rec)
			c.Set("storeID", &storeID)
			c.Request = httptest.NewRequest(http.MethodPut, "/pricing-rules/1", strings.NewReader(body))
			c.Params = gin.Params{{Key: "id", Value: "1"}}

			h.UpdateRule(c)

			if !repo.updated {
				t.Fatalf("UpdateRule did not reach the repository; status = %d, body = %s",
					rec.Code, rec.Body.String())
			}
			if repo.saved.Status != tc.wantStatus {
				t.Errorf("persisted status = %q, want %q (the stored row's value)",
					repo.saved.Status, tc.wantStatus)
			}
		})
	}
}
