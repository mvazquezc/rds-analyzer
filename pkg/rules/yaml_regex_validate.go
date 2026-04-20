package rules

import (
	"fmt"
	"path/filepath"
	"regexp"

	"gopkg.in/yaml.v3"
)

// validateRegexPatternsFromYAML walks the rules YAML AST and validates every non-empty
// regex and value_regex with regexp.Compile. Source line numbers come from yaml.v3 nodes.
func validateRegexPatternsFromYAML(data []byte, rulesPath string) []string {
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil
	}
	base := filepath.Base(rulesPath)
	if len(doc.Content) == 0 {
		return nil
	}
	root := doc.Content[0]
	if root.Kind != yaml.MappingNode {
		return nil
	}

	var warnings []string
	if gr := yamlMappingGet(root, "global_rules"); gr != nil {
		warnings = append(warnings, validateConditionRegexesInRuleList(gr, base, "global rule")...)
	}
	if rr := yamlMappingGet(root, "rules"); rr != nil {
		warnings = append(warnings, validateConditionRegexesInRuleList(rr, base, "rule")...)
	}
	if lar := yamlMappingGet(root, "label_annotation_rules"); lar != nil && lar.Kind == yaml.MappingNode {
		if labels := yamlMappingGet(lar, "labels"); labels != nil {
			warnings = append(warnings, validateValueRegexInLabelList(labels, base, "label")...)
		}
		if anns := yamlMappingGet(lar, "annotations"); anns != nil {
			warnings = append(warnings, validateValueRegexInLabelList(anns, base, "annotation")...)
		}
	}
	return warnings
}

func yamlMappingGet(m *yaml.Node, key string) *yaml.Node {
	if m == nil || m.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i < len(m.Content); i += 2 {
		k := m.Content[i]
		if k.Kind == yaml.ScalarNode && k.Value == key {
			return m.Content[i+1]
		}
	}
	return nil
}

func validateConditionRegexesInRuleList(ruleList *yaml.Node, base string, ruleKind string) []string {
	if ruleList == nil || ruleList.Kind != yaml.SequenceNode {
		return nil
	}
	var warnings []string
	for _, ruleNode := range ruleList.Content {
		if ruleNode == nil || ruleNode.Kind != yaml.MappingNode {
			continue
		}
		ruleID := ""
		if idn := yamlMappingGet(ruleNode, "id"); idn != nil && idn.Kind == yaml.ScalarNode {
			ruleID = idn.Value
		}
		conds := yamlMappingGet(ruleNode, "conditions")
		if conds == nil || conds.Kind != yaml.SequenceNode {
			continue
		}
		for i, condNode := range conds.Content {
			if condNode == nil || condNode.Kind != yaml.MappingNode {
				continue
			}
			rx := yamlMappingGet(condNode, "regex")
			if rx == nil || rx.Kind != yaml.ScalarNode || rx.Value == "" {
				continue
			}
			if _, err := regexp.Compile(rx.Value); err != nil {
				warnings = append(warnings, fmt.Sprintf(
					"[%s] line %s: %s %q condition[%d]: invalid regex %q - %v",
					base, formatYAMLLine(rx), ruleKind, ruleID, i, rx.Value, err,
				))
			}
		}
	}
	return warnings
}

func validateValueRegexInLabelList(list *yaml.Node, base string, laKind string) []string {
	if list == nil || list.Kind != yaml.SequenceNode {
		return nil
	}
	var warnings []string
	for i, item := range list.Content {
		if item == nil || item.Kind != yaml.MappingNode {
			continue
		}
		keyStr := ""
		if kn := yamlMappingGet(item, "key"); kn != nil && kn.Kind == yaml.ScalarNode {
			keyStr = kn.Value
		}
		vr := yamlMappingGet(item, "value_regex")
		if vr == nil || vr.Kind != yaml.ScalarNode || vr.Value == "" {
			continue
		}
		if _, err := regexp.Compile(vr.Value); err != nil {
			warnings = append(warnings, fmt.Sprintf(
				"[%s] line %s: %s rule[%d] (key=%q): invalid value_regex %q - %v",
				base, formatYAMLLine(vr), laKind, i, keyStr, vr.Value, err,
			))
		}
	}
	return warnings
}

func formatYAMLLine(n *yaml.Node) string {
	if n == nil || n.Line < 1 {
		return "?"
	}
	return fmt.Sprintf("%d", n.Line)
}
