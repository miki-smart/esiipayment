// Package model defines the in-memory shape of a provider manifest,
// its metadata, and its cassettes, mirroring schema/manifest.v1.schema.json,
// schema/metadata.v1.schema.json, and schema/cassette.v1.schema.json.
//
// Fields that correspond to the DSL's own closed vocabulary are typed as
// named structs so unknown-key rejection (via strict YAML decoding) can
// enforce additionalProperties: false. Fields that carry a provider's own
// pass-through vocabulary (call.body, call.headers, status_map.values,
// emit.state, and cassette intent/ctx/credentials/state) are typed as
// plain maps, matching the additionalProperties exception documented in
// spec/03-manifest-dsl.md.
package model

// Manifest is providers/<name>/manifest.yaml.
type Manifest struct {
	Provider     string                    `yaml:"provider"`
	SpecVersion  string                    `yaml:"spec_version"`
	DisplayName  string                    `yaml:"display_name"`
	Country      string                    `yaml:"country"`
	Currencies   []string                  `yaml:"currencies"`
	Environments map[string]Environment    `yaml:"environments"`
	Auth         Auth                      `yaml:"auth"`
	Capabilities Capabilities              `yaml:"capabilities"`
	Operations   map[string]OperationEntry `yaml:"operations"`
	Flows        map[string]Flow           `yaml:"flows"`
	Errors       []ErrorMapping            `yaml:"errors"`
	Webhook      *Webhook                  `yaml:"webhook"`
}

type Environment struct {
	BaseURL string `yaml:"base_url"`
}

type Auth struct {
	Shape  string            `yaml:"shape"`
	Fields []CredentialField `yaml:"fields"`
	Apply  *AuthApply        `yaml:"apply"`
	Token  *AuthToken        `yaml:"token"`
}

type CredentialField struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

// AuthApply says where the resolved credential goes on every outbound
// call: exactly one of Header/Query/Body is set. See
// spec/03-manifest-dsl.md#auth.
type AuthApply struct {
	Header string `yaml:"header"`
	Query  string `yaml:"query"`
	Body   string `yaml:"body"`
	Value  string `yaml:"value"`
}

// Target returns which of header/query/body is set, and its name, or
// ("", "") if none is (a shape-validation error checks.checkAuth reports
// separately).
func (a *AuthApply) Target() (kind, name string) {
	switch {
	case a.Header != "":
		return "header", a.Header
	case a.Query != "":
		return "query", a.Query
	case a.Body != "":
		return "body", a.Body
	default:
		return "", ""
	}
}

// AuthToken is the token-exchange mechanism behind the
// oauth2_client_credentials CredentialShape. See spec/03-manifest-dsl.md#auth.
type AuthToken struct {
	Call      AuthTokenCall          `yaml:"call"`
	Body      map[string]interface{} `yaml:"body"`
	Extract   AuthTokenExtract       `yaml:"extract"`
	RefreshAt float64                `yaml:"refresh_at"`
}

type AuthTokenCall struct {
	Method string `yaml:"method"`
	Path   string `yaml:"path"`
}

type AuthTokenExtract struct {
	AccessToken string `yaml:"access_token"`
	ExpiresIn   string `yaml:"expires_in"`
}

type Capabilities struct {
	Operations  []string                  `yaml:"operations"`
	NextActions []string                  `yaml:"next_actions"`
	Currencies  map[string]CurrencyLimits `yaml:"currencies"`
}

type CurrencyLimits struct {
	MinAmount int64 `yaml:"min_amount"`
	MaxAmount int64 `yaml:"max_amount"`
}

type OperationEntry struct {
	EntryFlow string `yaml:"entry_flow"`
}

type Flow struct {
	Steps map[string]Step `yaml:"steps"`
}

type Step struct {
	Call      *Call                  `yaml:"call"`
	Triggers  []string               `yaml:"triggers"`
	StatusMap *StatusMap             `yaml:"status_map"`
	Emit      *Emit                  `yaml:"emit"`
	Goto      string                 `yaml:"goto"`
}

type Call struct {
	Method  string                 `yaml:"method"`
	Path    string                 `yaml:"path"`
	Headers map[string]string      `yaml:"headers"`
	Body    map[string]interface{} `yaml:"body"`
}

type StatusMap struct {
	Path   string            `yaml:"path"`
	Values map[string]string `yaml:"values"`
}

type Emit struct {
	Status string            `yaml:"status"`
	State  map[string]string `yaml:"state"`
	// NextAction is the raw decoded next_action mapping ({"type": "...",
	// ...variant fields}), not a fixed struct: its field set varies per
	// variant (see model.NextActionFields) the same way Call.Body's
	// varies per provider, so it goes through the same generic-map path
	// and the same NormalizeYAMLValue pass as Call.Body (see load.go).
	// nil when a step's emit does not set next_action at all.
	NextAction map[string]interface{} `yaml:"next_action"`
}

// NextActionType returns the "type" discriminator of e's next_action, or
// "" if e or e.NextAction is nil, or the field is missing/not a string.
func (e *Emit) NextActionType() string {
	if e == nil || e.NextAction == nil {
		return ""
	}
	s, _ := e.NextAction["type"].(string)
	return s
}

// ErrorMapping's Match is one of three kinds; exactly one of
// Path (with Equals), HTTPStatus, or Transport is set. See
// spec/03-manifest-dsl.md#errors.
type ErrorMapping struct {
	Match       ErrorMatch  `yaml:"match"`
	FailureCode string      `yaml:"failure_code"`
	RetryClass  string      `yaml:"retry_class"`
}

type ErrorMatch struct {
	Path       string      `yaml:"path"`
	Equals     interface{} `yaml:"equals"`
	HTTPStatus int         `yaml:"http_status"`
	Transport  string      `yaml:"transport"`
}

// Kind reports which of the three match kinds this ErrorMatch uses.
func (m ErrorMatch) Kind() string {
	switch {
	case m.Transport != "":
		return "transport"
	case m.HTTPStatus != 0:
		return "http_status"
	case m.Path != "":
		return "path"
	default:
		return ""
	}
}

type Webhook struct {
	Verification WebhookVerification `yaml:"verification"`
	BodyFormat   string              `yaml:"body_format"`
}

type WebhookVerification struct {
	Scheme          string `yaml:"scheme"`
	SignatureHeader string `yaml:"signature_header"`
	Secret          string `yaml:"secret"`
}

// NativeCapabilities is providers/<name>/capabilities.yaml: the identity
// and capability declaration for a provider implemented as native code
// per language runtime instead of manifest.yaml. See
// spec/03-manifest-dsl.md#native-providers.
type NativeCapabilities struct {
	Provider       string       `yaml:"provider"`
	SpecVersion    string       `yaml:"spec_version"`
	Implementation string       `yaml:"implementation"`
	DisplayName    string       `yaml:"display_name"`
	Country        string       `yaml:"country"`
	Currencies     []string     `yaml:"currencies"`
	Auth           NativeAuth   `yaml:"auth"`
	Capabilities   Capabilities `yaml:"capabilities"`
}

// NativeAuth is capabilities.yaml's auth block: shape/fields only, for
// generic credential-input-form generation. Unlike manifest.yaml's Auth,
// it has no apply/token: a native implementation applies credentials and
// performs any token exchange in code, not through the DSL's mechanism.
type NativeAuth struct {
	Shape  string            `yaml:"shape"`
	Fields []CredentialField `yaml:"fields"`
}

// Metadata is providers/<name>/metadata.yaml.
type Metadata struct {
	Provider      string        `yaml:"provider"`
	Tier          string        `yaml:"tier"`
	Description   string        `yaml:"description"`
	Verification  Verification  `yaml:"verification"`
	Maintainers   []Maintainer  `yaml:"maintainers"`
	Links         Links         `yaml:"links"`
}

type Verification struct {
	Status         string  `yaml:"status"`
	DocsVerifiedOn *string `yaml:"docs_verified_on"`
	VerifiedBy     *string `yaml:"verified_by"`
	Notes          string  `yaml:"notes"`
}

type Maintainer struct {
	Name   string `yaml:"name"`
	GitHub string `yaml:"github"`
}

type Links struct {
	Website string `yaml:"website"`
	Docs    string `yaml:"docs"`
}

// Cassette is providers/<name>/cassettes/<name>.yaml.
type Cassette struct {
	Name            string                 `yaml:"name"`
	Operation       string                 `yaml:"operation"`
	Seed            Seed                   `yaml:"seed"`
	TransportFailure bool                  `yaml:"transport_failure"`
	Intent          map[string]interface{} `yaml:"intent"`
	Environment     string                 `yaml:"environment"`
	Ctx             map[string]string      `yaml:"ctx"`
	Credentials     map[string]string      `yaml:"credentials"`
	State           map[string]interface{} `yaml:"state"`
	Inputs          []NamedInput           `yaml:"inputs"`
	InboundWebhook  *InboundWebhook        `yaml:"inbound_webhook"`
	Interactions    []Interaction          `yaml:"interactions"`
}

type Seed struct {
	Clock          string   `yaml:"clock"`
	UUID           []string `yaml:"uuid"`
	IdempotencyKey string   `yaml:"idempotency_key"`
}

type NamedInput struct {
	Name  string `yaml:"name"`
	Value string `yaml:"value"`
}

type InboundWebhook struct {
	Headers map[string]string `yaml:"headers"`
	Body    string             `yaml:"body"`
}

type Interaction struct {
	Request  Request   `yaml:"request"`
	Response *Response `yaml:"response"`
}

type Request struct {
	Method  string            `yaml:"method"`
	Path    string            `yaml:"path"`
	Headers map[string]string `yaml:"headers"`
	Body    string            `yaml:"body"`
}

type Response struct {
	Status  int               `yaml:"status"`
	Headers map[string]string `yaml:"headers"`
	Body    string            `yaml:"body"`
}
