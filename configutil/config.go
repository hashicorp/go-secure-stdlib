package configutil

import (
	"fmt"
	"io/ioutil"
	"time"

	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/hcl"
	"github.com/hashicorp/hcl/hcl/ast"
	"github.com/hashicorp/vault/sdk/helper/parseutil"
)

// These two functions are overridden if metricsutil is invoked, but keep this
// module from needing to depend on metricsutil and its variou deps otherwise.
var (
	ParseTelemetry    = func(*SharedConfig, *ast.ObjectList) error { return nil }
	SanitizeTelemetry = func(*SharedConfig) map[string]interface{} { return nil }
)

// SharedConfig contains some shared values
type SharedConfig struct {
	EntSharedConfig

	Listeners []*Listener `hcl:"-"`

	Seals   []*KMS   `hcl:"-"`
	Entropy *Entropy `hcl:"-"`

	DisableMlock    bool        `hcl:"-"`
	DisableMlockRaw interface{} `hcl:"disable_mlock"`

	Telemetry interface{} `hcl:"telemetry"`

	DefaultMaxRequestDuration    time.Duration `hcl:"-"`
	DefaultMaxRequestDurationRaw interface{}   `hcl:"default_max_request_duration"`

	// LogFormat specifies the log format. Valid values are "standard" and
	// "json". The values are case-insenstive. If no log format is specified,
	// then standard format will be used.
	LogFormat string `hcl:"log_format"`
	LogLevel  string `hcl:"log_level"`

	PidFile string `hcl:"pid_file"`

	ClusterName string `hcl:"cluster_name"`
}

// LoadConfigFile loads the configuration from the given file.
func LoadConfigFile(path string) (*SharedConfig, error) {
	// Read the file
	d, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ParseConfig(string(d))
}

func LoadConfigKMSes(path string) ([]*KMS, error) {
	// Read the file
	d, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ParseKMSes(string(d))
}

func ParseConfig(d string) (*SharedConfig, error) {
	// Parse!
	obj, err := hcl.Parse(d)
	if err != nil {
		return nil, err
	}

	// Start building the result
	var result SharedConfig
	if err := hcl.DecodeObject(&result, obj); err != nil {
		return nil, err
	}

	if result.DefaultMaxRequestDurationRaw != nil {
		if result.DefaultMaxRequestDuration, err = parseutil.ParseDurationSecond(result.DefaultMaxRequestDurationRaw); err != nil {
			return nil, err
		}
		result.DefaultMaxRequestDurationRaw = nil
	}

	if result.DisableMlockRaw != nil {
		if result.DisableMlock, err = parseutil.ParseBool(result.DisableMlockRaw); err != nil {
			return nil, err
		}
		result.DisableMlockRaw = nil
	}

	list, ok := obj.Node.(*ast.ObjectList)
	if !ok {
		return nil, fmt.Errorf("error parsing: file doesn't contain a root object")
	}

	if o := list.Filter("hsm"); len(o.Items) > 0 {
		if err := parseKMS(&result.Seals, o, "hsm", 2); err != nil {
			return nil, errwrap.Wrapf("error parsing 'hsm': {{err}}", err)
		}
	}

	if o := list.Filter("seal"); len(o.Items) > 0 {
		if err := parseKMS(&result.Seals, o, "seal", 3); err != nil {
			return nil, errwrap.Wrapf("error parsing 'seal': {{err}}", err)
		}
	}

	if o := list.Filter("kms"); len(o.Items) > 0 {
		if err := parseKMS(&result.Seals, o, "kms", 4); err != nil {
			return nil, errwrap.Wrapf("error parsing 'kms': {{err}}", err)
		}
	}

	if o := list.Filter("entropy"); len(o.Items) > 0 {
		if err := ParseEntropy(&result, o, "entropy"); err != nil {
			return nil, errwrap.Wrapf("error parsing 'entropy': {{err}}", err)
		}
	}

	if o := list.Filter("listener"); len(o.Items) > 0 {
		if err := ParseListeners(&result, o); err != nil {
			return nil, errwrap.Wrapf("error parsing 'listener': {{err}}", err)
		}
	}

	if o := list.Filter("telemetry"); len(o.Items) > 0 {
		if err := ParseTelemetry(&result, o); err != nil {
			return nil, errwrap.Wrapf("error parsing 'telemetry': {{err}}", err)
		}
	}

	entConfig := &(result.EntSharedConfig)
	if err := entConfig.ParseConfig(list); err != nil {
		return nil, errwrap.Wrapf("error parsing enterprise config: {{err}}", err)
	}

	return &result, nil
}

// Sanitized returns a copy of the config with all values that are considered
// sensitive stripped. It also strips all `*Raw` values that are mainly
// used for parsing.
//
// Specifically, the fields that this method strips are:
// - KMS.Config
// - Telemetry.CirconusAPIToken
func (c *SharedConfig) Sanitized() map[string]interface{} {
	if c == nil {
		return nil
	}

	result := map[string]interface{}{
		"disable_mlock": c.DisableMlock,

		"default_max_request_duration": c.DefaultMaxRequestDuration,

		"log_level":  c.LogLevel,
		"log_format": c.LogFormat,

		"pid_file": c.PidFile,

		"cluster_name": c.ClusterName,
	}

	// Sanitize listeners
	if len(c.Listeners) != 0 {
		var sanitizedListeners []interface{}
		for _, ln := range c.Listeners {
			cleanLn := map[string]interface{}{
				"type":   ln.Type,
				"config": ln.RawConfig,
			}
			sanitizedListeners = append(sanitizedListeners, cleanLn)
		}
		result["listeners"] = sanitizedListeners
	}

	// Sanitize seals stanza
	if len(c.Seals) != 0 {
		var sanitizedSeals []interface{}
		for _, s := range c.Seals {
			cleanSeal := map[string]interface{}{
				"type":     s.Type,
				"disabled": s.Disabled,
			}
			sanitizedSeals = append(sanitizedSeals, cleanSeal)
		}
		result["seals"] = sanitizedSeals
	}

	// Sanitize telemetry stanza
	if c.Telemetry != nil {
		result["telemetry"] = SanitizeTelemetry(c)
	}

	return result
}
