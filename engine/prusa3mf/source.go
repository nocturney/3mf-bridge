package prusa3mf

import (
	"sort"
	"strings"
)

// readSource validates the package and extracts only the facts needed for
// conversion. Everything else (slice caches, G-code, firmware scripts,
// thumbnails) is listed as dropped; its content is never read or copied.
func readSource(path string) (*project, error) {
	a, err := openArchive(path)
	if err != nil {
		return nil, err
	}
	defer a.close()

	modelName, err := a.modelPath()
	if err != nil {
		return nil, err
	}
	for _, name := range a.names() {
		if name != modelName && strings.HasSuffix(strings.ToLower(name), ".model") {
			return nil, fail("UNSUPPORTED_MULTI_MODEL", "package has an extra model part %q; only one 3D model is supported", name)
		}
	}
	data, err := a.read(modelName, maxModelBytes)
	if err != nil {
		return nil, err
	}
	doc, err := parseModel(data, false)
	if err != nil {
		return nil, err
	}

	settings := make(map[int]objectSettings)
	if a.has(modelSettingsPart) {
		data, err := a.read(modelSettingsPart, maxConfigBytes)
		if err != nil {
			return nil, err
		}
		if settings, err = parseModelSettings(data); err != nil {
			return nil, err
		}
	}
	var slice []filamentEntry
	if a.has(sliceInfoPart) {
		data, err := a.read(sliceInfoPart, maxConfigBytes)
		if err != nil {
			return nil, err
		}
		if slice, err = parseSliceInfo(data); err != nil {
			return nil, err
		}
	}
	var ps projectSettings
	if a.has(projectSettingsPart) {
		data, err := a.read(projectSettingsPart, maxConfigBytes)
		if err != nil {
			return nil, err
		}
		if ps, err = parseProjectSettings(data); err != nil {
			return nil, err
		}
	}
	if strings.ContainsAny(ps.printerModel, "\r\n") {
		return nil, fail("SOURCE_INVALID", "printer_model contains line breaks")
	}
	channels, err := buildChannels(slice, ps)
	if err != nil {
		return nil, err
	}
	p := &project{
		unit:          doc.unit,
		objects:       doc.objects,
		items:         doc.items,
		channels:      channels,
		paints:        doc.paints,
		sourcePrinter: ps.printerModel,
		dropped:       a.droppedNames(modelName),
	}
	if err := p.bind(doc, settings); err != nil {
		return nil, err
	}
	return p, nil
}

// modelPath resolves the 3D model part from the package relationships, falling
// back to the conventional path when no relationship is declared.
func (a *archive) modelPath() (string, error) {
	var targets []string
	if a.has(relsPart) {
		data, err := a.read(relsPart, maxConfigBytes)
		if err != nil {
			return "", err
		}
		root, err := parseSmallXML(data)
		if err != nil {
			return "", err
		}
		for _, rel := range root.childrenNamed("Relationship") {
			if rel.attrs["Type"] == modelRelType {
				targets = append(targets, strings.TrimPrefix(rel.attrs["Target"], "/"))
			}
		}
	}
	if len(targets) > 1 {
		return "", fail("UNSUPPORTED_MULTI_MODEL", "package declares %d 3D model relationships; only one is supported", len(targets))
	}
	if len(targets) == 0 {
		targets = append(targets, modelPart)
	}
	if !a.has(targets[0]) {
		return "", fail("SOURCE_INVALID", "3D model part %q is missing", targets[0])
	}
	return targets[0], nil
}

// droppedNames lists every entry that is not carried into the output.
func (a *archive) droppedNames(modelName string) []string {
	keep := map[string]bool{
		contentTypesPart:    true,
		relsPart:            true,
		modelName:           true,
		modelSettingsPart:   true,
		sliceInfoPart:       true,
		projectSettingsPart: true,
	}
	var out []string
	for name := range a.files {
		if !keep[name] {
			out = append(out, name)
		}
	}
	sort.Strings(out)
	return out
}
