<p align="center">
  <img src=".github/assets/configarr.png" />
</p>

# ConfigArr

ConfigArr updates settings in XML files before an application starts. Use it with Radarr, Sonarr, or other applications that store their settings in XML.

You choose the setting and its new value. ConfigArr keeps the rest of the file as it is, including comments and formatting.

## Quick start

Suppose Radarr has this `config.xml`:

```xml
<Config>
  <LogLevel>info</LogLevel>
  <LaunchBrowser>True</LaunchBrowser>
</Config>
```

To turn off the browser launch, run:

```sh
configarr config.xml \
  --update.browser.path=/Config/LaunchBrowser \
  --update.browser.value=False
```

`LaunchBrowser` is now `False`. `LogLevel` stays `info`.

- `path` tells ConfigArr where to find the setting in the XML.
- `value` is the new text to put there.
- `browser` is just a name/identifier for this update. You can choose any name.

To change more settings, add another update with a different name:

```sh
configarr config.xml \
  --update.browser.path=/Config/LaunchBrowser \
  --update.browser.value=False \
  --update.logging.path=/Config/LogLevel \
  --update.logging.value=debug
```

## Update several files

List the files to apply the same updates to each one:

```sh
configarr /radarr/config.xml /sonarr/config.xml \
  --update.browser.path=/Config/LaunchBrowser \
  --update.browser.value=False
```

You can also use `--file` instead of positional filenames. Repeat it for more files:

```sh
configarr --file=/radarr/config.xml --file=/sonarr/config.xml \
  --update.browser.path=/Config/LaunchBrowser \
  --update.browser.value=False
```

You can combine both forms; files listed with `--file` are processed first. A filename containing spaces should be quoted. Use `--` before positional filenames beginning with a dash.

Files are updated one at a time. If a file fails, ConfigArr stops; files already updated keep their changes. `--ignore-missing-config` skips missing files and continues with the others.

## Use an environment variable

Use `valueFromEnv` when the new value is already in an environment variable:

```sh
export RADARR_LOG_LEVEL=debug

configarr config.xml \
  --update.logging.path=/Config/LogLevel \
  --update.logging.valueFromEnv=RADARR_LOG_LEVEL
```

Use either `value` or `valueFromEnv` for each update. A referenced variable must exist; an empty variable clears the setting. You can also clear it with `--update.logging.value=`.

## Radarr init container

Add this under `spec` in your Radarr pod, or under `spec.template.spec` in its Deployment or StatefulSet. It changes the configuration before Radarr starts:

```yaml
initContainers:
  - name: configarr
    image: ghcr.io/gi8lino/configarr:latest
    args:
      - /config/config.xml
    env:
      - name: CONFIGARR__UPDATE_BROWSER_PATH
        value: /Config/LaunchBrowser
      - name: CONFIGARR__UPDATE_BROWSER_VALUE
        value: "False"
    volumeMounts:
      - name: config
        mountPath: /config
```

Use the same `config` volume that Radarr mounts at `/config`. It must be writable and already contain `config.xml`. For a new installation, let Radarr start once to create that file before adding the init container.

The environment variables above do the same thing as the quick-start command. For another update, replace `BROWSER` with a different name, such as `LOGGING`. Use `CONFIGARR__UPDATE_LOGGING_VALUEFROMENV` to reference another environment variable instead of supplying a value directly.

## Choosing a setting

Paths follow the names in your XML, including uppercase and lowercase letters:

| Setting                                  | Example path                                     |
| ---------------------------------------- | ------------------------------------------------ |
| A setting directly inside the root       | `/Config/LogLevel`                               |
| A nested setting                         | `/Settings/Database/Host`                        |
| An attribute, such as `host="localhost"` | `/Settings/Database/@host`                       |
| The second item with the same name       | `/Settings/Servers/Server[2]/Port`               |
| An item selected by its name attribute   | `/Settings/Servers/Server[@name='primary']/Port` |

A path must point to exactly one existing setting. ConfigArr does not add or delete settings. If a path is missing, matches more than one setting, or the XML is invalid, the command fails and leaves the file unchanged.

## See what changed

Each changed setting gets one line. Values stay hidden by default:

```text
/config/config.xml: /Config/LaunchBrowser: updated
/config/config.xml: /Config/LogLevel: updated
```

Add `--show-values` to include the old and new values:

```text
/config/config.xml: /Config/LaunchBrowser: "True" -> "False"
/config/config.xml: /Config/LogLevel: "info" -> "debug"
```

This can expose passwords or API keys, including values supplied through environment variables. Quotes and line breaks inside values are escaped to keep each change on one line. Settings that already have the requested value are omitted.

For scripts, use `--output=json`. It writes one JSON object per line:

```json
{
  "file": "/config/config.xml",
  "status": "updated",
  "path": "/Config/LogLevel"
}
```

Combine it with `--show-values` to include `before` and `after` fields. Files with no changes get an `unchanged` record; missing files skipped with `--ignore-missing-config` get a `skipped` record. JSON records go to standard output; errors go to standard error. If a later file fails, records for earlier completed files are still printed.

## Options

| Option                    | What it does                                                              |
| ------------------------- | ------------------------------------------------------------------------- |
| `--file`                  | XML file to update. Can be repeated, or replaced by positional filenames. |
| `--ignore-missing-config` | Skip missing files and continue with the others.                          |
| `--output=text\|json`     | Choose readable text or one JSON record per line.                         |
| `--show-values`           | Include old and new values in reports. May expose secrets.                |
| `--help`                  | Show all available options.                                               |

Supply at least one file. `CONFIGARR__FILE` can provide a filename when none is passed on the command line. Command-line files take priority over this variable.

## License

Licensed under the [Apache License 2.0](LICENSE).
