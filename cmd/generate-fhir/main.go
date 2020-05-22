package main

import (
	"flag"
	"fmt"
	"microsoft.com/divoc/pkg/azure/auth"
	"microsoft.com/divoc/pkg/azure/azcopy"
	"microsoft.com/divoc/pkg/logger"
	"microsoft.com/divoc/pkg/synthea"
	"path"
	"path/filepath"
)

func main() {
	////////////////////////////////////////////////////////////////////////////////
	// Parse command line args
	////////////////////////////////////////////////////////////////////////////////
	// Synthea flags
	population := flag.Int("synthea-population", 100, "Sample size for Synthea to generate")
	state := flag.String("synthea-state", "California", "State which the sample size will be generated in")
	city := flag.String("synthea-city", "San Francisco", "City which the sample size will be generated in")
	csv := flag.Bool("synthea-csv", false, "Generate CSV output in addition to FHIR")
	ndjson := flag.Bool("synthea-ndjson", false, "Generate bulk FHIR dumps in NDJSON format (standard JSON will not be generated)")
	noClean := flag.Bool("synthea-no-clean", false, "Do not cleanup temporary directories after running -- useful if you want to generated output locally")
	syntheaPath := flag.String("synthea-path", "", "Path to local Synthea repository -- if provided, will skip cloning the repo locally and force -synthea-no-clean, if not, will clone the repository to a temporary directory")

	// azcopy flags
	spClientId := flag.String("sp-client-id", "", "Service principal client ID to authenticate with AzCopy -- The principal must have 'Storage Blob Data Contributor' role on the target storage account")
	spClientSecret := flag.String("sp-client-secret", "", "Service principal client secret to authenticate with AzCopy")
	spTenantId := flag.String("sp-tenant-id", "", "Service principal tenant ID to authenticate with AzCopy")
	storageAccount := flag.String("storage-account", "", "Azure storage account name to push FHIR data to")
	storageContainer := flag.String("storage-container", "", "Azure storage blob container name to push FHIR data to")

	flag.Parse()

	// Validate flags
	if *spClientId == "" {
		logger.Fatal("-sp-client-id required")
	}
	if *spClientSecret == "" {
		logger.Fatal("-sp-client-secret required")
	}
	if *spTenantId == "" {
		logger.Fatal("-sp-tenant-id required")
	}
	if *storageAccount == "" {
		logger.Fatal("-storage-account required")
	}
	if *storageContainer == "" {
		logger.Fatal("-storage-container required")
	}
	// if -synthea-path provided:
	// - set -synthea-no-clean to true
	// - calculate absolute version
	if *syntheaPath != "" {
		logger.Info("-synthea-path provided: forcing -synthea-no-clean")
		*noClean = true
		absSyntheaPath, err := filepath.Abs(*syntheaPath)
		if err != nil {
			logger.Error(err)
			logger.Fatalf("Failed to calculate absolute path for -synthia-path: %s", *syntheaPath)
		}
		syntheaPath = &absSyntheaPath
		logger.Infof("Using local Synthea repo at: %s", *syntheaPath)
	}

	////////////////////////////////////////////////////////////////////////////////
	// Generate data FHIR data with Synthea
	////////////////////////////////////////////////////////////////////////////////
	// If -synthea-path provided, set installPath in synthea package and skip cloning
	if *syntheaPath != "" {
		synthea.InstallPath = syntheaPath
	} else if err := synthea.Clone(); err != nil {
		logger.Error(err)
		logger.Fatal("Failed cloning the Synthea repository")
	}

	// clean up if no-clean set to false
	if *noClean == false {
		defer synthea.Clean()
	}

	options := synthea.Options{
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

	syntheaArgs := synthea.CliArgs{
		PopulationSize: *population,
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
	logger.Infof("Completed generating FHIR data at: %s", syntheaOut)

	////////////////////////////////////////////////////////////////////////////////
	// Copy data to Azure storage
	////////////////////////////////////////////////////////////////////////////////
	azc, err := azcopy.InstallIfNotPresentAndGetCtx()
	if err != nil {
		logger.Error(err)
		logger.Fatal("Failed to install AzCopy")
	}
	defer azc.Cleanup()

	// Login to azcopy with provided SP
	logger.Info("Logging into AzCopy with provided service principal credentials...")
	sp := auth.ServicePrincipal{
		ApplicationId: *spClientId,
		Password:      *spClientSecret,
		Tenant:        *spTenantId,
	}
	if err = azc.Login(sp); err != nil {
		logger.Error(err)
		logger.Fatal("Failed to authenticate AzCopy with provided service principal credentials")
	}
	logger.Info("Login complete!")

	// Copy the contents of the synthea output directory to azure storage
	targetBlob := fmt.Sprintf("https://%s.blob.core.windows.net/%s", *storageAccount, *storageContainer)
	logger.Infof("Beginning data copy from %s to %s", syntheaOut, targetBlob)
	if err = azc.Copy(path.Join(syntheaOut, "*"), targetBlob); err != nil {
		logger.Error(err)
		logger.Fatalf("Failed copying files from %s to %s", syntheaOut, targetBlob)
	}
	logger.Info("Transfer complete!")
}
