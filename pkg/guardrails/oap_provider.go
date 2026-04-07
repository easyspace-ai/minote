package guardrails

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

// OAPProvider implements Open AI Passport-based guardrails.
// It validates tool calls against a signed passport token to ensure
// the request is authorized by the platform's authentication system.
type OAPProvider struct {
	passport     string
	allowedTools []string
	deniedTools  []string
}

// NewOAPProvider creates an OAP guardrails provider.
func NewOAPProvider(passport string, args map[string]any) *OAPProvider {
	p := &OAPProvider{
		passport: strings.TrimSpace(passport),
	}
	if args != nil {
		p.allowedTools = readStringList(args["allowed_tools"])
		p.deniedTools = readStringList(args["denied_tools"])
	}
	return p
}

func (p *OAPProvider) Name() string {
	return "oap_passport_provider"
}

func (p *OAPProvider) Evaluate(req Request) (Decision, error) {
	// Validate passport
	if p.passport == "" {
		return Decision{
			Allow:   false,
			Reasons: []Reason{{Code: "missing_passport", Message: "OAP passport not configured"}},
		}, nil
	}

	// Verify passport signature (HMAC-SHA256 of tool call context)
	if !p.verifyPassport(req) {
		return Decision{
			Allow:   false,
			Reasons: []Reason{{Code: "invalid_passport", Message: "OAP passport verification failed"}},
		}, nil
	}

	// Apply allowlist/denylist rules
	toolName := strings.TrimSpace(req.ToolName)
	if len(p.deniedTools) > 0 {
		for _, denied := range p.deniedTools {
			if strings.EqualFold(denied, toolName) {
				return Decision{
					Allow:    false,
					Reasons:  []Reason{{Code: "denied_tool", Message: fmt.Sprintf("Tool %q is denied by OAP policy", toolName)}},
					PolicyID: "oap_deny_list",
				}, nil
			}
		}
	}

	if len(p.allowedTools) > 0 {
		allowed := false
		for _, a := range p.allowedTools {
			if strings.EqualFold(a, toolName) {
				allowed = true
				break
			}
		}
		if !allowed {
			return Decision{
				Allow:    false,
				Reasons:  []Reason{{Code: "not_allowed", Message: fmt.Sprintf("Tool %q is not in the OAP allowed list", toolName)}},
				PolicyID: "oap_allow_list",
			}, nil
		}
	}

	return Decision{
		Allow:    true,
		PolicyID: "oap_passport",
		Metadata: map[string]any{
			"verified_at": time.Now().UTC().Format(time.RFC3339),
		},
	}, nil
}

func (p *OAPProvider) verifyPassport(req Request) bool {
	if p.passport == "" {
		return false
	}

	// Simple HMAC verification: passport is expected to be "secret:signature"
	// For basic mode, just check passport is non-empty (presence-based auth)
	parts := strings.SplitN(p.passport, ":", 2)
	if len(parts) < 2 {
		// Simple presence-based auth: passport exists = authorized
		return true
	}

	secret := parts[0]
	expectedSig := parts[1]

	// Compute HMAC-SHA256 over tool name + agent ID
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(req.ToolName + ":" + req.AgentID))
	computed := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(computed), []byte(expectedSig))
}
