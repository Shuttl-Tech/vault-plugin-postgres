package backend

import "github.com/hashicorp/vault/sdk/framework"

type operationHandler struct {
	fn    framework.OperationFunc
	props framework.OperationProperties
}

func (h *operationHandler) Handler() framework.OperationFunc {
	return h.fn
}

func (h *operationHandler) Properties() framework.OperationProperties {
	return h.props
}

func NewOperationHandler(fn framework.OperationFunc, props framework.OperationProperties) framework.OperationHandler {
	return &operationHandler{
		fn:    fn,
		props: props,
	}
}

var propsPathInfo = framework.OperationProperties{
	Summary:     helpSynopsisInfo,
	Description: helpDescriptionInfo,
}

var propsMetadataUpdate = framework.OperationProperties{
	Summary:     helpSynopsisMetadata,
	Description: helpDescriptionMetadata,
}

var propsMetadataRead = propsMetadataUpdate

var propsMetadataList = propsMetadataUpdate

var propsMetadataDelete = propsMetadataUpdate

var propsClustersList = framework.OperationProperties{
	Summary:     helpSynopsisListClusters,
	Description: helpDescriptionListClusters,
}

var propsClusterRead = propsClustersList

var propsClusterUpdate = propsClustersList

var propsClusterDelete = propsClustersList

var propsDatabasesList = propsClustersList

var propsCloneUpdate = framework.OperationProperties{
	Summary:     helpSynopsisClone,
	Description: helpDescriptionClone,
}

var propsDatabaseUpdate = framework.OperationProperties{
	Summary:     helpSynopsisDatabase,
	Description: helpDescriptionDatabase,
}

var propsDatabaseRead = propsDatabaseUpdate

var propsDatabaseDelete = propsDatabaseUpdate

var propsRoleList = framework.OperationProperties{
	Summary:     helpSynopsisListRoles,
	Description: helpDescriptionListRoles,
}

var propsRoleUpdate = framework.OperationProperties{
	Summary:     helpSynopsisRoles,
	Description: helpDescriptionRoles,
}

var propsRoleRead = propsRoleUpdate

var propsRoleDelete = propsRoleUpdate

var propsCredsRead = framework.OperationProperties{
	Summary:     helpSynopsisCreds,
	Description: helpDescriptionCreds,
}

var propsGcListClusters = framework.OperationProperties{
	Summary:     helpSynopsisGCListClusters,
	Description: helpDescriptionGCListClusters,
}

var propsListDatabases = framework.OperationProperties{
	Summary:     helpSynopsisGCClusterOps,
	Description: helpDescriptionGCClusterOps,
}

var propsGcGetCluster = propsListDatabases

var propsGcPurgeCluster = propsListDatabases

var propsGcGetDatabase = framework.OperationProperties{
	Summary:     helpSynopsisGCDbOps,
	Description: helpDescriptionGCDbOps,
}

var propsGcPurgeDatabase = propsGcGetDatabase
