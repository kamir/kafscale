# Spark KafScale Word Count Demo (E40)

This demo runs a Spark Structured Streaming word count over Kafka records, counting words separately for headers, keys, and values. It also tracks missing fields in stats (`no-key`, `no-header`, `no-value`).

## Prerequisites

- Java 11+
- Apache Spark 3.5+ (set `SPARK_HOME` or have `spark-submit` on PATH)
- KafScale local demo or platform demo

## Profiles

The demo uses lightweight profiles to select the broker address.

- `default`: local broker on `127.0.0.1:39092`
- `cluster`: in-cluster broker at `kafscale-broker:9092`
- `local-lb`: local app + remote broker via LB/port-forward on `localhost:59092`

Set the profile with `KAFSCALE_SETUP_PROFILE` or `--profile=...`.

## Step 1: Run locally with the make demo setup

Start the local demo:

```bash
make demo
```

In a new terminal, run the Spark job:

```bash
cd examples/E40_spark-kafscale-demo
make run-jar-standalone
```

Override profile or Spark master:

```bash
KAFSCALE_SETUP_PROFILE=local-lb SPARK_MASTER=local[2] make run-jar-standalone
```

The job listens on `demo-topic-1` by default. Override with `KAFSCALE_TOPIC` if needed.

## Configuration

You can override defaults with environment variables:

- `KAFSCALE_BOOTSTRAP_SERVERS`
- `KAFSCALE_TOPIC`
- `KAFSCALE_GROUP_ID`
- `KAFSCALE_STARTING_OFFSETS` (`latest` or `earliest`)
- `KAFSCALE_SPARK_UI_PORT`
- `KAFSCALE_CHECKPOINT_DIR`

## Output format

The job prints running counts:

```
header | authorization => 5
key | order => 12
value | widget => 9
stats | no-key => 3
```

## Verify the job

- Spark UI: `http://localhost:4040` (or `KAFSCALE_SPARK_UI_PORT`)
