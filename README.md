<p align="center">
  <img src=".github/assets/configarr.png" />
</p>

# ConfigArr

`ConfigArr` is a lightweight application designed to manage and update XML configuration files based on environment variables for the `*arr` family (e.g., `Radarr`, `Sonarr`, `Lidarr`, `Prowlarr` ...).

## Usage

### Flags

- `--config`: Path to the XML configuration file (default: `/config/config.xml`).
- `--ignore-missing-config`: Ignore missing configuration file when set to `true`. Otherwise, `configarr` will exit with an error.
- `--prefix`: Prefix for environment variables (default: `CONFIGARR__`).
- `--debug`: Enable debug logging.

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
      - --prefix
      - RADARR__
    volumeMounts:
      - name: config
        mountPath: /config
```

### Environment Variables

Use environment variables prefixed with your specified prefix to update XML configurations following the format `<PREFIX><IDENTIFIER>=<PROPERTY>=<VALUE>`. The `IDENTIFIER` is only used for readability and can be any string. The `PROPERTY` and `VALUE` are the key and value of the property to be updated in the XML configuration file.

For example:

```bash
export CONFIGARR__LOGGING=LogLevel=debug
export CONFIGARR__LAUNCHBROWSER=LaunchBrowser=False
```

In the examples above:

- `CONFIGARR__LOGGING=LogLevel=debug` updates the `<LogLevel>` element in the XML to `debug`.
- `CONFIGARR__LAUNCHBROWSER=LaunchBrowser=False` updates the `<LaunchBrowser>` element in the XML to `False`.
