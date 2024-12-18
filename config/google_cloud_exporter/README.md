# Google Cloud Exporter Prerequisites

The Google Cloud Exporter supports Metrics, Logs, and Traces. This directory contains sub directories with usercase
specific configurations, all of which are compatible with Google Cloud Monitoring.

## Google Cloud APIs

Enable the following APIs.

- Cloud Metrics
- Cloud Logging
- Cloud Tracing

To learn more about enabling APIs, see the [documentation](https://cloud.google.com/endpoints/docs/openapi/enable-api).

## Authentication

The Google Cloud exporter supports three authentication mechanisms.

**Google Application Default Credentials**

When running within Google Cloud, instances with the correct [scopes](https://developers.google.com/identity/protocols/oauth2/scopes#monitoring) for monitoring, logging and tracing will have
the ability to send telemetry to Google Cloud without further configuration. Simply define the Google Cloud exporter in your configuration.

```yaml
exporters:
  googlecloud:
```

**Credential File**

A Google Cloud Service Account can used for authentication by creating a service account and key.

- [Create a service account](https://cloud.google.com/iam/docs/creating-managing-service-accounts) with the following roles:
  - Metrics: `roles/monitoring.metricWriter`
  - Logs: `roles/logging.logWriter`
  - Traces: `roles/cloudtrace.agent`
- [Create a service account json key](https://cloud.google.com/iam/docs/creating-managing-service-account-keys) and place it on the system that is running the agent.

The `credentials_file` parameter can be set:

```yaml
exporters:
  googlecloud:
    credentials_file: /opt/bindplane-otel-collector/credentials.json
```

**Credential JSON**

Using a credential JSON is an alternative to credential file. Instead of placing the service account key file on the system, it is embedded into the config. The service
account setup and credential file creation is idential to the `Credential File` option.

Setting the `credentials` parameter will override the `credentials_file` parameter.

The `credentials` parameter can be set by embedding the credential json:

```yaml
exporters:
  googlecloud:
    # this is a fake credential json
    credentials: '{   "type": "service_account",   "project_id": "myproject",   "private_key_id": "bt47a39d576b495709711c0536348edb41baf7cb",   "private_key": "-----BEGIN PRIVATE KEY-----\nFIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQDmjTOtSX18SBQw\nyIJ2Y9jpqXyBNeV4ghANRtckuc5bAnqAVsTaqGUD60zGpxa5mMSdPhFDRRw5Seeg\n8QR0TCZXDX3DaJ2pMeO8qiy5DyvllseS1vWjvabEdT0LLIB+dCbaqbVeaRRwOlFH\noqUQiND7WeVqu7m/d0USwjHIBoFZFg9j2Q23UBFPLJJ/7FwuZgiIYwzfJzFDXHG0\nK2JNSnksSjZ14wnXORU0GCRHMnAutoi1P5uZQPbaepQm+EwwZnRgcO54SQBRSGod\nK2iu81SprThEtQf5uGkO/0D1+b2A/81EJvPf1ye92Oi0z3ZebzBSX1cSbPCJDstY\nBeU+r+vpAgMBAAECggEAIu2r+X/90/tzm6R1J4yfC19is0zAFE7YMbq+06CW4/Dv\nMEx1DI+gpkRr2EjuR5znhM8nfGKXERLiVd7OBvSWhm/T0uwhnsWhNC8wEWV8CA+c\n8uFAW+tavb7pXa0DVqUwUcaOZNxUrWAYelro2SVxS/Nlr5L7ZEeknl/vfNeHd0B3\nt9bt06m/G0M/2ySA7jIxV0Fg0Z0IQonVowzMtUzbE2ZGgHyPIbdpYClp+EojA61m\nih9+VsFUzAK9KnFaRzNnoJHeLMKPHG1pCUaBt4qaWiZhQn+kRUvMgTga1ekeCzwT\nnBWqgnNGS/C6Quhpl+o7T04A5X6dHNXY/K1i9bqG0QKBgQD4838OBZ97oOtQLKlD\nusuS3r5QReUPN0X7FkDV9FGNe5Q9WUgrobOyXAaf5HmPYF7tPjxiP381KouovuYE\n1j0W8J9+vy3WhVRgo7ZyrG45atz/1AdM2PFcyCwQF0zf6wXsVG6Sqk+VedBnjkqt\nWG1tCSNIeq953E8X/GpPTc1c5QKBgQDtEy5A6PQxncvQbm2cIibIcMX4Gy7ZvJ82\nUR98sdT+w15j/Riy/VNj8BqJrcYfcBgs5MUtlk3SXC0T9WaYtZIZp14qvLj8FWvq\nkZNpNRKA43iMKS9L+pdNiHqMHZzoMwsbErNjZc0QL0b4vd+oWQ8CcNRNACySqeNP\nxSdqpfJmtQKBgANYup+EodU2n5MvVoMrkqsBxYsstVyUAKPUc8CsjSAaxi5g8eBs\nRw8hv5EMsDmmMQB9crBbbClZzhDRqCPugVm6mFpK1aHpnu3BpaU6/ixVbG0f+40j\n6XK22ijJN2ZXMXgw1l+wXGuE/LE3r3dPFgF+OvQxegRoWsPWx9MTF6ylAoGBAKSl\nnIrx/p3y1BjmiHNV+I9eWu8rmccYS46CmpaUPrPMZWKV5TBx5RdUKmoR6LXuuKt9\nGj/F0jhVUe05kk5eU6BDb4/Iz8Qq8G7ROYpolHg1AoR9Gd7vo2LydQGYk19kC8N6\nomFW0yr5WpXn8EvPxi/QwnDTvSECod7FstFLfOS9AoGANBIZghBf8HwphoHEV6q1\n+OkiCJF7Daf16ZGm6HBbDUzd6prC/lNzGFJcY97uNQy4C6p3v/OJzibvQAHb68ap\nEtZ+7M1TJN8x78BmX2NwGoC0Yjg42gt0nvulRnTrSZvnr2vyjkexhQ0lwkeuMYE5\nQGfwz9DP01LHNFF2711tMWY=\n-----END PRIVATE KEY-----\n",   "client_email": "serviceaccount@myproject.iam.gserviceaccount.com",   "client_id": "006890467469331372107",   "auth_uri": "https://accounts.google.com/o/oauth2/auth",   "token_uri": "https://oauth2.googleapis.com/token",   "auth_provider_x509_cert_url": "https://www.googleapis.com/oauth2/v1/certs",   "client_x509_cert_url": "https://www.googleapis.com/robot/v1/metadata/x509/serviceaccount%40myproject.iam.gserviceaccount.com" }'
```
