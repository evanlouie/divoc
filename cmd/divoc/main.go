package main

import (
	"divoc/pkg/azcopy"
	"divoc/pkg/logger"
	"divoc/pkg/synthea"
	"flag"
	"fmt"
	"os/exec"
	"path"
)

func main() {
	// Parse command line args
	populationPtr := flag.Int("population", 100, "Sample size for Synthea to generate")
	state := flag.String("state", "California", "State which the sample size will be generated in")
	city := flag.String("city", "San Francisco", "City which the sample size will be generated in")
	csv := flag.Bool("csv", false, "Generate CSV output in addition to FHIR")
	ndjson := flag.Bool("ndjson", false, "Generate bulk FHIR dumps in NDJSON format (standard JSON will not be generated)")
	noClean := flag.Bool("no-clean", false, "Don't cleanup temporary directories after running -- useful if you want to generated output locally")
	spClientId := flag.String("sp-client-id", "", "Service principal client ID to authenticate with azcopy")
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

	// Check for all required host dependencies
	logger.Info("Checking for required host dependencies...")
	var requiredSystemTools = []string{"git", "java", "azcopy"}
	for _, tool := range requiredSystemTools {
		lookPath, err := exec.LookPath(tool)
		if err != nil {
			logger.Fatal(err)
		} else {
			logger.Info(fmt.Sprintf("Found %s at %s", tool, lookPath))
		}
	}
	logger.Info("All host dependencies found!")

	if err := synthea.Clone(); err != nil {
		logger.Fatal(err)
	}
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
	logger.Info(fmt.Sprintf("Completed generating fhir data at: %s", syntheaOut))

	// Copy data to Azure storage
	targetBlob := fmt.Sprintf("https://%s.blob.core.windows.net/%s", *storageAccount, *storageContainer)
	logger.Info(fmt.Sprintf("Beginning data migration from %s to %s", syntheaOut, targetBlob))
	err = azcopy.Login(azcopy.ServicePrincipal{
		ApplicationId: *spClientId,
		Password:      *spClientSecret,
		Tenant:        *spTenantId,
	})
	if err != nil {
		logger.Fatal(err)
	}
	if err = azcopy.Copy(fmt.Sprintf("%s/*", syntheaOut), targetBlob); err != nil {
		logger.Fatal(err)
	}
	logger.Info("Copying complete!")
}
