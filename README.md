# mackerel-plugin-lm-sensors

[`lm-sensors`](https://github.com/lm-sensors/lm-sensors) の計測値を Mackerel のカスタムメトリックとして投稿するプラグインです。

`sensors -j` の出力から、現在値を表す `*_input` と `*_average` を取得します。温度、ファン回転数、電圧、電力、電力量、電流、湿度に対応しています。しきい値（`*_max`、`*_crit` など）やアラームは投稿しません。

## 必要なもの

JSON 出力（`-j`）に対応した `sensors` コマンドが必要です。

Debian / Ubuntu:

```sh
sudo apt-get install lm-sensors
```

RHEL / Fedora:

```sh
sudo dnf install lm_sensors
```

必要に応じて `sudo sensors-detect` を実行し、先に `sensors -j` で値を取得できることを確認してください。

## Usage

```sh
mackerel-plugin-lm-sensors
```

オプション:

```text
-metric-key-prefix string
      Metric key prefix (default "lm_sensors")
-sensors-command string
      Path to the sensors command (default "sensors")
-timeout duration
      Timeout for the sensors command (default 10s)
-version
      Show version
```

## Example

```text
$ mackerel-plugin-lm-sensors
lm_sensors.temperature.coretemp_isa_0000_package_id_0_temp1_input.value  42.500000  1710000000
lm_sensors.fan.nct6798_isa_0290_cpu_fan_fan1_input.value                1247.000000 1710000000
lm_sensors.voltage.nct6798_isa_0290_vcore_in0_input.value               1.224000    1710000000
```

メトリックキーは次の形式です。

```text
lm_sensors.<category>.<chip>_<feature>_<reading>.value
```

ハードウェアや `sensors` の設定によって `<chip>`、`<feature>`、`<reading>` は異なります。

## Install

リリースを公開した後は `mkr` でインストールできます。

```sh
mkr plugin install wakuwakup/mackerel-plugin-lm-sensors
```

## mackerel-agent.conf

```toml
[plugin.metrics.lm_sensors]
command = "/opt/mackerel-agent/plugins/bin/mackerel-plugin-lm-sensors"
```

`sensors` が標準の `PATH` にない場合:

```toml
[plugin.metrics.lm_sensors]
command = "/opt/mackerel-agent/plugins/bin/mackerel-plugin-lm-sensors -sensors-command /usr/bin/sensors"
```

## Build

```sh
go build ./...
go test ./...
```

クロスコンパイル用アーカイブは Docker を使って作成できます。

```sh
./build.sh
```

## License

MIT
