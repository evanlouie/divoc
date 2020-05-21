# divoc

## Requirements

| Dependency | Version  | Description                                                                             |
| ---------- | -------- | --------------------------------------------------------------------------------------- |
| Git        |          | Used to clone the [Synthea](https://github.com/synthetichealth/synthea) project locally |
| Go         | `>=1.14` | The core runtime for this project                                                       |
| Java       | `>=1.8`  | Used to run [Synthea](https://github.com/synthetichealth/synthea)                       |
| azcopy     | `>=10`   | Used to migrate generate FHIR files from host to Azure storage                          |

## Sample Usage

```shell script
SP_CLIENT_ID=<your service princpal client ID>
SP_CLIENT_SECRET=<your service princpal client secret>
SP_TENANT_ID=<your service princpal tenant ID>
go run cmd/divoc/main.go \
    -csv \
    -ndjson \
    -population 10 \
    -sp-client-id $SP_CLIENT_ID \
    -sp-client-secret $SP_CLIENT_SECRET \
    -sp-tenant-id $SP_TENANT_ID \
    -storage-account evlouiecovid \
    -storage-container synthea
```
