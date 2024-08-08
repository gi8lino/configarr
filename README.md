# ConfigArr

`ConfigArr` is a lightweight application designed to manage and update XML configuration files based on environment variables for the `*arr` family (e.g., `Radarr`, `Sonarr`).

## Usage

### Flags

- `--config`: Path to the XML configuration file (default: `/config/config.xml`).
- `--prefix`: Prefix for environment variables (default: `CONFIGARR__`).
- `--silent`: Suppress output when set to `true`.

### initContainer

The following is an example of how to use `ConfigArr` as an init container in a Kubernetes pod:

```yaml
initContainers:
  - name: configarr
    image: ghcr.io/gi8lino/configarr:latest
    env:
      - name: RADARR__LAUNCHBROWSER
        value: LaunchBrowser=False
    args:
      - --prefix RADARR__
    volumeMounts:
      - name: config
        mountPath: /config
```

### Environment Variables

Use environment variables prefixed with your specified prefix to update XML configurations following the format `<PREFIX>__<IDENTIFIER>=<PROPERTY>=<VALUE>`. The `IDENTIFIER` is only used for readability and can be any string. The `PROPERTY` and `VALUE` are the key and value of the property to be updated in the XML configuration file.

For example:

```bash
export CONFIGARR__LOGGING=LogLevel=debug
export CONFIGARR__LAUNCHBROWSER=LaunchBrowser=False
```

In the examples above:

- `CONFIGARR__LOGGING=LogLevel=debug` updates the `<LogLevel>` element in the XML to `debug`.
- `CONFIGARR__LAUNCHBROWSER=LaunchBrowser=False` updates the `<LaunchBrowser>` element in the XML to `False`.
