# Snowflake Exporter

This exporter allows logs, metrics, and traces to be sent to Snowflake, a cloud data warehouse service. This exporter utilizes the Go Snowflake Driver to send telemetry to a database in Snowflake.

## Minimum Collector Versions

- Introduced: [v1.45.0](https://github.com/observIQ/bindplane-agent/releases/tag/v1.45.0)

## Supported Pipelines

- Logs
- Metrics
- Traces

## Prerequisites

- Pre-existing Snowflake [data warehouse](https://docs.snowflake.com/en/user-guide/warehouses-overview)
- Snowflake user with [appropriate privileges](#granting-snowflake-privileges)

## How It Works

1. The exporter uses the configured credentials to connect to the Snowflake account.
2. The exporter initializes any needed resources (database, schemas, tables).
3. As the exporter receives telemetry, it flattens it out and then sends to Snowflake using a batch insert.

## Configuration

The exporter can be configured using the following fields:

| Field | Type | Default | Required | Description |
| ----- | ---- | ------- | -------- | ----------- |
| account_identifier | string | | `true` | The account identifier number of the Snowflake account telemetry data will be sent to. |
| username | string | | `true` | The username for the account the exporter will use to authenticate with Snowflake. |
| password | string | | `true` | The password for the account the exporter will use to authenticate with Snowflake. |
| warehouse | string | | `true` | The Snowflake data warehouse that should be used for storing data. |
| role | string | | `false` | The Snowflake role that the exporter should use to have the correct permissions. Only necessary if the default role of the given user is not the necessary role. |
| database | string | `otlp` | `false` | The database in Snowflake that the exporter will store telemetry data in. Will create it if it doesn't exist. |
| parameters | map | | `false` | A map of optional connection parameters that may be used for connecting with Snowflake. The exporter uses `client_session_keep_alive` by default. For more information, see this [Snowflake documentation](https://docs.snowflake.com/en/sql-reference/parameters) |
| logs.schema | string | `logs` | `false` | The name of the schema to use to store the log table in. |
| logs.table | string | `data` | `false` | The name of the table that logs will be stored in. |
| metrics.schema | string | `metrics` | `false` | The name of the schema to use to store the metric tables in. |
| metrics.table | string | `data` | `false` | The prefix to use for the tables that metrics will be stored in. |
| traces.schema | string | `traces` | `false` | The name of the schema to use to store the trace table in. |
| traces.table | string | `data` | `false` | The name of the table that traces will be stored in. |

This exporter can also be configured to use "Retry on Failure", "Sending Queue", and "Timeout". More information about these options can be found [here](https://github.com/open-telemetry/opentelemetry-collector/blob/main/exporter/exporterhelper/README.md) and examples in the configurations below.

The exporter performs best when used in conjunction with the [Batch processor](https://github.com/open-telemetry/opentelemetry-collector/blob/main/processor/batchprocessor/README.md). 

### Metrics

Each type of metric (exponential histogram, gauge, histogram, sum, summary) will have their own table created inside the metric schema. The `metrics.table` configuration variable will set the prefix that is used for the tables as each metric table name is appended with its type. For example, the default table name for gauges is `data_gauge`. The exact postfixes used for each metric type is below:

| Metric Type | Table Postfix | Default Table Name |
| ----------- | ------------- | ------------------ |
| Exponential Histogram | `_exponential_histogram` | `data_exponential_histogram` |
| Gauge | `_gauge` | `data_gauge` |
| Histogram | `_histogram` | `data_histogram` |
| Sum | `_sum` | `data_sum` |
| Summary | `_summary` | `data_summary` |

## Granting Snowflake Privileges

Snowflake's access control consists of a combination of Discretionary Access Control (DAC) and Role Based Access Control (RBAC). In order for this exporter to successfully send telemetry data to Snowflake, the user it is configured with needs to have the appropriate permissions. The following sections will outline how to configure Snowflake's access control policy for this exporter. For more information on Snowflake's access control framework, see this [Snowflake documentation](https://docs.snowflake.com/en/user-guide/security-access-control-overview).

### Recommended Approach

The following instructions detail how to configure a role in Snowflake with privileges needed by the exporter, and how to create a new user with access to that role the exporter can authenticate with. Snowflake has a variety of ways to connect to it, but these instructions will be tailored for "Classic Console" as all accounts have access to it.

Before starting, log in to Classic Console using a user that has access to the `ACCOUNTADMIN` role or another role in you Snowflake account that has permission to grant privileges and create users. If the default role is not the required one, then you'll need to assume that role using this SQL command (replace the role as needed):

```sql
ASSUME ROLE "ACCOUNTADMIN";
```

These instructions will grant privileges to one of the default roles Snowflake is initialized with, `SYSADMIN`. If you want to grant privileges to a different role then just switch out `SYSADMIN` for your role in the SQL commands.

#### 1. Grant Warehouse Usage

First, we need to grant the `USAGE` privilege to the `SYSADMIN` role on the data warehouse telemetry data will be stored in. Run this SQL command next (replace `TEST` with your warehouse name):

```sql
GRANT USAGE ON WAREHOUSE "TEST" TO ROLE "SYSADMIN";
```

#### 2. Grant Create Database Privilege

Next the `SYSADMIN` role needs to be granted the ability to create databases in the Snowflake account. Run the following SQL to do so:

```SQL
GRANT CREATE DATABASE ON ACCOUNT TO ROLE "SYSADMIN";
```

#### 3. Create New User For BindPlane

Now a new user needs to be created that the BindPlane Agent can login as. The user should also have the default role assigned as `SYSADMIN`, although it isn't necessary. 

**Note:** If the default role is not assigned, then the exporter will need to be configured with the correct role to work. 

Remember the login name and password you use and configure the exporter with these values. Replace the user, password, and login name in the following SQL to match yours:

```sql
CREATE USER BP_AGENT PASSWORD="password" LOGIN_NAME="BP_AGENT" DEFAULT_ROLE="SYSADMIN";
```

#### 4. Grant Privilege to SYSADMIN Role

Even though the default role was set as `SYSADMIN` we still need to grant the new account permission to it. This can be done using the next SQL command (replace user as needed):

```sql
GRANT ROLE "SYSADMIN" TO USER BP_AGENT;
```

Now we have a Snowflake user with the correct permissions to be able to create a database, schemas, and tables and also use the configured warehouse to store telemetry data in. 

### Alternatives

You can take an alternative approach to the one outlined above if you prefer to configure Snowflake's access control differently. The recommended approach above is meant to get telemetry flowing into Snowflake with limited time spent configuring. Note that this exporter will require the following privileges to work correctly. In most cases, the `ALL` keyword should be used instead of `FUTURE` so that the exporter can integrate with any pre-existing resources. 

- USE on warehouse
- CREATE, USE DATABASES on account
- CREATE, USE SCHEMAS on databases
- CREATE, INSERT, SELECT TABLES on schemas

For more information on Snowflake's various privileges and subsequent commands, see this [Snowflake documentation](https://docs.snowflake.com/en/sql-reference/commands-user-role).

## Example Configurations

### Basic Configuration With Default Schemas & Tables

This configuration includes only the required fields. It will use the default database, `otlp`, and use the default schema and table names for whichever telemetry pipelines it is included in.

```yaml
snowflake:
    account_identifier: "account_id"
    username: "bp_agent"
    password: "password"
    warehouse: "TEST"
```

### Full Custom Configuration

This configuration includes all fields specific to this exporter. Custom database, schema, and table names will be used.

```yaml
snowflake:
    account_identifier: "account_id"
    username: "bp_agent"
    password: "password"
    warehouse: "TEST"
    role: "SYSADMIN"
    database: "db"
    logs:
        schema: "file_logs"
        table: "log_data"
    metrics:
        schema: "host_metrics"
        table: "metric_data"
    traces:
        schema: "my_traces"
        table: "trace_data"
```

### Basic Configuration With Exporter Helpers

This configuration uses some of the exporter helper configuration options and some non-default schemas and tables.

```yaml
snowflake:
    account_identifier: "account_id"
    username: "bp_agent"
    password: "password"
    warehouse: "TEST"
    logs:
        schema: "file_logs"
    metrics:
        table: "host_metrics"
    traces:
        schema: "my_traces"
        table: "trace_data"
    timeout: 10s
    retry_on_failure:
        enabled: true
    sending_queue:
        enabled: true
```
