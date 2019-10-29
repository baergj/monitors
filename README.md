# monitors

xrandr-wrapping layout engine using monitor serial numbers for identification.

## Usage
```text
Usage of ./monitors:
  -config-path string
        path to the config file (default ".config/monitors/config.json")
  -pretend
        print what would have been executed and exit
  -xdisplay string
        which X display to manage (default ":0")
```

In its default mode, `monitors` reads displays and layouts from the config file, calls out to xrandr to figure out which displays are currently connected, and then evaluates the configured list of layouts until it finds one that matches the connected set of displays. If it finds a matching layout, it enables all the displays in the layout, using the layout's positions for those displays, and then disables all other displays.

Using `-pretend` does all of the above except for the final xrandr callout to enact changes. It's highly recommended that you use this option as you develop your configuration.

Any xrandr displays that aren't matched against the given configuration will be noted along with their serial number to make it easy to add them to the config, if desired.

All output is emitted via golang's `log` package.

### Config file format
```json
{
    "displays": [
        { "name": "my-left-monitor", "serial": 12345 },
        { "name": "my-right-monitor", "serial": 67890 },
        { "name": "laptop", "serial": 0, "is-laptop": true }
    ],
    "layouts": [
        { "name": "desk", "displays": [
            { "display": "my-left-monitor", "primary": true },
            { "display": "my-middle-monitor", "relative-locations": [ { "location": "right-of", "display": "my-left-monitor" } ] },
            { "display": "laptop", "relative-locations": [ { "location": "right-of", "display": "my-middle-monitor" } ] }
        ] },
        { "name": "laptop", "displays": [ { "display": "laptop", "primary": true } ] }
    ]
}
```

Some laptop displays don't have serial numbers (i.e. their serial number is 0). `is-laptop` can be set to make such a display identifiable from another display that may have a serial number of 0 (though hopefully there's not more than one!).

For the meaning behind locations or the `primary` flag, please see the xrandr manpage.

Layouts are evaluated in the order given. Thus, any layouts that are subsets of another layout should be listed after the superset, otherwise the superset will never be chosen (since the subset will match first).
