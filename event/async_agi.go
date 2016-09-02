package event

import (
	"net/url"
	"reflect"
	"strings"

	"github.com/party79/gami"
)

type AgiEnv map[string]string

func newAgiEnv(env string) *AgiEnv {
	ae := &AgiEnv{}
	var envStr string
	envStr, _ = url.QueryUnescape(env)
	lines := strings.Split(strings.TrimSpace(envStr), "\n")
	for _, v := range lines {
		line := strings.SplitN(strings.TrimSpace(v), ":", 2)
		k := strings.TrimSpace(line[0])
		v := strings.TrimSpace(line[1])
		ae.Set(k, v)
	}
	return ae
}

func (ae *AgiEnv) Set(key, value string) {
	if ae == nil {
		reflect.ValueOf(&ae).Elem().Set(reflect.ValueOf(&AgiEnv{}))
	}
	(*ae)[key] = value
}
func (ae *AgiEnv) Get(key string) string {
	if ae == nil {
		return ""
	}
	if v, ok := (*ae)[key]; ok {
		return v
	}
	return ""
}
func (ae *AgiEnv) Del(key string) {
	delete(*ae, key)
}

type AsyncAGI struct {
	Privilege []string
	Event     string  `AMI:"Event"`
	Channel   string  `AMI:"Channel"`
	SubEvent  string  `AMI:"Subevent"`
	CommandID string  `AMI:"Commandid"`
	Result    string  `AMI:"Result"`
	Env       string  `AMI:"Env"`
	ResultStr string  `AMI:"-"`
	EnvMap    *AgiEnv `AMI:"-"`
}

func (e *AsyncAGI) BuildEvent(event *gami.AMIEvent) {
	if e != nil {
		ret := Build(event, reflect.TypeOf(e))
		if e2, ok := ret.(*AsyncAGI); ok && e2 != nil {
			reflect.ValueOf(e).Elem().Set(reflect.ValueOf(e2).Elem())
		}
		if e.Result != "" {
			e.ResultStr, _ = url.QueryUnescape(e.Result)
		}
		if e.Env != "" {
			e.EnvMap = newAgiEnv(e.Env)
		}
	}
}

func init() {
	RegisterEvent((*AsyncAGI)(nil), "AsyncAGI")
}
