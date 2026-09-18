package mcpserver

import (
	"context"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"

	conf "github.com/TechXploreLabs/seristack/internal/config"
)

func contextWithIdentity(identity map[string][]string) context.Context {
	return context.WithValue(context.Background(), mcpIdentityKey, identity)
}

func TestGetHeaderValues(t *testing.T) {
	tests := []struct {
		name     string
		identity map[string][]string
		header   string
		want     []string
	}{
		{
			name: "single value",
			identity: map[string][]string{
				"X-Auth-Request-Groups": {"platform"},
			},
			header: "X-Auth-Request-Groups",
			want:   []string{"platform"},
		},
		{
			name: "comma separated values",
			identity: map[string][]string{
				"X-Auth-Request-Groups": {"platform,sre,developers"},
			},
			header: "X-Auth-Request-Groups",
			want:   []string{"platform", "sre", "developers"},
		},
		{
			name: "multiple header values",
			identity: map[string][]string{
				"X-Auth-Request-Groups": {
					"platform",
					"sre,developers",
				},
			},
			header: "X-Auth-Request-Groups",
			want:   []string{"platform", "sre", "developers"},
		},
		{
			name: "whitespace is trimmed",
			identity: map[string][]string{
				"X-Auth-Request-Groups": {" platform, sre , developers "},
			},
			header: "X-Auth-Request-Groups",
			want:   []string{"platform", "sre", "developers"},
		},
		{
			name: "missing header",
			identity: map[string][]string{
				"X-Auth-Request-Email": {"user@example.com"},
			},
			header: "X-Auth-Request-Groups",
			want:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getHeaderValues(tt.identity, tt.header)

			if len(got) != len(tt.want) {
				t.Fatalf("expected %v, got %v", tt.want, got)
			}

			for i := range tt.want {
				if got[i] != tt.want[i] {
					t.Errorf("expected %q at index %d, got %q",
						tt.want[i], i, got[i])
				}
			}
		})
	}
}

func TestCheckMCPAccess_ANY(t *testing.T) {
	rules := []conf.AccessRule{
		{
			HeaderName:  "X-Auth-Request-Groups",
			HeaderValue: []string{"platform", "sre"},
		},
	}

	tests := []struct {
		name     string
		identity map[string][]string
		want     bool
	}{
		{
			name: "platform is allowed",
			identity: map[string][]string{
				"X-Auth-Request-Groups": {"platform"},
			},
			want: true,
		},
		{
			name: "sre is allowed",
			identity: map[string][]string{
				"X-Auth-Request-Groups": {"sre"},
			},
			want: true,
		},
		{
			name: "developers are denied",
			identity: map[string][]string{
				"X-Auth-Request-Groups": {"developers"},
			},
			want: false,
		},
		{
			name: "multiple groups with one allowed",
			identity: map[string][]string{
				"X-Auth-Request-Groups": {"developers", "platform"},
			},
			want: true,
		},
		{
			name:     "no identity",
			identity: nil,
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := contextWithIdentity(tt.identity)

			got := checkMCPAccess(ctx, rules, "ANY")

			if got != tt.want {
				t.Errorf("expected %v, got %v", tt.want, got)
			}
		})
	}
}

func TestCheckMCPAccess_ALL(t *testing.T) {
	rules := []conf.AccessRule{
		{
			HeaderName:  "X-Auth-Request-Groups",
			HeaderValue: []string{"platform", "platform-admin"},
		},
		{
			HeaderName:  "X-Auth-Request-Roles",
			HeaderValue: []string{"admin"},
		},
	}

	tests := []struct {
		name     string
		identity map[string][]string
		want     bool
	}{
		{
			name: "platform and admin allowed",
			identity: map[string][]string{
				"X-Auth-Request-Groups": {"platform"},
				"X-Auth-Request-Roles":  {"admin"},
			},
			want: true,
		},
		{
			name: "platform-admin and admin allowed",
			identity: map[string][]string{
				"X-Auth-Request-Groups": {"platform-admin"},
				"X-Auth-Request-Roles":  {"admin"},
			},
			want: true,
		},
		{
			name: "missing role denied",
			identity: map[string][]string{
				"X-Auth-Request-Groups": {"platform"},
			},
			want: false,
		},
		{
			name: "wrong group denied",
			identity: map[string][]string{
				"X-Auth-Request-Groups": {"developers"},
				"X-Auth-Request-Roles":  {"admin"},
			},
			want: false,
		},
		{
			name: "wrong role denied",
			identity: map[string][]string{
				"X-Auth-Request-Groups": {"platform"},
				"X-Auth-Request-Roles":  {"viewer"},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := contextWithIdentity(tt.identity)

			got := checkMCPAccess(ctx, rules, "ALL")

			if got != tt.want {
				t.Errorf("expected %v, got %v", tt.want, got)
			}
		})
	}
}

func TestCheckMCPAccess_NoRules(t *testing.T) {
	ctx := contextWithIdentity(nil)

	got := checkMCPAccess(ctx, nil, "ANY")

	if !got {
		t.Error("expected access to be allowed when there are no rules")
	}
}

func TestCheckMCPAccess_DefaultMatchMode(t *testing.T) {
	rules := []conf.AccessRule{
		{
			HeaderName:  "X-Auth-Request-Groups",
			HeaderValue: []string{"platform"},
		},
		{
			HeaderName:  "X-Auth-Request-Groups",
			HeaderValue: []string{"sre"},
		},
	}

	ctx := contextWithIdentity(map[string][]string{
		"X-Auth-Request-Groups": {"sre"},
	})

	// Anything other than ALL currently behaves as ANY.
	got := checkMCPAccess(ctx, rules, "")

	if !got {
		t.Error("expected default match mode to behave as ANY")
	}
}

func TestIdentityFromContext(t *testing.T) {
	identity := map[string][]string{
		"X-Auth-Request-User":   {"auth0|123"},
		"X-Auth-Request-Email":  {"user@example.com"},
		"X-Auth-Request-Groups": {"platform"},
	}

	ctx := contextWithIdentity(identity)

	got := identityFromContext(ctx)

	if len(got) != len(identity) {
		t.Fatalf("expected %d identity headers, got %d",
			len(identity), len(got))
	}

	if got["X-Auth-Request-Groups"][0] != "platform" {
		t.Errorf(
			"expected group platform, got %v",
			got["X-Auth-Request-Groups"],
		)
	}
}

func TestIdentityFromContext_Missing(t *testing.T) {
	ctx := context.Background()

	got := identityFromContext(ctx)

	if got != nil {
		t.Errorf("expected nil identity, got %v", got)
	}
}

func TestFlattenIdentity(t *testing.T) {
	identity := map[string][]string{
		"X-Auth-Request-User":   {"auth0|123"},
		"X-Auth-Request-Email":  {"user@example.com"},
		"X-Auth-Request-Groups": {"platform", "sre"},
		"Authorization":         {"Bearer secret"},
	}

	got := flattenIdentity(
		identity,
		[]string{"Authorization"},
	)

	if _, exists := got["Authorization"]; exists {
		t.Error("Authorization should have been excluded")
	}

	if got["X-Auth-Request-User"] != "auth0|123" {
		t.Errorf(
			"unexpected user: %q",
			got["X-Auth-Request-User"],
		)
	}

	if got["X-Auth-Request-Groups"] != "platform,sre" {
		t.Errorf(
			"unexpected groups: %q",
			got["X-Auth-Request-Groups"],
		)
	}
}

func TestFlattenIdentity_Nil(t *testing.T) {
	got := flattenIdentity(nil, nil)

	if got != nil {
		t.Errorf("expected nil result, got %v", got)
	}
}

func TestMCPToolFilter(t *testing.T) {
	stackMap := map[string]*conf.Stack{
		"system-health": {
			Name: "system-health",
			Access: []conf.AccessRule{
				{
					HeaderName: "X-Auth-Request-Groups",
					HeaderValue: []string{
						"developers",
						"platform",
						"platform-admin",
					},
				},
			},
		},
		"inspect-runtime": {
			Name: "inspect-runtime",
			Access: []conf.AccessRule{
				{
					HeaderName: "X-Auth-Request-Groups",
					HeaderValue: []string{
						"platform",
						"platform-admin",
					},
				},
			},
		},
		"production-operation": {
			Name: "production-operation",
			Access: []conf.AccessRule{
				{
					HeaderName: "X-Auth-Request-Groups",
					HeaderValue: []string{
						"platform-admin",
					},
				},
			},
		},
	}

	filter := mcpToolFilter(stackMap)

	tools := []mcp.Tool{
		mcp.NewTool("system-health"),
		mcp.NewTool("inspect-runtime"),
		mcp.NewTool("production-operation"),
	}

	tests := []struct {
		name     string
		identity map[string][]string
		want     []string
	}{
		{
			name: "developers only see system-health",
			identity: map[string][]string{
				"X-Auth-Request-Groups": {"developers"},
			},
			want: []string{
				"system-health",
			},
		},
		{
			name: "platform sees system-health and inspect-runtime",
			identity: map[string][]string{
				"X-Auth-Request-Groups": {"platform"},
			},
			want: []string{
				"system-health",
				"inspect-runtime",
			},
		},
		{
			name: "platform-admin sees all tools",
			identity: map[string][]string{
				"X-Auth-Request-Groups": {"platform-admin"},
			},
			want: []string{
				"system-health",
				"inspect-runtime",
				"production-operation",
			},
		},
		{
			name: "unknown group sees nothing",
			identity: map[string][]string{
				"X-Auth-Request-Groups": {"unknown"},
			},
			want: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := contextWithIdentity(tt.identity)

			got := filter(ctx, tools)

			if len(got) != len(tt.want) {
				t.Fatalf(
					"expected tools %v, got %v",
					tt.want,
					toolNames(got),
				)
			}

			for i, expected := range tt.want {
				if got[i].Name != expected {
					t.Errorf(
						"expected tool %q at index %d, got %q",
						expected,
						i,
						got[i].Name,
					)
				}
			}
		})
	}
}

func TestMCPToolFilter_UnknownTool(t *testing.T) {
	stackMap := map[string]*conf.Stack{
		"system-health": {
			Name: "system-health",
		},
	}

	filter := mcpToolFilter(stackMap)

	ctx := contextWithIdentity(nil)

	tools := []mcp.Tool{
		mcp.NewTool("system-health"),
		mcp.NewTool("unknown-tool"),
	}

	got := filter(ctx, tools)

	if len(got) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(got))
	}

	if got[0].Name != "system-health" {
		t.Errorf("expected system-health, got %s", got[0].Name)
	}
}

func toolNames(tools []mcp.Tool) []string {
	result := make([]string, 0, len(tools))

	for _, tool := range tools {
		result = append(result, tool.Name)
	}

	return result
}
