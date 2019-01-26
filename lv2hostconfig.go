package lv2hostconfig

import (
	"fmt"
	"io/ioutil"
	"math"
	"reflect"
	"strconv"

	"github.com/Knetic/govaluate"

	yaml "gopkg.in/yaml.v1"
)

// LV2 config parsing is done in two
// stages, because our YAML file is not a simple
// YAML file - it is also meant to perform some
// declarative calculations over its values (for
// example, transform "$v + 1" to "4" if
// value of 'v' was set to 3).
// This is the first stage: the raw text form.
type lv2HostRaw struct {
	Plugins []lv2PluginRaw `yaml:"plugins"`
}

// LV2PluginRaw is the raw parsed data from a
// YAML config file.
type lv2PluginRaw struct {
	URI  string            `yaml:"pluginUri"`
	Data map[string]string `yaml:"parameters"`
}

func readConfig(file string) (*lv2HostRaw, error) {
	var host lv2HostRaw
	yamlFile, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("Failed to read config: %v", err)
	}
	err = yaml.Unmarshal(yamlFile, &host)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse config: %v", err)
	}

	return &host, nil
}

func writeConfig(hostRaw *lv2HostRaw, file string) error {
	d, err := yaml.Marshal(hostRaw)
	if err != nil {
		return fmt.Errorf("Failed to serialize config: %v", err)
	}
	err = ioutil.WriteFile(file, d, 0644)
	if err != nil {
		return fmt.Errorf("Failed to write config: %v", err)
	}
	return nil
}

// LV2HostConfig is main config structure containing
// plugin configuration. In addition, it also contains
// a parameter map (untyped), as well as govaluate
// expression function map, to enable evaluating arbitrary
// functions as part of config parsing.
type LV2HostConfig struct {
	Plugins     []LV2PluginConfig
	ValueMap    map[string]interface{}
	FunctionMap map[string]govaluate.ExpressionFunction
}

// LV2PluginConfig is plugin config structure. Use
// LV2 symbols to map parameters to values. Also
// contains original formatting for data, in case
// the config would need to be saved back into
// file form.
type LV2PluginConfig struct {
	PluginURI string
	Data      map[string]float32
	DataFmt   map[string]string
}

func newLV2HostRaw() *lv2HostRaw {
	return &lv2HostRaw{
		make([]lv2PluginRaw, 0),
	}
}

func newLV2PluginRaw() lv2PluginRaw {
	return lv2PluginRaw{
		"",
		make(map[string]string),
	}
}

// NewLV2HostConfig allocate new host config (usually
// for purposes of setting up its value map parameters)
func NewLV2HostConfig() *LV2HostConfig {
	return &LV2HostConfig{
		make([]LV2PluginConfig, 0),
		make(map[string]interface{}),
		make(map[string]govaluate.ExpressionFunction),
	}
}

// NewLV2PluginConfig allocate new plugin config
func NewLV2PluginConfig() LV2PluginConfig {
	return LV2PluginConfig{
		"",
		make(map[string]float32),
		make(map[string]string),
	}
}

func getFloat32(val interface{}) (float32, error) {
	t := reflect.TypeOf(float32(0))
	v := reflect.ValueOf(val)
	v = reflect.Indirect(v)
	if !v.Type().ConvertibleTo(t) {
		return float32(math.NaN()), fmt.Errorf("Value is not a float")
	}
	fv := v.Convert(t)
	return float32(fv.Float()), nil
}

// ParseFile parses a specified config file, using
// mapped values.
func (c *LV2HostConfig) ParseFile(file string) error {
	raw, err := readConfig(file)
	if err != nil {
		return err
	}

	// parsing should be atomic, so operate on a copy
	pcs := make([]LV2PluginConfig, 0)

	// use govaluate to parse our values
	for _, rpd := range raw.Plugins {
		pc := NewLV2PluginConfig()

		uri := rpd.URI

		for param, value := range rpd.Data {
			pc.PluginURI = uri

			// if we can parse value as float, there is no expression
			result64, err := strconv.ParseFloat(value, 32)
			if err == nil {
				pc.Data[param] = float32(result64)
				pc.DataFmt[param] = ""

				pcs = append(pcs, pc)
				continue
			}
			// expression failed to parse, so evaluate it
			expr, err := govaluate.NewEvaluableExpressionWithFunctions(value, c.FunctionMap)
			if err != nil {
				return fmt.Errorf("Error parsing expression '%v': %v", value, err)
			}
			evalResult, err := expr.Evaluate(c.ValueMap)
			if err != nil {
				return fmt.Errorf("Error evaluating expression '%v': %v", value, err)
			}

			// we've evaluated the expression, however it may not be a float
			result32, err := getFloat32(evalResult)
			if err != nil {
				return fmt.Errorf("Error parsing expression '%v' result: %v", value, err)
			}
			pc.Data[param] = result32
			pc.DataFmt[param] = value

			pcs = append(pcs, pc)
		}
	}

	// we're successfully parsed plugin data, so clear current contents
	// and overwrite them with parsed data
	c.Plugins = pcs

	return nil
}

// WriteToFile will write LV2HostConfig data back into
// YAML form. Note that for any formatted values, Data
// contents is not dumped into YAML - DataFmt is dumped
// instead. Therefore, for formatted values, any changes
// to Data values will not be reflected in the YAML file
// unless DataFmt was changed. Any value that has DataFmt
// as empty string, will be treated as not formatted.
func (c *LV2HostConfig) WriteToFile(file string) error {
	raw := newLV2HostRaw()

	for _, pcfg := range c.Plugins {
		rawp := newLV2PluginRaw()
		rawp.URI = pcfg.PluginURI
		for k, v := range pcfg.Data {
			if pcfg.DataFmt[k] == "" {
				rawp.Data[k] = fmt.Sprintf("%f", v)
			} else {
				rawp.Data[k] = pcfg.DataFmt[k]
			}
		}
		raw.Plugins = append(raw.Plugins, rawp)
	}

	return writeConfig(raw, file)
}
