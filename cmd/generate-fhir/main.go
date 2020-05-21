package main

import (
	"divoc/pkg/azure/auth"
	"divoc/pkg/azure/azcopy"
	"divoc/pkg/logger"
	"divoc/pkg/synthea"
	"flag"
	"fmt"
	"path"
)

func main() {
	////////////////////////////////////////////////////////////////////////////////
	// Parse command line args
	////////////////////////////////////////////////////////////////////////////////
	// Synthea flags
	populationPtr := flag.Int("population", 100, "Sample size for Synthea to generate")
	state := flag.String("state", "California", "State which the sample size will be generated in")
	city := flag.String("city", "San Francisco", "City which the sample size will be generated in")
	csv := flag.Bool("csv", false, "Generate CSV output in addition to FHIR")
	ndjson := flag.Bool("ndjson", false, "Generate bulk FHIR dumps in NDJSON format (standard JSON will not be generated)")

	// azcopy flags
	noClean := flag.Bool("no-clean", false, "Don't cleanup temporary directories after running -- useful if you want to generated output locally")
	spClientId := flag.String("sp-client-id", "", "Service principal client ID to authenticate with azcopy -- The principal must have 'Storage Blob Data Contributor' role on the target storage account")
	spClientSecret := flag.String("sp-client-secret", "", "Service principal client secret to authenticate with azcopy")
	spTenantId := flag.String("sp-tenant-id", "", "Service principal tenant ID to authenticate with azcopy")
	storageAccount := flag.String("storage-account", "", "Azure storage account name to push FHIR data to")
	storageContainer := flag.String("storage-container", "", "Azure storage blob container name to push FHIR data to")

	flag.Parse()

	// Validate flags
	if *spClientId == "" {
		logger.Fatal("sp-client-id required")
	}
	if *spClientSecret == "" {
		logger.Fatal("sp-client-secret required")
	}
	if *spTenantId == "" {
		logger.Fatal("sp-tenant-id required")
	}
	if *storageAccount == "" {
		logger.Fatal("storage-account required")
	}
	if *storageContainer == "" {
		logger.Fatal("storage-container required")
	}

	////////////////////////////////////////////////////////////////////////////////
	// Generate data FHIR data with Synthea
	////////////////////////////////////////////////////////////////////////////////
	if err := synthea.Clone(); err != nil {
		logger.Fatal(err)
	}
	// only clean if -no-clean was passed
	if *noClean == false {
		defer synthea.Clean()
	}

	var options = synthea.Options{
		"exporter.fhir.bulk_data": "false",
		"exporter.csv.export":     "false",
	}
	if *csv {
		options["exporter.csv.export"] = "true"
	}
	if *ndjson {
		options["exporter.fhir.bulk_data"] = "true"
	}
	if err := synthea.SetOptions(options); err != nil {
		logger.Fatal(err)
	}

	var syntheaArgs = synthea.CliArgs{
		PopulationSize: *populationPtr,
		State:          *state,
		City:           *city,
	}
	if err := synthea.Run(syntheaArgs); err != nil {
		logger.Fatal(err)
	}
	installPath, err := synthea.GetInstallPath()
	if err != nil {
		logger.Fatal(err)
	}
	syntheaOut := path.Join(installPath, "output")
	logger.Info(fmt.Sprintf("Completed generating FHIR data at: %s", syntheaOut))

	////////////////////////////////////////////////////////////////////////////////
	// Copy data to Azure storage
	////////////////////////////////////////////////////////////////////////////////
	targetBlob := fmt.Sprintf("https://%s.blob.core.windows.net/%s", *storageAccount, *storageContainer)
	logger.Info(fmt.Sprintf("Beginning data migration from %s to %s", syntheaOut, targetBlob))

	// Login to azcopy with provided SP
	sp := auth.ServicePrincipal{
		ApplicationId: *spClientId,
		Password:      *spClientSecret,
		Tenant:        *spTenantId,
	}
	if err = azcopy.Login(sp); err != nil {
		logger.Fatal(err)
	}

	// Copy the contents of the synthea output directory to azure storage
	if err = azcopy.Copy(path.Join(syntheaOut, "*"), targetBlob); err != nil {
		logger.Fatal(err)
	}
	logger.Info("Transfer complete!")
}
