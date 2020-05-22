# divoc

## Requirements

| Dependency | Version           | Description                                                                             |
| ---------- | ----------------- | --------------------------------------------------------------------------------------- |
| Git        |                   | Used to clone the [Synthea](https://github.com/synthetichealth/synthea) project locally |
| Go         | `>=1.14`          | The core runtime for this project                                                       |
| Java       | `>=1.8` & `<1.14` | Used to run [Synthea](https://github.com/synthetichealth/synthea)                       |
| azcopy     | `>=10`            | Used to migrate generate FHIR files from host to Azure storage                          |

## Commands

### `generate-fhir`

Utilizing the [Synthea](https://github.com/synthetichealth/synthea) project,
this command will generate a sample `FHIR` dataset and upload it to a target
storage account container using `azcopy`.

### Sample Usage

> Run `go run cmd/generate-fhir/main.go --help` to view descriptions of all
> available flags.

#### Basic

```shell script
SP_CLIENT_ID=<your service princpal client ID>
SP_CLIENT_SECRET=<your service princpal client secret>
SP_TENANT_ID=<your service princpal tenant ID>
STORAGE_ACCOUNT=<your storage account name>
STORAGE_CONTAINER=<your target storage container (can contain a target subdirectory)>

go run cmd/generate-fhir/main.go \
    -synthea-csv \
    -synthea-ndjson \
    -synthea-population 10 \
    -sp-client-id $SP_CLIENT_ID \
    -sp-client-secret $SP_CLIENT_SECRET \
    -sp-tenant-id $SP_TENANT_ID \
    -storage-account $STORAGE_ACCOUNT \
    -storage-container $STORAGE_CONTAINER
```
