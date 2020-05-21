package auth

type ServicePrincipal struct {
	ApplicationId string
	Password      string
	Tenant        string
	DisplayName   string // no usage as of yet
	Name          string // no usage as of yet
}
