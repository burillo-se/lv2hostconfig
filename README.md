# lv2hostconfig
Config parser for LV2 host

If you don't know why you need this, you don't.

However, just in case you're curious how it works, here it is in a nutshell.

This is intended for use with [lv2host-go](https://github.com/burillo-se/lv2host-go/lv2host/) library, to make it
easy to configure LV2 plugins without writing loads of code to do it.

At its core, it is a YAML parser. Format is as follows:

```
plugins:
- pluginUri: <LV2 URI of plugin you want to load>
  parameters:
    <param symbol>: <param value as float string, e.g. "123.45", with quotes>
```

Example:

```
plugins:
- pluginUri: myuri
  parameters:
    test1: "123.000000"
    test2: "234.000000"
- pluginUri: myuri2
  parameters:
    test3: "345.000000"
    test4: "456.000000"
```

You can then use these parameters to load plugins and set their parameters with Go bindings for LV2Host.

However, the config parser is not *just* a config parser. It also uses govaluate to specify parameters in
a declarative manner, for example:

    knee: "10 + 5"

This might seem trivial, but the plugin config data also has a value map and a function map, to use with
govaluate. For example, you could set value "myvalue" to 10, and rewrite your config as this:

    knee: "myvalue + 5"

This, given `myvalue`'s value of 10, will evaluate to 15, and that's the value that will be stored in
the LV2 config structure. Keep in mind that standard govaluate escaping rules apply.

But wait, there's more! There is also a number of utility functions provided within the config library. Usage
of these functions is done in a similar, declarative way:

    knee: "sqrt(9)"

If `sqrt` was defined as a function that returns a square root of whatever argument supplied, then the value
stored in config would be equal to 3.

List of utility functions provided is as follows:

-   decibel(value) - will convert a float value to decibels
-   linear(value) - will convert a decibel value to float
-   min(a, b), max(a, b), abs(a), sqrt(a), pow(a, b) - self-explanatory
-   scale(val, orig_min, orig_max, new_min, new_max) - scale value `val` from range `orig_min`-`orig_max` to
    fit into the new range `new_min`-`new_max`

Keep in mind that while you're allowed to modify values inside the config and even write it out, any formatted
values will *not* have their changed values reflected in the resulting YAML. So, if your initial parameter value
was like this:

    knee: "10"

and you've overwritten the value, the YAML will have the new value written out. If, however, your initial parameter
value was like this:

    knee: "myvar + func() - 10"

(or in fact anything that isn't parseable as float32), then the new value will *not* be written out to the YAML
config. If you want to change data in such a field, change its format.
