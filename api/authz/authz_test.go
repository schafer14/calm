package authz_test

import (
	"encoding/json"
	"errors"
	"testing"
	"timeapi/authz"

	"github.com/casbin/casbin/v3/model"
	"github.com/casbin/casbin/v3/persist"
	"github.com/matryer/is"
)

func TestAuthz(t *testing.T) {

	is := is.New(t)
	must := Must(is)

	b := []byte{}
	adapter := NewAdapter(&b)
	e := authz.New(adapter)

	e.AddRoleForUser("banner", "super_admin")
	e.AddPolicy("angie", "/users/23/new-device-token", "POST")
	e.AddRoleForUser("addy", "admin")

	is.Equal(true, must(e.Enforce("banner", "/users", "GET")))
	is.Equal(true, must(e.Enforce("banner", "/users", "POST")))

	is.Equal(true, must(e.Enforce("banner", "/users/23/new-device-token", "POST")))
	is.Equal(true, must(e.Enforce("angie", "/users/23/new-device-token", "POST")))
	is.Equal(false, must(e.Enforce("angie", "/users/24/new-device-token", "POST")))

	is.Equal(true, must(e.Enforce("addy", "/users", "GET")))
	is.Equal(false, must(e.Enforce("addy", "/users", "POST")))
}

//////
// FIXTURES
// ////

func Must(i *is.I) func(res bool, err error) bool {
	return func(res bool, err error) bool {
		i.NoErr(err)
		return res
	}
}

type CasbinRule struct {
	PType string
	V0    string
	V1    string
	V2    string
	V3    string
	V4    string
	V5    string
}

// Adapter represents the Redis adapter for policy storage.
// It can load policy from JSON bytes or save policy to JSON bytes.
type Adapter struct {
	source *[]byte
	policy []CasbinRule
}

// NewAdapter is the constructor for Adapter.
func NewAdapter(source *[]byte) *Adapter {
	a := Adapter{}
	a.source = source
	a.policy = []CasbinRule{}
	return &a
}

func (a *Adapter) saveToBuffer() error {
	data, err := json.Marshal(a.policy)
	if err == nil {
		*a.source = data
	}
	return err
}

func (a *Adapter) loadFromBuffer() error {
	if len(*a.source) == 0 {
		return nil
	}

	var policy []CasbinRule
	err := json.Unmarshal(*a.source, &policy)
	if err == nil {
		a.policy = policy
	}
	return err
}

func loadPolicyLine(line CasbinRule, model model.Model) {
	lineText := line.PType
	if line.V0 != "" {
		lineText += ", " + line.V0
	}
	if line.V1 != "" {
		lineText += ", " + line.V1
	}
	if line.V2 != "" {
		lineText += ", " + line.V2
	}
	if line.V3 != "" {
		lineText += ", " + line.V3
	}
	if line.V4 != "" {
		lineText += ", " + line.V4
	}
	if line.V5 != "" {
		lineText += ", " + line.V5
	}

	persist.LoadPolicyLine(lineText, model)
}

// LoadPolicy loads policy from database.
func (a *Adapter) LoadPolicy(model model.Model) error {
	err := a.loadFromBuffer()
	if err != nil {
		return err
	}

	for _, line := range a.policy {
		loadPolicyLine(line, model)
	}
	return nil
}

func savePolicyLine(ptype string, rule []string) CasbinRule {
	line := CasbinRule{}

	line.PType = ptype
	if len(rule) > 0 {
		line.V0 = rule[0]
	}
	if len(rule) > 1 {
		line.V1 = rule[1]
	}
	if len(rule) > 2 {
		line.V2 = rule[2]
	}
	if len(rule) > 3 {
		line.V3 = rule[3]
	}
	if len(rule) > 4 {
		line.V4 = rule[4]
	}
	if len(rule) > 5 {
		line.V5 = rule[5]
	}

	return line
}

// SavePolicy saves policy to database.
func (a *Adapter) SavePolicy(model model.Model) error {
	a.policy = []CasbinRule{}

	var lines []CasbinRule

	for ptype, ast := range model["p"] {
		for _, rule := range ast.Policy {
			line := savePolicyLine(ptype, rule)
			lines = append(lines, line)
		}
	}

	for ptype, ast := range model["g"] {
		for _, rule := range ast.Policy {
			line := savePolicyLine(ptype, rule)
			lines = append(lines, line)
		}
	}

	a.policy = lines

	err := a.saveToBuffer()
	return err
}

// AddPolicy adds a policy rule to the storage.
func (a *Adapter) AddPolicy(sec string, ptype string, rule []string) error {
	return errors.New("not implemented")
}

// RemovePolicy removes a policy rule from the storage.
func (a *Adapter) RemovePolicy(sec string, ptype string, rule []string) error {
	return errors.New("not implemented")
}

// RemoveFilteredPolicy removes policy rules that match the filter from the storage.
func (a *Adapter) RemoveFilteredPolicy(sec string, ptype string, fieldIndex int, fieldValues ...string) error {
	return errors.New("not implemented")
}
