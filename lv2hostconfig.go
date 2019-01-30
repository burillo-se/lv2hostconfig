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
	Reference float32        `yaml:"referenceLevel`
	Plugins   []lv2PluginRaw `yaml:"plugins"`
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
		0,
		make([]lv2PluginRaw, 0),
	}
}

func newLV2PluginRaw() lv2PluginRaw {
	return lv2PluginRaw{
		"",
		make(map[string]string),
	}
}

func dbToLinear(db float32) float32 {
	return float32(math.Pow(10, float64(db)/20.0))
}

func linearToDb(linear float32) float32 {
	if linear != 0 {
		return 20 * float32(math.Log10(float64(linear)))
	}
	return -144
}

func getFloat(val interface{}) (float32, error) {
	floatType := reflect.TypeOf(float32(0))
	stringType := reflect.TypeOf("")
	v := reflect.ValueOf(val)
	v = reflect.Indirect(v)
	if v.Type().ConvertibleTo(floatType) {
		fv := v.Convert(floatType)
		return float32(fv.Float()), nil
	} else if v.Type().ConvertibleTo(stringType) {
		sv := v.Convert(stringType)
		s := sv.String()
		f64, err := strconv.ParseFloat(s, 32)
		if err != nil {
			return float32(math.NaN()), err
		}
		return float32(f64), nil
	} else {
		return float32(math.NaN()), fmt.Errorf("Can't convert %v to float", v.Type())
	}
}

func setUpLV2HostConfigFuncs(lvc *LV2HostConfig) {
	lvc.FunctionMap["linear"] = func(args ...interface{}) (interface{}, error) {
		if len(args) != 1 {
			return math.NaN(), fmt.Errorf("Function 'linear' expects exactly 1 argument")
		}
		db, err := getFloat(args[0])
		if err != nil {
			return math.NaN(), fmt.Errorf("Value '%v' was not a float", args[0])
		}
		return dbToLinear(db), nil
	}
	lvc.FunctionMap["decibel"] = func(args ...interface{}) (interface{}, error) {
		if len(args) != 1 {
			return math.NaN(), fmt.Errorf("Function 'decibel' expects exactly 1 argument")
		}
		linear, err := getFloat(args[0])
		if err != nil {
			return math.NaN(), fmt.Errorf("Value '%v' was not a float", args[0])
		}
		return linearToDb(linear), nil
	}
	lvc.FunctionMap["min"] = func(args ...interface{}) (interface{}, error) {
		if len(args) != 2 {
			return math.NaN(), fmt.Errorf("Function 'min' expects exactly 2 arguments")
		}
		a, err := getFloat(args[0])
		if err != nil {
			return math.NaN(), fmt.Errorf("Value '%v' was not a float", args[0])
		}
		b, err := getFloat(args[1])
		if err != nil {
			return math.NaN(), fmt.Errorf("Value '%v' was not a float", args[0])
		}
		return math.Min(float64(a), float64(b)), nil
	}
	lvc.FunctionMap["max"] = func(args ...interface{}) (interface{}, error) {
		if len(args) != 2 {
			return math.NaN(), fmt.Errorf("Function 'max' expects exactly 2 arguments")
		}
		a, err := getFloat(args[0])
		if err != nil {
			return math.NaN(), fmt.Errorf("Value '%v' was not a float", args[0])
		}
		b, err := getFloat(args[1])
		if err != nil {
			return math.NaN(), fmt.Errorf("Value '%v' was not a float", args[0])
		}
		return math.Max(float64(a), float64(b)), nil
	}
	lvc.FunctionMap["abs"] = func(args ...interface{}) (interface{}, error) {
		if len(args) != 1 {
			return math.NaN(), fmt.Errorf("Function 'abs' expects exactly 1 argument")
		}
		v, err := getFloat(args[0])
		if err != nil {
			return math.NaN(), fmt.Errorf("Value '%v' was not a float", args[0])
		}
		return math.Abs(float64(v)), nil
	}
	lvc.FunctionMap["sqrt"] = func(args ...interface{}) (interface{}, error) {
		if len(args) != 1 {
			return math.NaN(), fmt.Errorf("Function 'sqrt' expects exactly 1 argument")
		}
		v, err := getFloat(args[0])
		if err != nil {
			return math.NaN(), fmt.Errorf("Value '%v' was not a float", args[0])
		}
		return math.Sqrt(float64(v)), nil
	}
	lvc.FunctionMap["pow"] = func(args ...interface{}) (interface{}, error) {
		if len(args) != 2 {
			return math.NaN(), fmt.Errorf("Function 'pow' expects exactly 2 arguments")
		}
		a, err := getFloat(args[0])
		if err != nil {
			return math.NaN(), fmt.Errorf("Value '%v' was not a float", args[0])
		}
		b, err := getFloat(args[1])
		if err != nil {
			return math.NaN(), fmt.Errorf("Value '%v' was not a float", args[0])
		}
		return math.Pow(float64(a), float64(b)), nil
	}
	lvc.FunctionMap["scale"] = func(args ...interface{}) (interface{}, error) {
		if len(args) != 5 {
			return math.NaN(), fmt.Errorf("Function 'scale' expects exactly 5 arguments")
		}
		// where is Python when you need it...
		var val, oldMin, oldMax, newMin, newMax float32
		floatPtrs := []*float32{&val, &oldMin, &oldMax, &newMin, &newMax}

		for i, arg := range args {
			val, err := getFloat(arg)
			if err != nil {
				return math.NaN(), fmt.Errorf("Value '%v' was not a float", arg)
			}
			*floatPtrs[i] = val
		}
		if oldMin >= oldMax {
			return math.NaN(), fmt.Errorf("Range '%v-%v' is invalid", oldMin, oldMax)
		}
		if newMin >= newMax {
			return math.NaN(), fmt.Errorf("Range '%v-%v' is invalid", newMin, newMax)
		}
		if val < oldMin || val > oldMax {
			return math.NaN(), fmt.Errorf("Value '%v' is not within range '%v-%v'", val, oldMin, oldMax)
		}
		oldScale := oldMax - oldMin
		newScale := newMax - newMin
		oldScaledVal := (val - oldMin) / oldScale
		newScaledVal := newScale * oldScaledVal
		newVal := newScaledVal + newMin

		return newVal, nil
	}
}

// NewLV2HostConfig allocate new host config (usually
// for purposes of setting up its value map parameters)
func NewLV2HostConfig() *LV2HostConfig {
	lvc := LV2HostConfig{
		make([]LV2PluginConfig, 0),
		make(map[string]interface{}),
		make(map[string]govaluate.ExpressionFunction),
	}

	// set up standard functions
	setUpLV2HostConfigFuncs(&lvc)

	return &lvc
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

// ReadFile will read a YAML config into an LV2HostConfig
// data structure. Note that any Data fields will not be
// initialized until Evaluate is called.
func (c *LV2HostConfig) ReadFile(file string) error {
	raw, err := readConfig(file)
	if err != nil {
		return err
	}

	// parsing should be atomic, so operate on a copy
	pcs := make([]LV2PluginConfig, 0)

	// read raw string values into DataFmt
	for _, rpd := range raw.Plugins {
		pc := NewLV2PluginConfig()

		uri := rpd.URI

		pc.PluginURI = uri

		for param, value := range rpd.Data {
			pc.DataFmt[param] = value
		}
		pcs = append(pcs, pc)
	}

	// we're successfully parsed plugin data, so clear current contents
	// and overwrite them with parsed data
	c.Plugins = pcs
	c.ValueMap["reference"] = raw.Reference

	return nil
}

// Evaluate uses govaluate to (re-)parse contents of
// config structure into actual values.
func (c *LV2HostConfig) Evaluate() error {
	// parsing should be atomic, so operate on a copy
	pcs := make([]LV2PluginConfig, 0)

	// use govaluate to parse our values
	for _, pd := range c.Plugins {
		pc := NewLV2PluginConfig()

		uri := pd.PluginURI

		pc.PluginURI = uri

		for param, value := range pd.DataFmt {
			// keep current DataFmt to enable future re-parsing
			pc.DataFmt[param] = value

			// if we can parse value as float, there is no expression
			result64, err := strconv.ParseFloat(value, 32)
			if err == nil {
				pc.Data[param] = float32(result64)
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
		}

		pcs = append(pcs, pc)
	}

	// we're successfully parsed plugin data, so clear current contents
	// and overwrite them with parsed data
	c.Plugins = pcs

	return nil
}

// WriteToFile will write LV2HostConfig data back into
// YAML form. Note that Data contents is not dumped into
// YAML - DataFmt is dumped instead. Therefore, any changes
// to Data values will not be reflected in the YAML file
// unless DataFmt was changed accordingly.
func (c *LV2HostConfig) WriteToFile(file string) error {
	raw := newLV2HostRaw()

	for _, pcfg := range c.Plugins {
		rawp := newLV2PluginRaw()
		rawp.URI = pcfg.PluginURI
		for k, v := range pcfg.DataFmt {
			rawp.Data[k] = v
		}
		raw.Plugins = append(raw.Plugins, rawp)
	}

	return writeConfig(raw, file)
}
