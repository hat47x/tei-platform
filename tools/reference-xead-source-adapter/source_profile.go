package main

import (
	"encoding/json"
	"fmt"
)

type SourceProfileContract struct {
	RootKind                string   `json:"rootKind"`
	SupportedFormatVersions []string `json:"supportedFormatVersions"`
}

type ObservedSourceProfile struct {
	RootKind      string `json:"rootKind"`
	FormatVersion string `json:"formatVersion"`
}

func parseSourceProfileContract(format SourceFormat, contractBytes []byte) (SourceProfileContract, error) {
	var contract struct {
		SourceProfiles map[string]SourceProfileContract `json:"sourceProfiles"`
	}
	if err := json.Unmarshal(contractBytes, &contract); err != nil {
		return SourceProfileContract{}, fmt.Errorf("parse transcription contract source profile: %w", err)
	}
	profile, ok := contract.SourceProfiles[string(format)]
	if !ok {
		return SourceProfileContract{}, fmt.Errorf("transcription contract has no %s source profile", format)
	}
	if profile.RootKind == "" {
		return SourceProfileContract{}, fmt.Errorf("transcription contract source profile %s has no root kind", format)
	}
	if len(profile.SupportedFormatVersions) == 0 {
		return SourceProfileContract{}, fmt.Errorf("transcription contract source profile %s has no supported format versions", format)
	}
	return profile, nil
}

func observedSourceProfile(doc ObservedDocument) (ObservedSourceProfile, error) {
	if len(doc.Nodes) == 0 {
		return ObservedSourceProfile{}, fmt.Errorf("observed document has no root element")
	}
	root := doc.Nodes[0]
	if root.Namespace != "" || !root.Known {
		return ObservedSourceProfile{}, fmt.Errorf("observed document root is not a known unqualified source construct")
	}
	formatVersion := root.Attributes["FormatVersion"]
	if formatVersion == "" {
		return ObservedSourceProfile{}, fmt.Errorf("observed document root %s has no FormatVersion", root.Kind)
	}
	return ObservedSourceProfile{
		RootKind:      root.Kind,
		FormatVersion: formatVersion,
	}, nil
}

func validateSourceProfile(format SourceFormat, doc ObservedDocument, contractBytes []byte) (ObservedSourceProfile, error) {
	if format != doc.Format {
		return ObservedSourceProfile{}, fmt.Errorf("observed document format %s does not match requested format %s", doc.Format, format)
	}
	expected, err := parseSourceProfileContract(format, contractBytes)
	if err != nil {
		return ObservedSourceProfile{}, err
	}
	observed, err := observedSourceProfile(doc)
	if err != nil {
		return ObservedSourceProfile{}, err
	}
	if observed.RootKind != expected.RootKind {
		return ObservedSourceProfile{}, fmt.Errorf(
			"observed document root %s does not match %s source profile root %s",
			observed.RootKind,
			format,
			expected.RootKind,
		)
	}
	for _, version := range expected.SupportedFormatVersions {
		if observed.FormatVersion == version {
			return observed, nil
		}
	}
	return ObservedSourceProfile{}, fmt.Errorf(
		"observed %s FormatVersion %s is not supported by transcription contract",
		format,
		observed.FormatVersion,
	)
}
