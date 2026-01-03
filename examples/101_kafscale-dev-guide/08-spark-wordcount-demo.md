# Spark Word Count Demo (E40)

This section adds a Spark Structured Streaming word count job that consumes from KafScale and keeps separate counts for headers, keys, and values. It also tracks `no-key`, `no-header`, and `no-value` stats.

## Step 1: Run locally with the make demo setup

Start the local demo:

```bash
make demo
```

Run the Spark job:

```bash
cd examples/E40_spark-kafscale-demo
make run-jar-standalone
```

Override profile or Spark master:

```bash
KAFSCALE_SETUP_PROFILE=local-lb SPARK_MASTER=local[2] make run-jar-standalone
```

## Profiles

The Spark job uses the same three profiles as the Spring Boot app:

- `default` → `127.0.0.1:39092`
- `cluster` → `kafscale-broker:9092`
- `local-lb` → `localhost:59092`

Set the profile with `KAFSCALE_SETUP_PROFILE` or `--profile=...`.

## Configuration

You can override defaults with environment variables:

- `KAFSCALE_BOOTSTRAP_SERVERS`
- `KAFSCALE_TOPIC`
- `KAFSCALE_GROUP_ID`
- `KAFSCALE_STARTING_OFFSETS` (`latest` or `earliest`)
- `KAFSCALE_SPARK_UI_PORT`
- `KAFSCALE_CHECKPOINT_DIR`

## Verify the job

- Spark UI: `http://localhost:4040` (or `KAFSCALE_SPARK_UI_PORT`)
