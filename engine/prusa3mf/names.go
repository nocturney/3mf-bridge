package prusa3mf

// Part names inside a 3MF package. The first group is read from sources; the
// second group is written to output.
const (
	contentTypesPart     = "[Content_Types].xml"
	relsPart             = "_rels/.rels"
	modelPart            = "3D/3dmodel.model"
	modelSettingsPart    = "Metadata/model_settings.config"
	sliceInfoPart        = "Metadata/slice_info.config"
	projectSettingsPart  = "Metadata/project_settings.config"
	modelRelType         = "http://schemas.microsoft.com/3dmanufacturing/2013/01/3dmodel"
	prusaConfigPart      = "Metadata/Slic3r_PE.config"
	prusaModelConfigPart = "Metadata/Slic3r_PE_model.config"
)
