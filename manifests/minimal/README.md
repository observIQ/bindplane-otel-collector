# Minimal manifest

This manifest contains the minimal components needed to operate with the [opampsupervisor](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/cmd/opampsupervisor) and [BindPlane OP](https://observiq.com/).

You can use this manifest as a base when constructing your own custom distribution.

## Components

This is a list of components that will be available to use in the resulting collector binary.

| extensions           | exporters   | processors | receivers   | connectors |
| :------------------- | :---------- | :--------- | :---------- | :--------- |
| healthcheckextension | nopexporter |            | nopreceiver |            |
| opampextension       |             |            |             |            |
| bindplaneextension   |             |            |             |            |
